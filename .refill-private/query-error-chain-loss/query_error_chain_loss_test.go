package query_error_chain_loss_test

import (
	"context"
	"crypto/sha256"
	"dialect-release/internal/application"
	"dialect-release/internal/domain"
	"dialect-release/internal/httpapi"
	"dialect-release/internal/store"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

type failingQueryRepository struct{}

func (failingQueryRepository) Begin(context.Context) (store.Tx, error) {
	return nil, store.ErrNotFound
}

func (failingQueryRepository) GetCase(_ context.Context, id string) (*domain.ReleaseCase, error) {
	switch id {
	case "missing", "manifest-case-missing":
		return nil, store.ErrNotFound
	default:
		return &domain.ReleaseCase{ID: id, Status: domain.StatusSealed, Revision: 2}, nil
	}
}

func (failingQueryRepository) ListCases(context.Context, string) ([]domain.ReleaseCase, error) {
	return nil, store.ErrNotFound
}

func (failingQueryRepository) Timeline(_ context.Context, id string) ([]domain.AuditEvent, error) {
	if id == "timeline-store-error" {
		return nil, store.ErrNotFound
	}
	return []domain.AuditEvent{{CaseID: id, BeforeRevision: 3, AfterRevision: 4}}, nil
}

func (failingQueryRepository) Manifest(_ context.Context, id string) (*domain.ReleaseManifest, error) {
	if id == "manifest-missing" {
		return nil, store.ErrNotFound
	}
	if id == "manifest-corrupt" {
		return &domain.ReleaseManifest{CaseID: id, CaseRevision: 2, CanonicalJSON: "{}", SHA256: "wrong"}, nil
	}
	canonical := `{"schema_version":"1","case_id":"manifest-case-missing","case_revision":2,"assets":[],"constraints":[]}`
	digest := sha256.Sum256([]byte(canonical))
	return &domain.ReleaseManifest{
		CaseID:            id,
		CaseRevision:      2,
		AssetEntries:      []domain.ManifestAsset{},
		ConstraintSummary: []string{},
		CanonicalJSON:     canonical,
		SHA256:            hex.EncodeToString(digest[:]),
	}, nil
}

func (failingQueryRepository) Close() error { return nil }

func TestQueryErrorsPreserveClassificationAcrossHTTP(t *testing.T) {
	handler := httpapi.New(application.New(failingQueryRepository{})).Handler()
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "list store error", path: "/api/v1/cases", wantStatus: http.StatusNotFound},
		{name: "timeline case not found", path: "/api/v1/cases/missing/timeline", wantStatus: http.StatusNotFound},
		{name: "timeline store error", path: "/api/v1/cases/timeline-store-error/timeline", wantStatus: http.StatusNotFound},
		{name: "timeline integrity error", path: "/api/v1/cases/timeline-corrupt/timeline", wantStatus: http.StatusUnprocessableEntity},
		{name: "manifest not found", path: "/api/v1/cases/manifest-missing/manifest", wantStatus: http.StatusNotFound},
		{name: "manifest integrity error", path: "/api/v1/cases/manifest-corrupt/manifest", wantStatus: http.StatusUnprocessableEntity},
		{name: "manifest case not found", path: "/api/v1/cases/manifest-case-missing/manifest", wantStatus: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}
