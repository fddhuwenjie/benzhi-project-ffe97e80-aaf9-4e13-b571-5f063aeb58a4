package application

import (
	"context"
	"dialect-release/internal/audit"
	"dialect-release/internal/domain"
	"sort"
	"time"
)

func (s *Service) AddConsent(ctx context.Context, caseID string, input ConsentInput) (Result, error) {
	return s.mutate(ctx, caseID, "add_consent", "CONSENT_ADDED", []string{caseID}, input.CommandMeta, input, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		id, err := audit.NewID("consent")
		if err != nil {
			return nil, err
		}
		grant := domain.ConsentGrant{ID: id, CaseID: c.ID, ParticipantCode: input.ParticipantCode, EvidenceRef: input.EvidenceRef, PermittedUses: input.PermittedUses, RegionLimits: input.RegionLimits, ValidUntil: input.ValidUntil}
		return nil, c.AddConsent(grant, now)
	})
}

func (s *Service) WithdrawConsent(ctx context.Context, caseID, consentID string, input WithdrawConsentInput) (Result, error) {
	return s.mutate(ctx, caseID, "withdraw_consent", "CONSENT_WITHDRAWN", []string{caseID, consentID}, input.CommandMeta, input, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		return nil, c.WithdrawConsent(consentID, now)
	})
}

func (s *Service) AddAsset(ctx context.Context, caseID string, input AssetInput) (Result, error) {
	return s.mutate(ctx, caseID, "add_asset", "ASSET_ADDED", []string{caseID}, input.CommandMeta, input, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		id, err := audit.NewID("asset")
		if err != nil {
			return nil, err
		}
		asset := domain.RecordingAsset{ID: id, CaseID: c.ID, StableKey: input.StableKey, Summary: input.Summary, DurationMS: input.DurationMS, CapturedOn: input.CapturedOn, ParticipantCodes: input.ParticipantCodes, ContentSHA256: input.ContentSHA256}
		return nil, c.AddAsset(asset, now)
	})
}

func (s *Service) AddAssetBatch(ctx context.Context, caseID string, input AssetBatchInput) (Result, error) {
	return s.mutate(ctx, caseID, "add_asset_batch", "ASSETS_BATCH_ADDED", []string{caseID}, input.CommandMeta, input, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		assets := make([]domain.RecordingAsset, 0, len(input.Assets))
		for _, item := range input.Assets {
			id, err := audit.NewID("asset")
			if err != nil {
				return nil, err
			}
			assets = append(assets, domain.RecordingAsset{ID: id, CaseID: c.ID, StableKey: item.StableKey, Summary: item.Summary, DurationMS: item.DurationMS, CapturedOn: item.CapturedOn, ParticipantCodes: item.ParticipantCodes, ContentSHA256: item.ContentSHA256})
		}
		sort.Slice(assets, func(i, j int) bool { return assets[i].StableKey < assets[j].StableKey })
		return nil, c.AddAssets(assets, now)
	})
}

func (s *Service) AddFinding(ctx context.Context, caseID, assetID string, input FindingInput) (Result, error) {
	return s.mutate(ctx, caseID, "add_finding", "FINDING_ADDED", []string{caseID, assetID}, input.CommandMeta, input, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		id, err := audit.NewID("finding")
		if err != nil {
			return nil, err
		}
		finding := domain.SensitivityFinding{ID: id, AssetID: assetID, StartMS: input.StartMS, EndMS: input.EndMS, Category: input.Category, Severity: input.Severity, Disposition: input.Disposition, TreatmentNote: input.TreatmentNote, Status: input.Status}
		if finding.Status == "" { finding.Status = "OPEN" }
		return nil, c.AddFinding(assetID, finding)
	})
}

func (s *Service) CloseFinding(ctx context.Context, caseID, findingID string, input CloseFindingInput) (Result, error) {
	return s.mutate(ctx, caseID, "close_finding", "FINDING_CLOSED", []string{caseID, findingID}, input.CommandMeta, input, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		return nil, c.CloseFindingWithDisposition(findingID, input.Disposition, input.TreatmentNote)
	})
}

func (s *Service) Submit(ctx context.Context, caseID string, meta CommandMeta) (Result, error) {
	return s.mutate(ctx, caseID, "submit_review", "REVIEW_SUBMITTED", []string{caseID}, meta, meta, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		return nil, c.SubmitForReview(now)
	})
}

func (s *Service) Review(ctx context.Context, caseID string, input ReviewInput) (Result, error) {
	return s.mutate(ctx, caseID, "steward_review", "STEWARD_REVIEWED", []string{caseID}, input.CommandMeta, input, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		id, err := audit.NewID("review")
		if err != nil {
			return nil, err
		}
		review := domain.StewardReview{ID: id, CaseID: c.ID, Reviewer: input.Reviewer, Decision: input.Decision, ReasonCodes: input.ReasonCodes, Comment: input.Comment, DecidedAt: now}
		return nil, c.Review(review)
	})
}

func (s *Service) ApproveAndSeal(ctx context.Context, caseID string, meta CommandMeta) (Result, error) {
	return s.mutate(ctx, caseID, "approve_and_seal", "CASE_SEALED", []string{caseID}, meta, meta, func(c *domain.ReleaseCase, now time.Time) (*domain.ReleaseManifest, error) {
		manifestID, err := audit.NewID("manifest")
		if err != nil {
			return nil, err
		}
		manifest, err := audit.BuildManifest(manifestID, c, now)
		if err != nil {
			return nil, err
		}
		if err := c.Seal(now); err != nil {
			return nil, err
		}
		return manifest, nil
	})
}
