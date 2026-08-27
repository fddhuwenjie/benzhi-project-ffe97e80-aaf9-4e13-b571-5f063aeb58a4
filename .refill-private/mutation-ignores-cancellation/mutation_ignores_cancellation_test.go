package mutationignorescancellation

import (
	"context"
	"dialect-release/internal/application"
	"dialect-release/internal/store"
	"encoding/json"
	"errors"
	"testing"
)

func TestCanceledMutationDoesNotCommit(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/case.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	service := application.New(repo)
	created, err := service.CreateCase(context.Background(), application.CreateCaseInput{
		RequestID:       "create-cancel-case",
		Actor:           "管理员",
		LanguageName:    "测试方言",
		CollectionBatch: "batch-1",
		Owner:           "负责人",
		ReleaseLevel:    "PUBLIC",
	})
	if err != nil {
		t.Fatal(err)
	}
	var envelope application.CaseEnvelope
	if err := json.Unmarshal(created.Body, &envelope); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.AddConsent(ctx, envelope.Case.ID, application.ConsentInput{
		CommandMeta: application.CommandMeta{RequestID: "canceled-consent", Actor: "管理员", ExpectedRevision: 1},
		ParticipantCode: "P-1",
		EvidenceRef:     "evidence-1",
		PermittedUses:   []string{"PUBLIC"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("取消的变更应返回 context.Canceled，得到 %v", err)
	}

	current, err := service.GetCase(context.Background(), envelope.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Revision != 1 || len(current.Consents) != 0 {
		t.Fatalf("取消的变更不应提交，revision=%d consents=%d", current.Revision, len(current.Consents))
	}
}
