package detail_cache_lost_invalidation_test

import (
	"context"
	"dialect-release/internal/application"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
)

type pausedReadRepository struct {
	store.Repository
	captured chan struct{}
	release  chan struct{}
	once     sync.Once
}

func (r *pausedReadRepository) GetCase(ctx context.Context, id string) (*domain.ReleaseCase, error) {
	value, err := r.Repository.GetCase(ctx, id)
	if err != nil {
		return nil, err
	}
	r.once.Do(func() {
		close(r.captured)
		select {
		case <-r.release:
		case <-ctx.Done():
		}
	})
	return value, ctx.Err()
}

func TestGetCaseCacheDoesNotResurrectStaleSnapshot(t *testing.T) {
	base, err := store.Open(filepath.Join(t.TempDir(), "case-cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	repo := &pausedReadRepository{Repository: base, captured: make(chan struct{}), release: make(chan struct{})}
	service := application.New(repo)

	created, err := service.CreateCase(context.Background(), application.CreateCaseInput{
		RequestID: "create-cache-case", Actor: "管理员", LanguageName: "测试方言",
		CollectionBatch: "CACHE-01", Owner: "负责人", ReleaseLevel: "PUBLIC",
	})
	if err != nil {
		t.Fatal(err)
	}
	var envelope application.CaseEnvelope
	if err := json.Unmarshal(created.Body, &envelope); err != nil {
		t.Fatal(err)
	}

	readDone := make(chan error, 1)
	go func() {
		_, readErr := service.GetCase(context.Background(), envelope.Case.ID)
		readDone <- readErr
	}()
	<-repo.captured

	_, err = service.AddConsent(context.Background(), envelope.Case.ID, application.ConsentInput{
		CommandMeta:     application.CommandMeta{RequestID: "add-cache-consent", Actor: "管理员", ExpectedRevision: 1},
		ParticipantCode: "P001", EvidenceRef: "consent://P001", PermittedUses: []string{"PUBLIC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	close(repo.release)
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}

	got, err := service.GetCase(context.Background(), envelope.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 || got.Status != domain.StatusConsentReady || len(got.Consents) != 1 {
		t.Fatalf("并发失效后返回了过期详情: revision=%d status=%s consents=%d", got.Revision, got.Status, len(got.Consents))
	}
}
