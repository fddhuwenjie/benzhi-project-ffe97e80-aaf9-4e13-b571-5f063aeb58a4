package domain

import (
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
	"time"
)

var stableKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,127}$`)

func ValidateNewCase(c *ReleaseCase) error {
	if strings.TrimSpace(c.LanguageName) == "" {
		return Required("language_name")
	}
	if strings.TrimSpace(c.CollectionBatch) == "" {
		return Required("collection_batch")
	}
	if strings.TrimSpace(c.Owner) == "" {
		return Required("owner")
	}
	switch c.ReleaseLevel {
	case "PUBLIC", "RESTRICTED", "COMMUNITY_ONLY":
	default:
		return NewRuleError("validation_error", "release_level 必须是 PUBLIC、RESTRICTED 或 COMMUNITY_ONLY")
	}
	return nil
}

func ValidateConsent(c ConsentGrant, now time.Time) error {
	if strings.TrimSpace(c.ParticipantCode) == "" {
		return Required("participant_code")
	}
	if strings.TrimSpace(c.EvidenceRef) == "" {
		return Required("evidence_ref")
	}
	if len(c.PermittedUses) == 0 {
		return NewRuleError("validation_error", "permitted_uses 至少包含一项")
	}
	for _, use := range c.PermittedUses {
		if strings.TrimSpace(use) == "" {
			return NewRuleError("validation_error", "permitted_uses 不得包含空值")
		}
	}
	for _, region := range c.RegionLimits {
		if strings.TrimSpace(region) == "" {
			return NewRuleError("validation_error", "region_limits 不得包含空值")
		}
	}
	if c.ValidUntil != nil && !c.ValidUntil.After(now) {
		return NewRuleError("consent_expired", "授权有效期必须晚于当前时间")
	}
	if c.WithdrawnAt != nil {
		return NewRuleError("consent_withdrawn", "不能登记已撤回的授权")
	}
	return nil
}

func NormalizeConsent(c *ConsentGrant) {
	c.ParticipantCode = strings.TrimSpace(c.ParticipantCode)
	c.EvidenceRef = strings.TrimSpace(c.EvidenceRef)
	c.PermittedUses = normalizedUnique(c.PermittedUses)
	c.RegionLimits = normalizedUnique(c.RegionLimits)
}

func ValidateAsset(a RecordingAsset) error {
	if !stableKeyPattern.MatchString(a.StableKey) {
		return NewRuleError("validation_error", "stable_key 格式无效")
	}
	if strings.TrimSpace(a.Summary) == "" {
		return Required("summary")
	}
	if a.DurationMS <= 0 {
		return NewRuleError("validation_error", "duration_ms 必须大于 0")
	}
	if _, err := time.Parse("2006-01-02", a.CapturedOn); err != nil {
		return NewRuleError("validation_error", "captured_on 必须是 YYYY-MM-DD")
	}
	if len(a.ParticipantCodes) == 0 {
		return NewRuleError("validation_error", "participant_codes 至少包含一项")
	}
	decoded, err := hex.DecodeString(a.ContentSHA256)
	if err != nil || len(decoded) != 32 {
		return NewRuleError("validation_error", "content_sha256 必须是 64 位十六进制 SHA-256")
	}
	return nil
}

func ValidateFinding(f SensitivityFinding, duration int64) error {
	if f.StartMS < 0 || f.EndMS <= f.StartMS || f.EndMS > duration {
		return NewRuleError("invalid_time_range", "片段时间范围必须位于录音时长内")
	}
	switch f.Category {
	case "CULTURAL", "IDENTITY", "USE_RESTRICTED":
	default:
		return NewRuleError("validation_error", "category 无效")
	}
	switch f.Severity {
	case "LOW", "MEDIUM", "HIGH":
	default:
		return NewRuleError("validation_error", "severity 无效")
	}
	switch f.Disposition {
	case "REMOVE", "MUTE", "VOICE_SHIFT", "RESTRICT_ACCESS":
	default:
		return NewRuleError("validation_error", "disposition 无效")
	}
	if f.Status == "CLOSED" && len(strings.TrimSpace(f.TreatmentNote)) < 8 {
		return NewRuleError("validation_error", "treatment_note 至少需要 8 个字符")
	}
	if f.Status != "OPEN" && f.Status != "CLOSED" {
		return NewRuleError("validation_error", "status 必须是 OPEN 或 CLOSED")
	}
	return nil
}

func normalizedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func normalizedUnique(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
