package timeline_cache_alias_test

import (
	"context"
	"dialect-release/internal/application"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"errors"
	"testing"
)

type sealedTimelineRepository struct {
	caseValue    domain.ReleaseCase
	events       []domain.AuditEvent
	timelineRead int
}

func (r *sealedTimelineRepository) Begin(context.Context) (store.Tx, error) {
	return nil, errors.New("unexpected transaction")
}

func (r *sealedTimelineRepository) GetCase(context.Context, string) (*domain.ReleaseCase, error) {
	value := r.caseValue
	return &value, nil
}

func (r *sealedTimelineRepository) ListCases(context.Context, string) ([]domain.ReleaseCase, error) {
	return nil, errors.New("unexpected case list")
}

func (r *sealedTimelineRepository) Timeline(context.Context, string) ([]domain.AuditEvent, error) {
	r.timelineRead++
	return append([]domain.AuditEvent(nil), r.events...), nil
}

func (r *sealedTimelineRepository) Manifest(context.Context, string) (*domain.ReleaseManifest, error) {
	return nil, errors.New("unexpected manifest read")
}

func (r *sealedTimelineRepository) Close() error { return nil }

func TestTimelineCacheIsolatedFromReturnedEventMutation(t *testing.T) {
	repo := &sealedTimelineRepository{
		caseValue: domain.ReleaseCase{ID: "case-sealed", Status: domain.StatusSealed, Revision: 3},
		events: []domain.AuditEvent{
			{CaseID: "case-sealed", BeforeRevision: 0, AfterRevision: 1, Details: map[string]any{"status": "DRAFT"}},
			{CaseID: "case-sealed", BeforeRevision: 1, AfterRevision: 2, Details: map[string]any{"status": "APPROVED"}},
			{CaseID: "case-sealed", BeforeRevision: 2, AfterRevision: 3, Details: map[string]any{"status": "SEALED"}},
		},
	}
	service := application.New(repo)

	initial, err := service.Timeline(context.Background(), "case-sealed")
	if err != nil || len(initial) != 3 {
		t.Fatalf("initial timeline read failed: len=%d err=%v", len(initial), err)
	}
	initial[0].Details["status"] = "FORGED_BY_CALLER"

	complete, err := service.Timeline(context.Background(), "case-sealed")
	if err != nil {
		t.Fatal(err)
	}
	if repo.timelineRead != 1 {
		t.Fatalf("sealed timeline cache was not reused: reads=%d", repo.timelineRead)
	}
	if got := complete[0].Details["status"]; got != "DRAFT" {
		t.Fatalf("cached audit details were changed through returned alias: got=%v", got)
	}
}
