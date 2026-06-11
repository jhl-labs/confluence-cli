package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"confluence-cli/internal/config"
)

func TestSearch(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(SearchResult{
			Results: []Content{{ID: "1", Title: "A"}}, Size: 1, Limit: 25,
		})
	}))
	defer srv.Close()

	res, err := testClient(t, srv).Search(context.Background(), `text ~ "x"`, 25, []string{"space", "version"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("results = %d", len(res.Results))
	}
	if !strings.Contains(gotQuery, "cql=") || !strings.Contains(gotQuery, "expand=space%2Cversion") {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestCreate(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		json.NewEncoder(w).Encode(Content{ID: "100", Title: "New"})
	}))
	defer srv.Close()

	out, err := testClient(t, srv).Create(context.Background(), CreateInput{
		SpaceKey: "ENG", Title: "New", Body: "<p>x</p>", ParentID: "5",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.ID != "100" {
		t.Errorf("id = %q", out.ID)
	}
	if body["title"] != "New" {
		t.Errorf("title = %v", body["title"])
	}
	if anc, ok := body["ancestors"].([]any); !ok || len(anc) != 1 {
		t.Errorf("ancestors = %v", body["ancestors"])
	}
	// default representation should be storage
	bodyField := body["body"].(map[string]any)
	if _, ok := bodyField["storage"]; !ok {
		t.Errorf("expected storage body, got %v", bodyField)
	}
}

func TestCreateWikiRepresentation(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		json.NewEncoder(w).Encode(Content{ID: "1"})
	}))
	defer srv.Close()

	_, err := testClient(t, srv).Create(context.Background(), CreateInput{
		SpaceKey: "ENG", Title: "W", Body: "h1. Hi", Representation: RepresentationWiki,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	bf := body["body"].(map[string]any)
	if _, ok := bf["wiki"]; !ok {
		t.Errorf("expected wiki body, got %v", bf)
	}
}

func TestAddComment(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		json.NewEncoder(w).Encode(Content{ID: "c1", Type: "comment"})
	}))
	defer srv.Close()

	out, err := testClient(t, srv).AddComment(context.Background(), "42", "<p>hi</p>", "")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if out.ID != "c1" {
		t.Errorf("id = %q", out.ID)
	}
	if body["type"] != "comment" {
		t.Errorf("type = %v", body["type"])
	}
	cont := body["container"].(map[string]any)
	if cont["id"] != "42" {
		t.Errorf("container id = %v", cont["id"])
	}
}

func TestGetSpaceAndRootPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/content/page"):
			json.NewEncoder(w).Encode(ContentList{Results: []Content{{ID: "10", Title: "Root"}}, Size: 1})
		default:
			json.NewEncoder(w).Encode(Space{Key: "ENG", Name: "Eng", Homepage: &Content{ID: "9", Title: "Home"}})
		}
	}))
	defer srv.Close()
	cl := testClient(t, srv)

	sp, err := cl.GetSpace(context.Background(), "ENG", []string{"homepage"})
	if err != nil {
		t.Fatalf("GetSpace: %v", err)
	}
	if sp.Homepage == nil || sp.Homepage.ID != "9" {
		t.Errorf("homepage = %+v", sp.Homepage)
	}

	roots, err := cl.GetSpaceRootPages(context.Background(), "ENG", 50)
	if err != nil {
		t.Fatalf("GetSpaceRootPages: %v", err)
	}
	if len(roots.Results) != 1 || roots.Results[0].ID != "10" {
		t.Errorf("roots = %+v", roots.Results)
	}
}

func TestDeletePage(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := testClient(t, srv).DeletePage(context.Background(), "7"); err != nil {
		t.Fatalf("DeletePage: %v", err)
	}
	if method != http.MethodDelete {
		t.Errorf("method = %s", method)
	}
}

func TestGetLabels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(LabelList{Results: []Label{{Name: "a"}, {Name: "b"}}, Size: 2})
	}))
	defer srv.Close()

	res, err := testClient(t, srv).GetLabels(context.Background(), "1")
	if err != nil {
		t.Fatalf("GetLabels: %v", err)
	}
	if res.Size != 2 {
		t.Errorf("size = %d", res.Size)
	}
}

func TestWebURL(t *testing.T) {
	cl := &Client{baseURL: "https://x.example.com"}
	c := Content{Links: map[string]string{"webui": "/display/ENG/Page"}}
	if got := cl.WebURL(c); got != "https://x.example.com/display/ENG/Page" {
		t.Errorf("WebURL = %q", got)
	}
	if got := cl.WebURL(Content{}); got != "" {
		t.Errorf("WebURL(empty) = %q", got)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	e := &APIError{StatusCode: 404, Message: "not found"}
	if !strings.Contains(e.Error(), "404") || !strings.Contains(e.Error(), "not found") {
		t.Errorf("Error() = %q", e.Error())
	}
	e2 := &APIError{StatusCode: 500}
	if !strings.Contains(e2.Error(), "500") {
		t.Errorf("Error() = %q", e2.Error())
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	if _, err := New(config.Config{}, 0); err == nil {
		t.Fatal("expected error for missing creds")
	}
}
