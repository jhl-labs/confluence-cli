package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hermeticEnv isolates config discovery and clears all CONFLUENCE_* vars.
func hermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, k := range []string{
		"CONFLUENCE_BASE_URL", "CONFLUENCE_TOKEN", "CONFLUENCE_USER",
		"CONFLUENCE_PASSWORD", "CONFLUENCE_SPACE", "CONFLUENCE_INSECURE", "CONFLUENCE_CONFIG",
	} {
		t.Setenv(k, "")
	}
}

// withServer starts a test server and points the CLI's env at it.
func withServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	hermeticEnv(t)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("CONFLUENCE_BASE_URL", srv.URL)
	t.Setenv("CONFLUENCE_TOKEN", "test-token")
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns output.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := fn()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	return string(data), runErr
}

func mustJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, s)
	}
	return m
}

func TestRunSearchText(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{"id": "1", "title": "Release notes", "_links": map[string]string{"webui": "/x"}}},
			"size":    1,
		})
	})
	out, err := captureStdout(t, func() error {
		return runSearch([]string{"--text", "release", "--space", "ENG", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runSearch: %v", err)
	}
	if !strings.Contains(out, "Release notes") {
		t.Errorf("output missing title:\n%s", out)
	}
}

func TestRunSearchJSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "size": 0})
	})
	out, err := captureStdout(t, func() error {
		return runSearch([]string{"--cql", "type=page", "--output", "json"})
	})
	if err != nil {
		t.Fatalf("runSearch: %v", err)
	}
	if _, ok := mustJSON(t, out)["results"]; !ok {
		t.Errorf("expected results key in JSON")
	}
}

func TestRunSearchNoQuery(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CONFLUENCE_BASE_URL", "https://x.example.com")
	t.Setenv("CONFLUENCE_TOKEN", "t")
	if err := runSearch([]string{}); err == nil {
		t.Fatal("expected error when no query provided")
	}
}

func TestRunGetText(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": "123", "title": "Doc", "version": map[string]int{"number": 2},
		})
	})
	out, err := captureStdout(t, func() error {
		return runGet([]string{"--id", "123", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runGet: %v", err)
	}
	if !strings.Contains(out, "Doc") || !strings.Contains(out, "version") {
		t.Errorf("output = %s", out)
	}
}

func TestRunGetBodyOnly(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": "123", "title": "Doc",
			"body": map[string]any{"storage": map[string]string{"value": "<p>hello</p>", "representation": "storage"}},
		})
	})
	out, err := captureStdout(t, func() error {
		return runGet([]string{"--id", "123", "--body", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runGet: %v", err)
	}
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("body output = %s", out)
	}
}

func TestRunGetMissingID(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CONFLUENCE_BASE_URL", "https://x.example.com")
	t.Setenv("CONFLUENCE_TOKEN", "t")
	if err := runGet([]string{}); err == nil {
		t.Fatal("expected error for missing --id")
	}
}

func TestRunCreate(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "200", "title": "New", "_links": map[string]string{"webui": "/p/200"}})
	})
	out, err := captureStdout(t, func() error {
		return runCreate([]string{"--space", "ENG", "--title", "New", "--body", "<p>x</p>", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out, "created page 200") {
		t.Errorf("output = %s", out)
	}
}

func TestRunCreateBodyFile(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "201"})
	})
	f := filepath.Join(t.TempDir(), "body.xhtml")
	os.WriteFile(f, []byte("<p>from file</p>"), 0o644)
	_, err := captureStdout(t, func() error {
		return runCreate([]string{"--space", "ENG", "--title", "F", "--body-file", f})
	})
	if err != nil {
		t.Fatalf("runCreate: %v", err)
	}
}

func TestRunCreateMissingFlags(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CONFLUENCE_BASE_URL", "https://x.example.com")
	t.Setenv("CONFLUENCE_TOKEN", "t")
	if err := runCreate([]string{"--title", "X"}); err == nil {
		t.Fatal("expected error for missing --space")
	}
}

func TestRunUpdate(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"id": "9", "type": "page", "title": "Old", "version": map[string]int{"number": 1}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "9", "version": map[string]int{"number": 2}})
	})
	out, err := captureStdout(t, func() error {
		return runUpdate([]string{"--id", "9", "--body", "<p>new</p>", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out, "version 2") {
		t.Errorf("output = %s", out)
	}
}

func TestRunComment(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "c1"})
	})
	out, err := captureStdout(t, func() error {
		return runComment([]string{"--id", "9", "--body", "<p>hi</p>", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runComment: %v", err)
	}
	if !strings.Contains(out, "added comment c1") {
		t.Errorf("output = %s", out)
	}
}

func TestRunCommentEmpty(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CONFLUENCE_BASE_URL", "https://x.example.com")
	t.Setenv("CONFLUENCE_TOKEN", "t")
	if err := runComment([]string{"--id", "9"}); err == nil {
		t.Fatal("expected error for empty comment body")
	}
}

func TestRunDelete(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	out, err := captureStdout(t, func() error {
		return runDelete([]string{"--id", "9", "--yes", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out, "deleted page 9") {
		t.Errorf("output = %s", out)
	}
}

func TestRunSpaces(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{"key": "ENG", "name": "Eng", "type": "global"}},
			"size":    1,
		})
	})
	out, err := captureStdout(t, func() error {
		return runSpaces([]string{"--output", "text"})
	})
	if err != nil {
		t.Fatalf("runSpaces: %v", err)
	}
	if !strings.Contains(out, "ENG") {
		t.Errorf("output = %s", out)
	}
}

func TestRunChildren(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{"id": "2", "title": "Child"}}, "size": 1,
		})
	})
	out, err := captureStdout(t, func() error {
		return runChildren([]string{"--id", "1", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runChildren: %v", err)
	}
	if !strings.Contains(out, "Child") {
		t.Errorf("output = %s", out)
	}
}

func TestRunTreeBySpace(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/content/page"): // space root pages
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": "10", "title": "Root"}}, "size": 1})
		case r.URL.Path == "/rest/api/content/10/child/page":
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": "11", "title": "Leaf"}}, "size": 1})
		default:
			json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "size": 0})
		}
	})
	out, err := captureStdout(t, func() error {
		return runTree([]string{"--space", "ENG", "--depth", "3", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runTree: %v", err)
	}
	if !strings.Contains(out, "Root") || !strings.Contains(out, "Leaf") {
		t.Errorf("tree output = %s", out)
	}
}

func TestRunTreeByID(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/content/10":
			json.NewEncoder(w).Encode(map[string]any{"id": "10", "title": "Root"})
		default: // children
			json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "size": 0})
		}
	})
	out, err := captureStdout(t, func() error {
		return runTree([]string{"--id", "10", "--output", "json"})
	})
	if err != nil {
		t.Fatalf("runTree: %v", err)
	}
	if !strings.Contains(out, "Root") {
		t.Errorf("tree output = %s", out)
	}
}

func TestRunTreeNoTarget(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CONFLUENCE_BASE_URL", "https://x.example.com")
	t.Setenv("CONFLUENCE_TOKEN", "t")
	if err := runTree([]string{}); err == nil {
		t.Fatal("expected error when neither --id nor --space given")
	}
}

func TestRunMove(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"id": "7", "type": "page", "title": "M", "version": map[string]int{"number": 1}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "7", "version": map[string]int{"number": 2}})
	})
	out, err := captureStdout(t, func() error {
		return runMove([]string{"--id", "7", "--parent", "9", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runMove: %v", err)
	}
	if !strings.Contains(out, "under parent 9") {
		t.Errorf("output = %s", out)
	}
}

func TestRunMoveMissingParent(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CONFLUENCE_BASE_URL", "https://x.example.com")
	t.Setenv("CONFLUENCE_TOKEN", "t")
	if err := runMove([]string{"--id", "7"}); err == nil {
		t.Fatal("expected error for missing --parent")
	}
}

func TestRunLabelsAddRemove(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"name": "docs"}}, "size": 1})
	})
	out, err := captureStdout(t, func() error {
		return runLabels([]string{"--id", "1", "--add", "docs", "--remove", "draft", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runLabels: %v", err)
	}
	if !strings.Contains(out, "docs") {
		t.Errorf("output = %s", out)
	}
}

func TestRunGenerateSkillStdout(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runGenerateSkill([]string{"claude", "--stdout"})
	})
	if err != nil {
		t.Fatalf("runGenerateSkill: %v", err)
	}
	if !strings.HasPrefix(out, "---\nname: confluence-cli") {
		t.Errorf("unexpected skill output start:\n%.60s", out)
	}
}

func TestRunGenerateSkillFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "skill.md")
	if err := runGenerateSkill([]string{"--out", out}); err != nil {
		t.Fatalf("runGenerateSkill: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil || !strings.Contains(string(data), "confluence-cli") {
		t.Errorf("file not written correctly: %v", err)
	}
}

func TestRunGenerateSkillUnknownFlavor(t *testing.T) {
	if err := runGenerateSkill([]string{"bogus", "--stdout"}); err == nil {
		t.Fatal("expected error for unknown flavor")
	}
}
