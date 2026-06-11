package confluence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test500ErrorPaths drives every method against a failing server to cover the
// error-return branches.
func Test500ErrorPaths(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"boom"}`))
	}))
	defer srv.Close()
	cl := testClient(t, srv)
	cl.MaxRetries = 0
	ctx := context.Background()

	calls := map[string]func() error{
		"Search":            func() error { _, e := cl.Search(ctx, "x", 10, nil); return e },
		"Get":               func() error { _, e := cl.Get(ctx, "1", nil); return e },
		"Create":            func() error { _, e := cl.Create(ctx, CreateInput{SpaceKey: "E", Title: "T"}); return e },
		"Update":            func() error { _, e := cl.Update(ctx, UpdateInput{ID: "1"}); return e },
		"AddComment":        func() error { _, e := cl.AddComment(ctx, "1", "b", ""); return e },
		"ListSpaces":        func() error { _, e := cl.ListSpaces(ctx, "", 0, 0); return e },
		"GetSpace":          func() error { _, e := cl.GetSpace(ctx, "K", nil); return e },
		"GetChildPages":     func() error { _, e := cl.GetChildPages(ctx, "1", 0, 0); return e },
		"GetSpaceRootPages": func() error { _, e := cl.GetSpaceRootPages(ctx, "K", 0); return e },
		"MovePage":          func() error { _, e := cl.MovePage(ctx, "1", "2"); return e },
		"DeletePage":        func() error { return cl.DeletePage(ctx, "1") },
		"GetLabels":         func() error { _, e := cl.GetLabels(ctx, "1"); return e },
		"AddLabels":         func() error { _, e := cl.AddLabels(ctx, "1", []string{"a"}); return e },
		"RemoveLabel":       func() error { return cl.RemoveLabel(ctx, "1", "a") },
	}
	for name, fn := range calls {
		if err := fn(); err == nil {
			t.Errorf("%s: expected error on HTTP 500", name)
		}
	}
}

// TestPaginationParams covers the limit/start query-building branches.
func TestPaginationParams(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		w.Write([]byte(`{"results":[],"size":0}`))
	}))
	defer srv.Close()
	cl := testClient(t, srv)
	ctx := context.Background()

	cl.GetChildPages(ctx, "1", 10, 5)
	cl.ListSpaces(ctx, "global", 20, 7)
	cl.GetSpaceRootPages(ctx, "K", 30)

	joined := strings.Join(queries, "\n")
	for _, want := range []string{"limit=10", "start=5", "type=global", "start=7", "depth=root", "limit=30"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in queries:\n%s", want, joined)
		}
	}
}

// TestUpdateExplicitVersionTitle covers Update's non-default branches: an
// explicit version, an explicit title, and the wiki representation.
func TestUpdateExplicitVersionTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"id":"1","type":"page","title":"Old","version":{"number":3}}`))
			return
		}
		w.Write([]byte(`{"id":"1","version":{"number":9}}`))
	}))
	defer srv.Close()

	out, err := testClient(t, srv).Update(context.Background(), UpdateInput{
		ID: "1", Title: "New", Body: "x", Version: 9, Representation: RepresentationWiki,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Version == nil || out.Version.Number != 9 {
		t.Errorf("version = %+v", out.Version)
	}
}
