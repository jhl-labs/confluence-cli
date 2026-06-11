package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDispatch(t *testing.T) {
	if code := run([]string{"version"}); code != 0 {
		t.Errorf("version exit = %d", code)
	}
	if code := run([]string{"help"}); code != 0 {
		t.Errorf("help exit = %d", code)
	}
	if code := run(nil); code != 2 {
		t.Errorf("no-args exit = %d", code)
	}
	if code := run([]string{"bogus-cmd"}); code != 2 {
		t.Errorf("unknown exit = %d", code)
	}
}

func TestRunDispatchCommandError(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CONFLUENCE_BASE_URL", "https://x.example.com")
	t.Setenv("CONFLUENCE_TOKEN", "t")
	// "get" without --id should return exit code 1.
	if code := run([]string{"get"}); code != 1 {
		t.Errorf("expected exit 1 for command error, got %d", code)
	}
}

func TestReadBody(t *testing.T) {
	// literal value
	if got, _ := readBody("inline", ""); got != "inline" {
		t.Errorf("value = %q", got)
	}
	// from file
	f := filepath.Join(t.TempDir(), "b.txt")
	os.WriteFile(f, []byte("file body"), 0o644)
	if got, err := readBody("ignored", f); err != nil || got != "file body" {
		t.Errorf("file = %q err=%v", got, err)
	}
	// missing file
	if _, err := readBody("", filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected error for missing file")
	}
	// from stdin
	old := os.Stdin
	r, w, _ := os.Pipe()
	go func() { w.WriteString("piped"); w.Close() }()
	os.Stdin = r
	got, err := readBody("", "-")
	os.Stdin = old
	if err != nil || got != "piped" {
		t.Errorf("stdin = %q err=%v", got, err)
	}
}

func TestRequireFlag(t *testing.T) {
	if err := requireFlag("id", ""); err == nil {
		t.Error("expected error for empty value")
	}
	if err := requireFlag("id", "x"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmit(t *testing.T) {
	_, err := captureStdout(t, func() error {
		return emit("json", map[string]string{"a": "b"}, func() {})
	})
	if err != nil {
		t.Errorf("json emit err = %v", err)
	}
	called := false
	_, _ = captureStdout(t, func() error {
		return emit("text", nil, func() { called = true })
	})
	if !called {
		t.Error("text emit did not call textFn")
	}
	if err := emit("xml", nil, func() {}); err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestClientNoCreds(t *testing.T) {
	c := &commonFlags{baseURL: "https://x.example.com"} // no token/user
	if _, err := c.client(); err == nil {
		t.Error("expected error building client without credentials")
	}
}

func TestUsageWrites(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "usage")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	usage(f)
	info, _ := f.Stat()
	if info.Size() == 0 {
		t.Error("usage wrote nothing")
	}
}
