package confluence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"confluence-cli/internal/config"
)

func newTestClient(t *testing.T, srv *httptest.Server, cfg config.Config) *Client {
	t.Helper()
	cfg.BaseURL = srv.URL
	cl, err := New(cfg, 5*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cl.RetryWait = time.Millisecond // keep tests fast
	return cl
}

func TestGetSendsBearerAuth(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(Content{ID: "42", Type: "page", Title: "Hi"})
	}))
	defer srv.Close()

	cl := newTestClient(t, srv, config.Config{Token: "secret-pat"})
	c, err := cl.Get(context.Background(), "42", []string{"version"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotAuth != "Bearer secret-pat" {
		t.Errorf("auth header = %q, want Bearer secret-pat", gotAuth)
	}
	if gotPath != "/rest/api/content/42" {
		t.Errorf("path = %q", gotPath)
	}
	if c.Title != "Hi" {
		t.Errorf("title = %q", c.Title)
	}
}

func TestBasicAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(Content{ID: "1"})
	}))
	defer srv.Close()

	cl := newTestClient(t, srv, config.Config{User: "alice", Password: "pw"})
	if _, err := cl.Get(context.Background(), "1", nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	// base64("alice:pw") = YWxpY2U6cHc=
	if gotAuth != "Basic YWxpY2U6cHc=" {
		t.Errorf("auth header = %q", gotAuth)
	}
}

func TestRetriesOn503ThenSucceeds(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(Content{ID: "ok"})
	}))
	defer srv.Close()

	cl := newTestClient(t, srv, config.Config{Token: "t"})
	c, err := cl.Get(context.Background(), "ok", nil)
	if err != nil {
		t.Fatalf("Get after retries: %v", err)
	}
	if calls != 3 {
		t.Errorf("server calls = %d, want 3", calls)
	}
	if c.ID != "ok" {
		t.Errorf("id = %q", c.ID)
	}
}

func TestNoRetryOn404(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"statusCode":404,"message":"No content found"}`))
	}))
	defer srv.Close()

	cl := newTestClient(t, srv, config.Config{Token: "t"})
	_, err := cl.Get(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 404 || apiErr.Message != "No content found" {
		t.Errorf("apiErr = %+v", apiErr)
	}
	if calls != 1 {
		t.Errorf("server calls = %d, want 1 (no retry on 404)", calls)
	}
}

func TestUpdateIncrementsVersion(t *testing.T) {
	var putBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(Content{
				ID: "7", Type: "page", Title: "Old",
				Version: &Version{Number: 4},
			})
		case http.MethodPut:
			json.NewDecoder(r.Body).Decode(&putBody)
			json.NewEncoder(w).Encode(Content{ID: "7", Version: &Version{Number: 5}})
		}
	}))
	defer srv.Close()

	cl := newTestClient(t, srv, config.Config{Token: "t"})
	c, err := cl.Update(context.Background(), UpdateInput{ID: "7", Body: "<p>new</p>"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	ver, _ := putBody["version"].(map[string]any)
	if ver == nil || ver["number"].(float64) != 5 {
		t.Errorf("PUT version = %v, want 5", putBody["version"])
	}
	// Title should be preserved from the fetched content.
	if putBody["title"] != "Old" {
		t.Errorf("title = %v, want Old", putBody["title"])
	}
	if c.Version.Number != 5 {
		t.Errorf("returned version = %d", c.Version.Number)
	}
}
