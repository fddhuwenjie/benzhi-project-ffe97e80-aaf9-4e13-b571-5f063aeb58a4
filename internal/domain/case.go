package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (c *ReleaseCase) EnsureMutable() error {
	if c.Status == StatusSealed {
		return NewRuleError("case_sealed", "个案已封存，不能再修改")
	}
	return nil
}

func (c *ReleaseCase) AddConsent(grant ConsentGrant, now time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusDraft, StatusConsentReady); err != nil {
		return err
	}
	NormalizeConsent(&grant)
	if err := ValidateConsent(grant, now); err != nil {
		return err
	}
	for _, existing := range c.Consents {
		if existing.ParticipantCode == grant.ParticipantCode && existing.WithdrawnAt == nil {
			return NewRuleError("duplicate_participant", "参与者有效授权已存在")
		}
	}
	for index := range c.Consents {
		if c.Consents[index].ParticipantCode == grant.ParticipantCode {
			c.Consents[index] = grant
			c.Status = StatusConsentReady
			return nil
		}
	}
	c.Consents = append(c.Consents, grant)
	c.Status = StatusConsentReady
	return nil
}

func (c *ReleaseCase) WithdrawConsent(consentID string, at time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	for index := range c.Consents {
		if c.Consents[index].ID != consentID {
			continue
		}
		if c.Consents[index].WithdrawnAt != nil {
			return NewRuleError("consent_withdrawn", "授权已经撤回")
		}
		c.Consents[index].WithdrawnAt = &at
		c.Status = StatusDraft
		return nil
	}
	return NewRuleError("consent_not_found", "参与者授权不存在")
}

func (c *ReleaseCase) consentFor(participant, use string, now time.Time) *ConsentGrant {
	for i := range c.Consents {
		grant := &c.Consents[i]
		if grant.ParticipantCode != participant || grant.WithdrawnAt != nil {
			continue
		}
		if grant.ValidUntil != nil && !grant.ValidUntil.After(now) {
			continue
		}
		for _, permitted := range grant.PermittedUses {
			if permitted == use || permitted == "ALL" {
				return grant
			}
		}
	}
	return nil
}

func (c *ReleaseCase) AddAsset(asset RecordingAsset, now time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusConsentReady, StatusMaterialRegistered); err != nil {
		return err
	}
	if err := ValidateAsset(asset); err != nil {
		return err
	}
	for _, existing := range c.Assets {
		if existing.StableKey == asset.StableKey {
			return NewRuleError("duplicate_stable_key", "材料 stable_key 已存在")
		}
	}
	seen := map[string]bool{}
	for _, participant := range asset.ParticipantCodes {
		if seen[participant] {
			return NewRuleError("validation_error", "participant_codes 不得重复")
		}
		seen[participant] = true
		if c.consentFor(participant, c.ReleaseLevel, now) == nil {
			return NewRuleError("consent_not_covered", fmt.Sprintf("参与者 %s 未提供有效授权", participant))
		}
	}
	c.Assets = append(c.Assets, asset)
	c.Status = StatusMaterialRegistered
	return nil
}

func (c *ReleaseCase) AddAssets(assets []RecordingAsset, now time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusConsentReady, StatusMaterialRegistered); err != nil {
		return err
	}
	if len(assets) == 0 {
		return NewRuleError("validation_error", "assets 至少包含一项")
	}
	keys := map[string]bool{}
	hashes := map[string]bool{}
	for _, existing := range c.Assets {
		keys[existing.StableKey] = true
		hashes[existing.ContentSHA256] = true
	}
	for i := range assets {
		if err := ValidateAsset(assets[i]); err != nil {
			return err
		}
		if keys[assets[i].StableKey] {
			return NewRuleError("duplicate_stable_key", "材料 stable_key 已存在")
		}
		if hashes[assets[i].ContentSHA256] {
			return NewRuleError("duplicate_content_sha256", "材料内容摘要已存在")
		}
		keys[assets[i].StableKey], hashes[assets[i].ContentSHA256] = true, true
		seen := map[string]bool{}
		for _, participant := range assets[i].ParticipantCodes {
			if seen[participant] {
				return NewRuleError("validation_error", "participant_codes 不得重复")
			}
			seen[participant] = true
			if c.consentFor(participant, c.ReleaseLevel, now) == nil {
				return NewRuleError("consent_not_covered", fmt.Sprintf("参与者 %s 未提供有效授权", participant))
			}
		}
	}
	for i := range assets {
		assets[i].CaseID = c.ID
		c.Assets = append(c.Assets, assets[i])
	}
	c.Status = StatusMaterialRegistered
	return nil
}

func (c *ReleaseCase) AddFinding(assetID string, finding SensitivityFinding) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusMaterialRegistered, StatusRedactionReview); err != nil {
		return err
	}
	for i := range c.Assets {
		if c.Assets[i].ID != assetID {
			continue
		}
		if err := ValidateFinding(finding, c.Assets[i].DurationMS); err != nil {
			return err
		}
		for _, existing := range c.Assets[i].Findings {
			if finding.StartMS < existing.EndMS && existing.StartMS < finding.EndMS {
				return NewRuleError("finding_overlap", fmt.Sprintf("发现片段与 %s 重叠", existing.ID))
			}
		}
		c.Assets[i].Findings = append(c.Assets[i].Findings, finding)
		c.Status = StatusRedactionReview
		return nil
	}
	return NewRuleError("asset_not_found", "录音材料不存在")
}

func (c *ReleaseCase) CloseFinding(findingID, note string) error {
	return c.CloseFindingWithDisposition(findingID, "", note)
}

func (c *ReleaseCase) CloseFindingWithDisposition(findingID, disposition, note string) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusRedactionReview, StatusMaterialRegistered); err != nil {
		return err
	}
	if len(strings.TrimSpace(note)) < 8 {
		return NewRuleError("validation_error", "treatment_note 至少需要 8 个字符")
	}
	for i := range c.Assets {
		for j := range c.Assets[i].Findings {
			f := &c.Assets[i].Findings[j]
			if f.ID == findingID {
				if f.Status == "CLOSED" {
					return NewRuleError("finding_already_closed", "敏感性发现已经关闭")
				}
				if len(strings.TrimSpace(note)) < 8 {
					return NewRuleError("validation_error", "treatment_note 至少需要 8 个字符")
				}
				if disposition != "" {
					switch disposition {
					case "REMOVE", "MUTE", "VOICE_SHIFT", "RESTRICT_ACCESS":
						f.Disposition = disposition
					default:
						return NewRuleError("validation_error", "disposition 无效")
					}
				}
				f.Status = "CLOSED"
				f.TreatmentNote = note
				c.Status = StatusRedactionReview
				return nil
			}
		}
	}
	return NewRuleError("finding_not_found", "敏感性发现不存在")
}

func (c *ReleaseCase) ValidateAuthorizationCoverage(now time.Time) error {
	if len(c.Assets) == 0 {
		return NewRuleError("materials_required", "至少需要一份录音材料")
	}
	for _, asset := range c.Assets {
		for _, participant := range asset.ParticipantCodes {
			if c.consentFor(participant, c.ReleaseLevel, now) == nil {
				return NewRuleError("consent_not_covered", fmt.Sprintf("材料 %s 的参与者 %s 当前没有有效授权", asset.StableKey, participant))
			}
		}
	}
	return nil
}

func (c *ReleaseCase) RefreshAuthorizationSummary(now time.Time) {
	c.AssetCount = len(c.Assets)
	c.OpenFindingCount = 0
	for _, asset := range c.Assets {
		for _, finding := range asset.Findings {
			if finding.Status == "OPEN" {
				c.OpenFindingCount++
			}
		}
	}
	c.LatestReviewRound = len(c.Reviews)
	c.RemediationSummary = nil
	if len(c.Reviews) > 0 {
		last := c.Reviews[len(c.Reviews)-1]
		if last.Decision == "RETURN" {
			c.RemediationSummary = append([]string(nil), last.ReasonCodes...)
		}
	}
	for i := range c.Consents {
		grant := &c.Consents[i]
		grant.CoveredAssetCount = 0
		if grant.WithdrawnAt != nil {
			grant.Status = "WITHDRAWN"
		} else if grant.ValidUntil != nil && !grant.ValidUntil.After(now) {
			grant.Status = "EXPIRED"
		} else {
			grant.Status = "ACTIVE"
		}
		for _, asset := range c.Assets {
			for _, participant := range asset.ParticipantCodes {
				if participant == grant.ParticipantCode {
					grant.CoveredAssetCount++
					break
				}
			}
		}
	}
}

func (c *ReleaseCase) SubmitForReview(now time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusMaterialRegistered, StatusRedactionReview); err != nil {
		return err
	}
	if err := c.ValidateAuthorizationCoverage(now); err != nil {
		return err
	}
	if len(c.Reviews) > 0 {
		last := c.Reviews[len(c.Reviews)-1]
		if last.Decision == "RETURN" {
			open := 0
			for _, asset := range c.Assets {
				for _, finding := range asset.Findings {
					if finding.Status == "OPEN" {
						open++
					}
				}
			}
			if open > 0 {
				return NewRuleError("remediation_required", "退回复核待办尚未完成")
			}
		}
	}
	for _, asset := range c.Assets {
		for i := range asset.Findings {
			for j := i + 1; j < len(asset.Findings); j++ {
				if asset.Findings[i].StartMS < asset.Findings[j].EndMS && asset.Findings[j].StartMS < asset.Findings[i].EndMS {
					return NewRuleError("finding_overlap", fmt.Sprintf("材料 %s 存在重叠发现", asset.StableKey))
				}
			}
		}
		for _, finding := range asset.Findings {
			if finding.Status != "CLOSED" {
				return NewRuleError("open_findings", "所有敏感性发现关闭后才能送审")
			}
		}
	}
	c.Status = StatusStewardReview
	return nil
}

func (c *ReleaseCase) Review(review StewardReview) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if err := requireStatus(c.Status, StatusStewardReview); err != nil {
		return err
	}
	if strings.TrimSpace(review.Reviewer) == "" {
		return Required("reviewer")
	}
	switch review.Decision {
	case "APPROVE":
		if err := c.ValidateAuthorizationCoverage(review.DecidedAt); err != nil {
			return err
		}
		c.Status = StatusApproved
	case "RETURN":
		if len(review.ReasonCodes) == 0 || len(strings.TrimSpace(review.Comment)) < 4 {
			return NewRuleError("review_reason_required", "退回复核必须包含 reason_codes 和 comment")
		}
		for _, code := range review.ReasonCodes {
			switch code {
			case "AUDIO_LEAK", "IDENTITY_EXPOSURE", "CULTURAL_SENSITIVITY", "USE_SCOPE", "METADATA", "DETAIL", "MORE_REDACTION", "AUDIO_QUALITY", "CONSENT_SCOPE", "TIMESTAMP", "PARTICIPANT_REDACTION", "PRIVACY", "CONTENT", "QUALITY", "SCOPE":
			default:
				return NewRuleError("review_reason_required", "reason_codes 包含不支持的原因")
			}
		}
		c.Status = StatusRedactionReview
	default:
		return NewRuleError("validation_error", "decision 必须是 APPROVE 或 RETURN")
	}
	review.Round = len(c.Reviews) + 1
	review.FindingCount = 0
	for _, asset := range c.Assets {
		review.FindingCount += len(asset.Findings)
	}
	c.Reviews = append(c.Reviews, review)
	return nil
}

func (c *ReleaseCase) ManifestInput() ([]ManifestAsset, []string, error) {
	if err := requireStatus(c.Status, StatusApproved); err != nil {
		return nil, nil, err
	}
	assets := make([]ManifestAsset, 0, len(c.Assets))
	constraints := map[string]bool{}
	for _, grant := range c.Consents {
		for _, region := range grant.RegionLimits {
			constraints["region:"+region] = true
		}
		if grant.ValidUntil != nil {
			constraints["valid_until:"+grant.ValidUntil.UTC().Format(time.RFC3339)] = true
		}
	}
	for _, asset := range c.Assets {
		assets = append(assets, ManifestAsset{StableKey: asset.StableKey, Summary: asset.Summary, DurationMS: asset.DurationMS, CapturedOn: asset.CapturedOn, Participants: normalizedStrings(asset.ParticipantCodes), ContentSHA256: asset.ContentSHA256})
		for _, finding := range asset.Findings {
			if finding.Disposition == "RESTRICT_ACCESS" {
				constraints["restricted_asset:"+asset.StableKey] = true
			}
		}
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].StableKey < assets[j].StableKey })
	list := make([]string, 0, len(constraints))
	for value := range constraints {
		list = append(list, value)
	}
	sort.Strings(list)
	return assets, list, nil
}

func (c *ReleaseCase) Seal(at time.Time) error {
	if err := requireStatus(c.Status, StatusApproved); err != nil {
		return err
	}
	c.Status = StatusSealed
	c.SealedAt = &at
	return nil
}
