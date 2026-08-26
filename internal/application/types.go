package application

import (
	"dialect-release/internal/domain"
	"encoding/json"
	"time"
)

type CommandMeta struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
	Actor            string `json:"actor"`
}

type Result struct {
	StatusCode int
	Body       json.RawMessage
	Replayed   bool
}

type CreateCaseInput struct {
	RequestID       string `json:"request_id"`
	Actor           string `json:"actor"`
	LanguageName    string `json:"language_name"`
	CollectionBatch string `json:"collection_batch"`
	Owner           string `json:"owner"`
	ReleaseLevel    string `json:"release_level"`
}

type ConsentInput struct {
	CommandMeta
	ParticipantCode string     `json:"participant_code"`
	EvidenceRef     string     `json:"evidence_ref"`
	PermittedUses   []string   `json:"permitted_uses"`
	RegionLimits    []string   `json:"region_limits"`
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
}

type AssetInput struct {
	CommandMeta
	StableKey        string   `json:"stable_key"`
	Summary          string   `json:"summary"`
	DurationMS       int64    `json:"duration_ms"`
	CapturedOn       string   `json:"captured_on"`
	ParticipantCodes []string `json:"participant_codes"`
	ContentSHA256    string   `json:"content_sha256"`
}

type AssetBatchInput struct {
	CommandMeta
	Assets []AssetInput `json:"assets"`
}

type FindingInput struct {
	CommandMeta
	StartMS       int64  `json:"start_ms"`
	EndMS         int64  `json:"end_ms"`
	Category      string `json:"category"`
	Severity      string `json:"severity"`
	Disposition   string `json:"disposition"`
	TreatmentNote string `json:"treatment_note"`
	Status        string `json:"status"`
}

type CloseFindingInput struct {
	CommandMeta
	Disposition   string `json:"disposition,omitempty"`
	TreatmentNote string `json:"treatment_note"`
}
type WithdrawConsentInput struct{ CommandMeta }
type ReviewInput struct {
	CommandMeta
	Reviewer    string   `json:"reviewer"`
	Decision    string   `json:"decision"`
	ReasonCodes []string `json:"reason_codes"`
	Comment     string   `json:"comment"`
}

type CaseEnvelope struct {
	Case     *domain.ReleaseCase `json:"case"`
	Revision int64               `json:"revision"`
}
type ManifestEnvelope struct {
	Manifest *domain.ReleaseManifest `json:"manifest"`
	Revision int64                   `json:"revision"`
}

type CaseListQuery struct {
	Status       string
	ReleaseLevel string
	Owner        string
	CreatedFrom  string
	CreatedTo    string
}

type CaseListResult struct {
	Cases        []domain.ReleaseCase `json:"cases"`
	Count        int                  `json:"count"`
	StatusCounts map[string]int       `json:"status_counts"`
}

type TimelineQuery struct {
	Limit          int
	AfterRevision  int64
	BeforeRevision int64
}
type TimelineResult struct {
	Events     []domain.AuditEvent `json:"events"`
	Count      int                 `json:"count"`
	NextCursor string              `json:"next_cursor,omitempty"`
}
