package canceledcaselock_test

import (
	"context"
	"dialect-release/internal/application"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"errors"
	"sync"
	"testing"
	"time"
)

type lockRepository struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *lockRepository) Begin(ctx context.Context) (store.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &lockTx{repo: r, ctx: ctx}, nil
}
func (*lockRepository) GetCase(context.Context, string) (*domain.ReleaseCase, error) {
	return nil, store.ErrNotFound
}
func (*lockRepository) ListCases(context.Context, string) ([]domain.ReleaseCase, error) {
	return nil, nil
}
func (*lockRepository) Timeline(context.Context, string) ([]domain.AuditEvent, error) {
	return nil, nil
}
func (*lockRepository) Manifest(context.Context, string) (*domain.ReleaseManifest, error) {
	return nil, store.ErrNotFound
}
func (*lockRepository) Close() error { return nil }

type lockTx struct {
	repo *lockRepository
	ctx  context.Context
}

func (t *lockTx) LoadCase(context.Context, string) (*domain.ReleaseCase, error) {
	t.repo.once.Do(func() { close(t.repo.entered) })
	select {
	case <-t.repo.release:
		return &domain.ReleaseCase{ID: "case-1", Status: domain.StatusDraft, Revision: 1, ReleaseLevel: "PUBLIC"}, nil
	case <-t.ctx.Done():
		return nil, t.ctx.Err()
	}
}
func (*lockTx) InsertCase(context.Context, *domain.ReleaseCase) error       { return nil }
func (*lockTx) SaveCase(context.Context, *domain.ReleaseCase, int64) error  { return nil }
func (*lockTx) AppendEvent(context.Context, domain.AuditEvent) error        { return nil }
func (*lockTx) SaveManifest(context.Context, *domain.ReleaseManifest) error { return nil }
func (*lockTx) GetIdempotency(context.Context, string) (*store.IdempotencyRecord, error) {
	return nil, store.ErrNotFound
}
func (*lockTx) SaveIdempotency(context.Context, store.IdempotencyRecord) error { return nil }
func (*lockTx) Commit() error                                                  { return nil }
func (*lockTx) Rollback() error                                                { return nil }

func TestCanceledMutationDoesNotWaitForCaseLock(t *testing.T) {
	repo := &lockRepository{entered: make(chan struct{}), release: make(chan struct{})}
	service := application.New(repo)
	firstDone := make(chan error, 1)
	go func() {
		_, err := service.AddConsent(context.Background(), "case-1", application.ConsentInput{
			CommandMeta:     application.CommandMeta{RequestID: "first", ExpectedRevision: 1, Actor: "actor"},
			ParticipantCode: "P001", EvidenceRef: "evidence", PermittedUses: []string{"PUBLIC"},
		})
		firstDone <- err
	}()
	<-repo.entered

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	secondDone := make(chan error, 1)
	go func() {
		_, err := service.AddConsent(canceled, "case-1", application.ConsentInput{
			CommandMeta:     application.CommandMeta{RequestID: "second", ExpectedRevision: 1, Actor: "actor"},
			ParticipantCode: "P002", EvidenceRef: "evidence", PermittedUses: []string{"PUBLIC"},
		})
		secondDone <- err
	}()

	timer := time.NewTimer(500 * time.Millisecond)
	returnedBeforeUnlock := false
	var secondErr error
	select {
	case secondErr = <-secondDone:
		returnedBeforeUnlock = true
	case <-timer.C:
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	close(repo.release)
	if !returnedBeforeUnlock {
		secondErr = <-secondDone
	}
	if firstErr := <-firstDone; firstErr != nil {
		t.Fatalf("持锁请求意外失败: %v", firstErr)
	}
	if !returnedBeforeUnlock || !errors.Is(secondErr, context.Canceled) {
		t.Fatalf("已取消请求应在持锁请求释放前返回 context.Canceled；returnedBeforeUnlock=%v err=%v", returnedBeforeUnlock, secondErr)
	}
}
