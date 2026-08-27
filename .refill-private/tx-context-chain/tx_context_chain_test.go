package txcontextchain_test

import (
	"context"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"errors"
	"testing"
	"time"
)

func TestTransactionOperationsPreserveCancellationCause(t *testing.T) {
	repository, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()

	caseValue := &domain.ReleaseCase{
		ID:              "case-context-chain",
		LanguageName:    "测试方言",
		CollectionBatch: "batch-context",
		Owner:           "管理员",
		ReleaseLevel:    "PUBLIC",
		Status:          domain.StatusDraft,
		Revision:        1,
		CreatedAt:       time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	}
	manifest := &domain.ReleaseManifest{
		ID:            "manifest-context-chain",
		CaseID:        caseValue.ID,
		CaseRevision:  2,
		CanonicalJSON: `{}`,
		SHA256:        "digest",
		SealedAt:      caseValue.CreatedAt,
	}
	event := domain.AuditEvent{
		CaseID:        caseValue.ID,
		Actor:         "管理员",
		RequestID:     "request-context-chain",
		EventType:     "CONTEXT_TEST",
		OccurredAt:    caseValue.CreatedAt,
		AfterRevision: 1,
	}
	record := store.IdempotencyRecord{
		RequestID:  "request-context-chain",
		CaseID:     caseValue.ID,
		Operation:  "context_test",
		StatusCode: 200,
		Body:       []byte(`{}`),
	}

	operations := []struct {
		name string
		run  func(store.Tx, context.Context) error
	}{
		{"InsertCase", func(tx store.Tx, ctx context.Context) error { return tx.InsertCase(ctx, caseValue) }},
		{"SaveCase", func(tx store.Tx, ctx context.Context) error { return tx.SaveCase(ctx, caseValue, 1) }},
		{"AppendEvent", func(tx store.Tx, ctx context.Context) error { return tx.AppendEvent(ctx, event) }},
		{"SaveManifest", func(tx store.Tx, ctx context.Context) error { return tx.SaveManifest(ctx, manifest) }},
		{"GetIdempotency", func(tx store.Tx, ctx context.Context) error {
			_, err := tx.GetIdempotency(ctx, record.RequestID)
			return err
		}},
		{"SaveIdempotency", func(tx store.Tx, ctx context.Context) error { return tx.SaveIdempotency(ctx, record) }},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			tx, err := repository.Begin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = operation.run(tx, ctx)
			_ = tx.Rollback()
			if err == nil {
				t.Fatal("取消后的事务操作意外成功")
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("%s 丢失 context.Canceled 错误链: %v", operation.name, err)
			}
		})
	}
}
