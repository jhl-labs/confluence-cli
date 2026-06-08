package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"confluence-cli/internal/config"
	"confluence-cli/internal/confluence"
)

// commonFlags are the connection/auth/output flags shared by every command.
type commonFlags struct {
	baseURL  string
	token    string
	user     string
	password string
	insecure bool
	output   string
	timeout  time.Duration
	retries  int

	// defaultSpace is the CONFLUENCE_SPACE / config default, used as the
	// fallback for the --space flag in commands that accept it.
	defaultSpace string
}

// registerCommon adds the shared flags to fs, defaulting to values loaded from
// the config file and environment. Flags therefore override env/file.
func registerCommon(fs *flag.FlagSet) (*commonFlags, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	c := &commonFlags{defaultSpace: cfg.Space}
	fs.StringVar(&c.baseURL, "base-url", cfg.BaseURL, "Confluence site base URL")
	fs.StringVar(&c.token, "token", cfg.Token, "Personal Access Token (Bearer)")
	fs.StringVar(&c.user, "user", cfg.User, "username for Basic auth")
	fs.StringVar(&c.password, "password", cfg.Password, "password/token for Basic auth")
	fs.BoolVar(&c.insecure, "insecure", cfg.Insecure, "skip TLS certificate verification")
	fs.StringVar(&c.output, "output", "json", "output format: json|text")
	fs.DurationVar(&c.timeout, "timeout", 30*time.Second, "HTTP request timeout")
	fs.IntVar(&c.retries, "retries", 3, "retry attempts on transient (429/5xx) errors")
	return c, nil
}

// client builds a Confluence client from the parsed common flags.
func (c *commonFlags) client() (*confluence.Client, error) {
	cfg := config.Config{
		BaseURL:  c.baseURL,
		Token:    c.token,
		User:     c.user,
		Password: c.password,
		Insecure: c.insecure,
	}
	cl, err := confluence.New(cfg, c.timeout)
	if err != nil {
		return nil, err
	}
	cl.MaxRetries = c.retries
	return cl, nil
}

// readBody resolves page/comment body content from a literal value, a file
// path, or stdin ("-"). file takes precedence over value when both are given.
func readBody(value, file string) (string, error) {
	switch {
	case file == "-":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	case file != "":
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("reading body file: %w", err)
		}
		return string(data), nil
	default:
		return value, nil
	}
}

// fail returns a usage-style error for a missing required flag.
func requireFlag(name, value string) error {
	if value == "" {
		return fmt.Errorf("--%s is required", name)
	}
	return nil
}
