package confluence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"confluence-cli/internal/config"
)

func TestRetryAfterHeader(t *testing.T) {
	h := http.Header{}
	if retryAfter(h) != 0 {
		t.Error("empty header should be 0")
	}
	h.Set("Retry-After", "2")
	if retryAfter(h) != 2*time.Second {
		t.Errorf("got %v", retryAfter(h))
	}
	h.Set("Retry-After", "not-a-number")
	if retryAfter(h) != 0 {
		t.Error("non-numeric should be 0")
	}
}

func TestBackoff(t *testing.T) {
	c := &Client{RetryWait: 10 * time.Millisecond}
	// server-provided hint takes precedence and is consumed once
	c.lastRetryAfter = 5 * time.Second
	if d := c.backoff(1, nil); d != 5*time.Second {
		t.Errorf("hinted backoff = %v", d)
	}
	if c.lastRetryAfter != 0 {
		t.Error("hint should reset after use")
	}
	// exponential otherwise
	if d := c.backoff(1, nil); d != 10*time.Millisecond {
		t.Errorf("attempt1 = %v", d)
	}
	if d := c.backoff(3, nil); d != 40*time.Millisecond {
		t.Errorf("attempt3 = %v", d)
	}
}

func TestParseAPIError(t *testing.T) {
	e := parseAPIError(404, []byte(`{"message":"missing"}`))
	if e.Message != "missing" {
		t.Errorf("json message = %q", e.Message)
	}
	e = parseAPIError(400, []byte("plain text error"))
	if e.Message != "plain text error" {
		t.Errorf("plain message = %q", e.Message)
	}
	e = parseAPIError(500, []byte(strings.Repeat("x", 500)))
	if e.Message != "" {
		t.Error("overly long body should not become message")
	}
}

func TestNewInsecure(t *testing.T) {
	cl, err := New(config.Config{BaseURL: "https://x", Token: "t", Insecure: true}, time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if cl == nil {
		t.Fatal("nil client")
	}
}

func TestDoJSONNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // now connections are refused

	cl, err := New(config.Config{BaseURL: url, Token: "t"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cl.RetryWait = time.Millisecond
	cl.MaxRetries = 1
	if _, err := cl.Get(context.Background(), "1", nil); err == nil {
		t.Fatal("expected network error")
	}
}

func TestSearchNoExpand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[],"size":0}`))
	}))
	defer srv.Close()
	if _, err := testClient(t, srv).Search(context.Background(), "x", 0, nil); err != nil {
		t.Fatalf("Search: %v", err)
	}
}
