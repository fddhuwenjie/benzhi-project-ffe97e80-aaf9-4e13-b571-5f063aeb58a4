package domain

import "time"

type ReleaseCase struct {
	ID                 string           `json:"id"`
	LanguageName       string           `json:"language_name"`
	CollectionBatch    string           `json:"collection_batch"`
	Owner              string           `json:"owner"`
	ReleaseLevel       string           `json:"release_level"`
	Status             CaseStatus       `json:"status"`
	Revision           int64            `json:"revision"`
	CreatedAt          time.Time        `json:"created_at"`
	SealedAt           *time.Time       `json:"sealed_at,omitempty"`
	Consents           []ConsentGrant   `json:"consents"`
	Assets             []RecordingAsset `json:"assets"`
	Reviews            []StewardReview  `json:"reviews"`
	AssetCount         int              `json:"asset_count"`
	OpenFindingCount   int              `json:"open_finding_count"`
	LatestReviewRound  int              `json:"latest_review_round"`
	RemediationSummary []string         `json:"remediation_summary"`
}

type ConsentGrant struct {
	ID                string     `json:"id"`
	CaseID            string     `json:"case_id"`
	ParticipantCode   string     `json:"participant_code"`
	EvidenceRef       string     `json:"evidence_ref"`
	PermittedUses     []string   `json:"permitted_uses"`
	RegionLimits      []string   `json:"region_limits"`
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
	WithdrawnAt       *time.Time `json:"withdrawn_at,omitempty"`
	Status            string     `json:"status,omitempty"`
	CoveredAssetCount int        `json:"covered_asset_count,omitempty"`
}

type RecordingAsset struct {
	ID               string               `json:"id"`
	CaseID           string               `json:"case_id"`
	StableKey        string               `json:"stable_key"`
	Summary          string               `json:"summary"`
	DurationMS       int64                `json:"duration_ms"`
	CapturedOn       string               `json:"captured_on"`
	ParticipantCodes []string             `json:"participant_codes"`
	ContentSHA256    string               `json:"content_sha256"`
	Findings         []SensitivityFinding `json:"findings"`
}

type SensitivityFinding struct {
	ID            string `json:"id"`
	AssetID       string `json:"asset_id"`
	StartMS       int64  `json:"start_ms"`
	EndMS         int64  `json:"end_ms"`
	Category      string `json:"category"`
	Severity      string `json:"severity"`
	Disposition   string `json:"disposition"`
	TreatmentNote string `json:"treatment_note"`
	Status        string `json:"status"`
}

type StewardReview struct {
	ID           string    `json:"id"`
	CaseID       string    `json:"case_id"`
	Round        int       `json:"round"`
	Reviewer     string    `json:"reviewer"`
	Decision     string    `json:"decision"`
	ReasonCodes  []string  `json:"reason_codes"`
	Comment      string    `json:"comment"`
	DecidedAt    time.Time `json:"decided_at"`
	FindingCount int       `json:"finding_count"`
}

type ManifestAsset struct {
	StableKey     string   `json:"stable_key"`
	Summary       string   `json:"summary"`
	DurationMS    int64    `json:"duration_ms"`
	CapturedOn    string   `json:"captured_on"`
	Participants  []string `json:"participant_codes"`
	ContentSHA256 string   `json:"content_sha256"`
}

type ReleaseManifest struct {
	ID                string          `json:"id"`
	CaseID            string          `json:"case_id"`
	CaseRevision      int64           `json:"case_revision"`
	AssetEntries      []ManifestAsset `json:"asset_entries"`
	ConstraintSummary []string        `json:"constraint_summary"`
	CanonicalJSON     string          `json:"canonical_json"`
	SHA256            string          `json:"sha256"`
	SealedAt          time.Time       `json:"sealed_at"`
}

type AuditEvent struct {
	Sequence       int64          `json:"sequence"`
	CaseID         string         `json:"case_id"`
	Actor          string         `json:"actor"`
	RequestID      string         `json:"request_id"`
	BeforeRevision int64          `json:"before_revision"`
	AfterRevision  int64          `json:"after_revision"`
	EventType      string         `json:"event_type"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Details        map[string]any `json:"details,omitempty"`
}
