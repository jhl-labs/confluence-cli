package config

import (
	"os"
	"path/filepath"
	"testing"
)

// clearEnv blanks every CONFLUENCE_* var and points config discovery at an
// empty temp dir so tests are hermetic.
func clearEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	for _, k := range []string{EnvBaseURL, EnvToken, EnvUser, EnvPassword, EnvSpace, EnvInsecure, EnvConfig} {
		t.Setenv(k, "")
	}
	return dir
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv(EnvBaseURL, "https://confluence.example.com/")
	t.Setenv(EnvToken, "tok")
	t.Setenv(EnvSpace, "DOCS")
	t.Setenv(EnvInsecure, "true")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.BaseURL != "https://confluence.example.com" { // trailing slash trimmed
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.Token != "tok" || c.Space != "DOCS" || !c.Insecure {
		t.Errorf("unexpected config: %+v", c)
	}
}

func TestLoadFromFileWithEnvOverride(t *testing.T) {
	dir := clearEnv(t)
	cfgDir := filepath.Join(dir, "confluence-cli")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"base_url":"https://file.example.com","token":"filetok","user":"alice","space":"FILE"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Env overrides token only; file supplies the rest.
	t.Setenv(EnvToken, "envtok")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.BaseURL != "https://file.example.com" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.Token != "envtok" {
		t.Errorf("Token = %q, want env override", c.Token)
	}
	if c.User != "alice" || c.Space != "FILE" {
		t.Errorf("file values lost: %+v", c)
	}
}

func TestLoadConfigEnvPath(t *testing.T) {
	clearEnv(t)
	f := filepath.Join(t.TempDir(), "custom.json")
	if err := os.WriteFile(f, []byte(`{"base_url":"https://custom.example.com","token":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvConfig, f)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.BaseURL != "https://custom.example.com" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
}

func TestLoadBadConfigFile(t *testing.T) {
	clearEnv(t)
	f := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(f, []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvConfig, f)

	if _, err := Load(); err == nil {
		t.Fatal("expected error for malformed config")
	}
}

func TestLoadConfigPathIsDirectory(t *testing.T) {
	clearEnv(t)
	t.Setenv(EnvConfig, t.TempDir()) // a directory, not a file -> read error
	if _, err := Load(); err == nil {
		t.Fatal("expected error when config path is a directory")
	}
}

func TestLoadNoHomeNoXDG(t *testing.T) {
	// Clear everything including HOME/XDG so config-dir discovery fails and the
	// fallback branches in defaultConfigDir/configFilePath run.
	for _, k := range []string{EnvBaseURL, EnvToken, EnvUser, EnvPassword, EnvSpace, EnvInsecure, EnvConfig} {
		t.Setenv(k, "")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	// Either Load returns an error (no config dir resolvable) or it succeeds
	// with no file; both exercise the fallback path without panicking.
	_, _ = Load()
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"ok token", Config{BaseURL: "https://x", Token: "t"}, false},
		{"ok basic", Config{BaseURL: "http://x", User: "u"}, false},
		{"no base", Config{Token: "t"}, true},
		{"bad scheme", Config{BaseURL: "ftp://x", Token: "t"}, true},
		{"no creds", Config{BaseURL: "https://x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTruthy(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on"} {
		if !truthy(v) {
			t.Errorf("truthy(%q) = false", v)
		}
	}
	for _, v := range []string{"", "0", "false", "no", "nope"} {
		if truthy(v) {
			t.Errorf("truthy(%q) = true", v)
		}
	}
}
