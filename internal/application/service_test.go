package application

import (
	"context"
	"dialect-release/internal/store"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestIdempotencySurvivesRepositoryRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	repo, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := New(repo)
	input := CreateCaseInput{RequestID: "req-create", Actor: "管理员", LanguageName: "测试方言", CollectionBatch: "B1", Owner: "负责人", ReleaseLevel: "PUBLIC"}
	first, err := service.CreateCase(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	repo, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	service = New(repo)
	second, err := service.CreateCase(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || string(first.Body) != string(second.Body) {
		t.Fatal("重启后未返回原始幂等响应")
	}
	var decoded CaseEnvelope
	if err := json.Unmarshal(second.Body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Revision != 1 || decoded.Case.ID == "" {
		t.Fatalf("响应错误: %+v", decoded)
	}
}

func TestExpectedRevisionConflict(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "revision.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	service := New(repo)
	created, err := service.CreateCase(context.Background(), CreateCaseInput{RequestID: "create", Actor: "a", LanguageName: "方言", CollectionBatch: "b", Owner: "o", ReleaseLevel: "PUBLIC"})
	if err != nil {
		t.Fatal(err)
	}
	var envelope CaseEnvelope
	if err := json.Unmarshal(created.Body, &envelope); err != nil {
		t.Fatal(err)
	}
	_, err = service.AddConsent(context.Background(), envelope.Case.ID, ConsentInput{CommandMeta: CommandMeta{RequestID: "bad-revision", Actor: "a", ExpectedRevision: 9}, ParticipantCode: "P", EvidenceRef: "ref", PermittedUses: []string{"PUBLIC"}})
	if err == nil {
		t.Fatal("错误 revision 应被拒绝")
	}
}
