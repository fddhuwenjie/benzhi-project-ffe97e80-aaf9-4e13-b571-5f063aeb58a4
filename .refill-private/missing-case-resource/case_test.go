package missingcaseresource

import (
	"context"
	"dialect-release/internal/application"
	"dialect-release/internal/domain"
	"dialect-release/internal/httpapi"
	"dialect-release/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

type invalidatedCaseRepo struct{ store.Repository }

func (invalidatedCaseRepo) GetCase(context.Context, string) (*domain.ReleaseCase, error) {
	return nil, nil
}

func TestMissingCaseReturnsNotFoundInsteadOfInternalPanic(t *testing.T) {
	repo, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	service := application.New(invalidatedCaseRepo{Repository: repo})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cases/resource-lost", nil)
	recorder := httptest.NewRecorder()
	httpapi.New(service).Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("资源失效应返回 404，实际为 %d，响应=%s", recorder.Code, recorder.Body.String())
	}
}
