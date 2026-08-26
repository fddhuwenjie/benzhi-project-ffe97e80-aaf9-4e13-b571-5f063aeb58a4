package httpapi

import (
	"bytes"
	"dialect-release/internal/application"
	"dialect-release/internal/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func testAPI(t *testing.T) *httptest.Server {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(application.New(repo)).Handler())
	t.Cleanup(func() { server.Close(); repo.Close() })
	return server
}

func TestStrictJSONAndContentType(t *testing.T) {
	server := testAPI(t)
	request, err := http.NewRequest("POST", server.URL+"/api/v1/cases", bytes.NewBufferString(`{"request_id":"r","actor":"a","language_name":"方言","collection_batch":"b","owner":"o","release_level":"PUBLIC","unknown":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatalf("未知字段返回 %d", response.StatusCode)
	}
	request, err = http.NewRequest("POST", server.URL+"/api/v1/cases", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 415 {
		t.Fatalf("错误 Content-Type 返回 %d", response.StatusCode)
	}
}

func TestHealthAndMissingCase(t *testing.T) {
	server := testAPI(t)
	response, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal(response.Status)
	}
	response, err = http.Get(server.URL + "/api/v1/cases/missing")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 404 {
		t.Fatal(response.Status)
	}
}
