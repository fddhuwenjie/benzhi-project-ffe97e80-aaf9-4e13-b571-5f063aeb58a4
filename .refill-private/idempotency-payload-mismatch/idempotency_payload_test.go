package idempotencypayload_test

import (
	"context"
	"dialect-release/internal/application"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func TestIdempotencyKeyRejectsChangedPayload(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "payload.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	service := application.New(repo)
	created, err := service.CreateCase(context.Background(), application.CreateCaseInput{
		RequestID: "create", Actor: "actor", LanguageName: "方言", CollectionBatch: "batch", Owner: "owner", ReleaseLevel: "PUBLIC",
	})
	if err != nil {
		t.Fatal(err)
	}
	var envelope application.CaseEnvelope
	if err := json.Unmarshal(created.Body, &envelope); err != nil {
		t.Fatal(err)
	}
	meta := application.CommandMeta{RequestID: "same-key", ExpectedRevision: 1, Actor: "actor"}
	first := application.ConsentInput{CommandMeta: meta, ParticipantCode: "P001", EvidenceRef: "evidence://P001", PermittedUses: []string{"PUBLIC"}}
	if _, err := service.AddConsent(context.Background(), envelope.Case.ID, first); err != nil {
		t.Fatal(err)
	}
	changed := application.ConsentInput{CommandMeta: meta, ParticipantCode: "P002", EvidenceRef: "evidence://P002", PermittedUses: []string{"PUBLIC"}}
	result, err := service.AddConsent(context.Background(), envelope.Case.ID, changed)
	var rule *domain.RuleError
	if !errors.As(err, &rule) || rule.Code != "request_id_reused" {
		t.Fatalf("同一 request_id 携带不同业务载荷时应返回 request_id_reused，实际 err=%v replayed=%v body=%s", err, result.Replayed, result.Body)
	}
}
