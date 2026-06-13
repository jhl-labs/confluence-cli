// Package confluence is a thin client for the Confluence Server/Data Center
// REST API (base path /rest/api). It handles auth (Personal Access Token or
// Basic), retries on transient failures, and structured error reporting.
package confluence

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"confluence-cli/internal/config"
)

// Client talks to a single Confluence instance.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authHeader string
	userAgent  string

	// MaxRetries is the number of additional attempts on transient errors.
	MaxRetries int
	// RetryWait is the base backoff between attempts.
	RetryWait time.Duration

	// lastRetryAfter carries a server-provided Retry-After hint into the
	// next backoff. It is only meaningful between attempts of one doJSON call.
	lastRetryAfter time.Duration
}

// APIError is a non-2xx response from Confluence.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("confluence API %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("confluence API %d", e.StatusCode)
}

// New builds a Client from config. It returns an error if the config is
// invalid or has no usable credentials.
func New(cfg config.Config, timeout time.Duration) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	auth, err := authHeader(cfg)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()

	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: &http.Client{Timeout: timeout, Transport: transport},
		authHeader: auth,
		userAgent:  "confluence-cli",
		MaxRetries: 3,
		RetryWait:  500 * time.Millisecond,
	}, nil
}

func authHeader(cfg config.Config) (string, error) {
	if cfg.Token != "" {
		return "Bearer " + cfg.Token, nil
	}
	if cfg.User != "" {
		raw := cfg.User + ":" + cfg.Password
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw)), nil
	}
	return "", errors.New("no credentials configured")
}

// doJSON performs an API request, decoding a JSON response into out (which may
// be nil). path is relative to the API root, e.g. "/rest/api/content".
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			wait := c.backoff(attempt, lastErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}

		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Authorization", c.authHeader)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.MaxRetries {
				continue // network errors are retryable
			}
			return fmt.Errorf("request failed: %w", err)
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out == nil || len(data) == 0 {
				return nil
			}
			if err := json.Unmarshal(data, out); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
			return nil
		}

		apiErr := parseAPIError(resp.StatusCode, data)
		if isRetryable(resp.StatusCode) && attempt < c.MaxRetries {
			lastErr = apiErr
			c.lastRetryAfter = retryAfter(resp.Header)
			continue
		}
		return apiErr
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("request failed after retries")
}

func (c *Client) backoff(attempt int, lastErr error) time.Duration {
	if c.lastRetryAfter > 0 {
		d := c.lastRetryAfter
		c.lastRetryAfter = 0
		return d
	}
	// Exponential backoff: base * 2^(attempt-1).
	return c.RetryWait * time.Duration(1<<(attempt-1))
}

func isRetryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests, // 429
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	}
	return false
}

func retryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return time.Duration(secs) * time.Second
	}
	return 0
}

func parseAPIError(status int, data []byte) *APIError {
	e := &APIError{StatusCode: status, Body: string(data)}
	// Confluence error shape: {"statusCode":404,"message":"..."} or
	// {"message":"...","data":{...}}.
	var parsed struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &parsed) == nil && parsed.Message != "" {
		e.Message = parsed.Message
	} else if len(data) > 0 && len(data) < 300 {
		e.Message = strings.TrimSpace(string(data))
	}
	return e
}
