package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"confluence-cli/internal/config"
)

func testClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	cl, err := New(config.Config{BaseURL: srv.URL, Token: "t"}, 5*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cl.RetryWait = time.Millisecond
	return cl
}

func TestListSpaces(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(SpaceList{Results: []Space{{Key: "ENG", Name: "Engineering", Type: "global"}}, Size: 1})
	}))
	defer srv.Close()

	res, err := testClient(t, srv).ListSpaces(context.Background(), "global", 10, 0)
	if err != nil {
		t.Fatalf("ListSpaces: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Key != "ENG" {
		t.Errorf("unexpected results: %+v", res.Results)
	}
	if gotQuery != "limit=10&type=global" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestGetChildPages(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(ContentList{
			Results: []Content{{ID: "2", Title: "Child"}},
			Size:    1, Limit: 100,
		})
	}))
	defer srv.Close()

	res, err := testClient(t, srv).GetChildPages(context.Background(), "1", 100, 0)
	if err != nil {
		t.Fatalf("GetChildPages: %v", err)
	}
	if gotPath != "/rest/api/content/1/child/page" {
		t.Errorf("path = %q", gotPath)
	}
	if len(res.Results) != 1 || res.Results[0].ID != "2" {
		t.Errorf("results = %+v", res.Results)
	}
}

func TestMovePageReparents(t *testing.T) {
	var putBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(Content{
				ID: "7", Type: "page", Title: "Movable",
				Version: &Version{Number: 3},
				Body:    &Body{Storage: &BodyValue{Value: "<p>x</p>", Representation: "storage"}},
			})
		case http.MethodPut:
			data, _ := io.ReadAll(r.Body)
			json.Unmarshal(data, &putBody)
			json.NewEncoder(w).Encode(Content{ID: "7", Version: &Version{Number: 4}})
		}
	}))
	defer srv.Close()

	out, err := testClient(t, srv).MovePage(context.Background(), "7", "9")
	if err != nil {
		t.Fatalf("MovePage: %v", err)
	}
	// ancestors should target the new parent
	anc, ok := putBody["ancestors"].([]any)
	if !ok || len(anc) != 1 {
		t.Fatalf("ancestors = %v", putBody["ancestors"])
	}
	if anc[0].(map[string]any)["id"] != "9" {
		t.Errorf("ancestor id = %v, want 9", anc[0])
	}
	// version bumped, title/body preserved
	if v := putBody["version"].(map[string]any); v["number"].(float64) != 4 {
		t.Errorf("version = %v, want 4", putBody["version"])
	}
	if putBody["title"] != "Movable" {
		t.Errorf("title = %v", putBody["title"])
	}
	if out.Version.Number != 4 {
		t.Errorf("returned version = %d", out.Version.Number)
	}
}

func TestMovePageToRoot(t *testing.T) {
	var putBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(Content{ID: "7", Type: "page", Title: "T", Version: &Version{Number: 1}})
		case http.MethodPut:
			data, _ := io.ReadAll(r.Body)
			json.Unmarshal(data, &putBody)
			json.NewEncoder(w).Encode(Content{ID: "7", Version: &Version{Number: 2}})
		}
	}))
	defer srv.Close()

	if _, err := testClient(t, srv).MovePage(context.Background(), "7", ""); err != nil {
		t.Fatalf("MovePage to root: %v", err)
	}
	anc, ok := putBody["ancestors"].([]any)
	if !ok || len(anc) != 0 {
		t.Errorf("ancestors should be empty for to-root, got %v", putBody["ancestors"])
	}
}

func TestAddLabels(t *testing.T) {
	var body []Label
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			data, _ := io.ReadAll(r.Body)
			json.Unmarshal(data, &body)
		}
		json.NewEncoder(w).Encode(LabelList{Results: []Label{{Name: "alpha"}, {Name: "beta"}}, Size: 2})
	}))
	defer srv.Close()

	res, err := testClient(t, srv).AddLabels(context.Background(), "1", []string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("AddLabels: %v", err)
	}
	if len(body) != 2 || body[0].Name != "alpha" || body[0].Prefix != "global" {
		t.Errorf("posted body = %+v", body)
	}
	if res.Size != 2 {
		t.Errorf("size = %d", res.Size)
	}
}

func TestRemoveLabel(t *testing.T) {
	var gotMethod, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotQuery = r.Method, r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := testClient(t, srv).RemoveLabel(context.Background(), "1", "alpha"); err != nil {
		t.Fatalf("RemoveLabel: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotQuery != "name=alpha" {
		t.Errorf("query = %q", gotQuery)
	}
}
