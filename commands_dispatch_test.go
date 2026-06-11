package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// universalHandler answers any Confluence request with a minimal valid body,
// so the top-level run() dispatch can be exercised for every command.
func universalHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		p := r.URL.Path
		switch {
		case strings.HasSuffix(p, "/search"),
			strings.HasSuffix(p, "/child/page"),
			strings.HasSuffix(p, "/content/page"),
			strings.HasSuffix(p, "/label"),
			p == "/rest/api/space":
			io.WriteString(w, `{"results":[],"size":0}`)
		case strings.HasPrefix(p, "/rest/api/space/"):
			io.WriteString(w, `{"key":"E","name":"E"}`)
		default:
			io.WriteString(w, `{"id":"1","type":"page","title":"T","version":{"number":1}}`)
		}
	case http.MethodPost:
		io.WriteString(w, `{"id":"1"}`)
	case http.MethodPut:
		io.WriteString(w, `{"id":"1","version":{"number":2}}`)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	}
}

func TestRunAllCommandsDispatch(t *testing.T) {
	withServer(t, universalHandler)
	argvs := [][]string{
		{"search", "--text", "x"},
		{"get", "--id", "1"},
		{"create", "--space", "E", "--title", "T", "--body", "b"},
		{"update", "--id", "1", "--body", "b"},
		{"comment", "--id", "1", "--body", "b"},
		{"delete", "--id", "1", "--yes"},
		{"spaces"},
		{"tree", "--space", "E"},
		{"children", "--id", "1"},
		{"move", "--id", "1", "--parent", "2"},
		{"labels", "--id", "1"},
		{"generate-skill", "--stdout"},
	}
	for _, argv := range argvs {
		code, _ := captureStdoutCode(t, func() int { return run(argv) })
		if code != 0 {
			t.Errorf("run(%v) exit = %d", argv, code)
		}
	}
}

// captureStdoutCode is like captureStdout but for a func returning an int.
func captureStdoutCode(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := fn()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	return code, string(data)
}

func feedStdin(t *testing.T, s string) {
	t.Helper()
	old := os.Stdin
	r, w, _ := os.Pipe()
	go func() { io.WriteString(w, s); w.Close() }()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
}

func TestRunDeleteConfirmYes(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			io.WriteString(w, `{"id":"9","title":"Doomed"}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	feedStdin(t, "yes\n")
	out, err := captureStdout(t, func() error {
		return runDelete([]string{"--id", "9", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out, "deleted page 9") {
		t.Errorf("output = %s", out)
	}
}

func TestRunDeleteAbort(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":"9","title":"Doomed"}`)
	})
	feedStdin(t, "no\n")
	if err := runDelete([]string{"--id", "9"}); err == nil {
		t.Fatal("expected abort error when not confirmed")
	}
}

func TestRunGenerateSkillExistsAndForce(t *testing.T) {
	out := filepath.Join(t.TempDir(), "s.md")
	if err := os.WriteFile(out, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGenerateSkill([]string{"--out", out}); err == nil {
		t.Error("expected error: file exists without --force")
	}
	if err := runGenerateSkill([]string{"--out", out, "--force"}); err != nil {
		t.Errorf("with --force: %v", err)
	}
}

func TestRunTreeTruncated(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/content/page"):
			io.WriteString(w, `{"results":[{"id":"10","title":"Root"}],"size":1}`)
		case r.URL.Path == "/rest/api/content/10/child/page":
			// Size exceeds returned results -> truncated.
			io.WriteString(w, `{"results":[{"id":"11","title":"Leaf"}],"size":5}`)
		default:
			io.WriteString(w, `{"results":[],"size":0}`)
		}
	})
	out, err := captureStdout(t, func() error {
		return runTree([]string{"--space", "E", "--output", "text"})
	})
	if err != nil {
		t.Fatalf("runTree: %v", err)
	}
	if !strings.Contains(out, "omitted") {
		t.Errorf("expected truncation note, got:\n%s", out)
	}
}
