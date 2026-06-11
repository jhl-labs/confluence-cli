package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// commandCases are representative invocations with all required flags present,
// used to exercise the error and empty-output branches across every command.
func commandCases() []struct {
	name string
	fn   func() error
} {
	return []struct {
		name string
		fn   func() error
	}{
		{"search", func() error { return runSearch([]string{"--text", "x", "--output", "text"}) }},
		{"get", func() error { return runGet([]string{"--id", "1", "--output", "text"}) }},
		{"create", func() error {
			return runCreate([]string{"--space", "E", "--title", "T", "--body", "b", "--output", "text"})
		}},
		{"update", func() error { return runUpdate([]string{"--id", "1", "--body", "b", "--output", "text"}) }},
		{"comment", func() error { return runComment([]string{"--id", "1", "--body", "b", "--output", "text"}) }},
		{"delete", func() error { return runDelete([]string{"--id", "1", "--yes", "--output", "text"}) }},
		{"spaces", func() error { return runSpaces([]string{"--output", "text"}) }},
		{"children", func() error { return runChildren([]string{"--id", "1", "--output", "text"}) }},
		{"tree", func() error { return runTree([]string{"--space", "E", "--output", "text"}) }},
		{"move", func() error { return runMove([]string{"--id", "1", "--parent", "2", "--output", "text"}) }},
		{"labels", func() error { return runLabels([]string{"--id", "1", "--output", "text"}) }},
	}
}

// TestCommandsAPIError verifies every command surfaces a server error.
func TestCommandsAPIError(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"message":"boom"}`)
	})
	for _, c := range commandCases() {
		_, err := captureStdout(t, c.fn)
		if err == nil {
			t.Errorf("%s: expected error on HTTP 500", c.name)
		}
	}
}

// TestCommandsClientError verifies every command fails cleanly when the client
// cannot be built (no base URL configured).
func TestCommandsClientError(t *testing.T) {
	hermeticEnv(t) // base URL intentionally empty
	for _, c := range commandCases() {
		if _, err := captureStdout(t, c.fn); err == nil {
			t.Errorf("%s: expected client-build error with no base URL", c.name)
		}
	}
}

// TestCommandsEmptyText exercises the "no results" text branches.
func TestCommandsEmptyText(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		// An empty list satisfies search/spaces/children/labels/tree-roots.
		io.WriteString(w, `{"results":[],"size":0}`)
	})
	cases := []struct {
		name, want string
		fn         func() error
	}{
		{"search", "no results", func() error { return runSearch([]string{"--text", "x", "--output", "text"}) }},
		{"spaces", "no spaces", func() error { return runSpaces([]string{"--output", "text"}) }},
		{"children", "no child pages", func() error { return runChildren([]string{"--id", "1", "--output", "text"}) }},
		{"labels", "no labels", func() error { return runLabels([]string{"--id", "1", "--output", "text"}) }},
		{"tree", "empty", func() error { return runTree([]string{"--space", "E", "--output", "text"}) }},
	}
	for _, c := range cases {
		out, err := captureStdout(t, c.fn)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: output %q missing %q", c.name, out, c.want)
		}
	}
}
