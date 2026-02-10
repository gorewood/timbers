package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NonexistentFile(t *testing.T) {
	err := Load("/nonexistent/.env")
	if err != nil {
		t.Fatalf("expected nil for nonexistent file, got %v", err)
	}
}

func TestLoad_SetsUnsetVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.local")
	content := "TEST_ENVFILE_A=hello\nTEST_ENVFILE_B=world\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Ensure vars are unset
	t.Setenv("TEST_ENVFILE_A", "")
	t.Setenv("TEST_ENVFILE_B", "")
	_ = os.Unsetenv("TEST_ENVFILE_A") //nolint:errcheck
	_ = os.Unsetenv("TEST_ENVFILE_B") //nolint:errcheck

	if err := Load(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("TEST_ENVFILE_A"); got != "hello" {
		t.Errorf("TEST_ENVFILE_A = %q, want %q", got, "hello")
	}
	if got := os.Getenv("TEST_ENVFILE_B"); got != "world" {
		t.Errorf("TEST_ENVFILE_B = %q, want %q", got, "world")
	}
}

func TestLoad_DoesNotOverrideExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "TEST_ENVFILE_C=from_file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_ENVFILE_C", "from_env")

	if err := Load(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("TEST_ENVFILE_C"); got != "from_env" {
		t.Errorf("TEST_ENVFILE_C = %q, want %q (env should take precedence)", got, "from_env")
	}
}

func TestLoad_SkipsCommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# This is a comment\n\nTEST_ENVFILE_D=yes\n  # indented comment\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_ENVFILE_D", "")
	_ = os.Unsetenv("TEST_ENVFILE_D") //nolint:errcheck

	if err := Load(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("TEST_ENVFILE_D"); got != "yes" {
		t.Errorf("TEST_ENVFILE_D = %q, want %q", got, "yes")
	}
}

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		line    string
		wantKey string
		wantVal string
		wantOK  bool
	}{
		{"KEY=value", "KEY", "value", true},
		{"KEY=\"quoted value\"", "KEY", "quoted value", true},
		{"KEY='single quoted'", "KEY", "single quoted", true},
		{"export KEY=value", "KEY", "value", true},
		{"  KEY = value  ", "KEY", "value", true},
		{"no-equals-sign", "", "", false},
		{"=no-key", "", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		key, val, ok := parseEnvLine(tt.line)
		if ok != tt.wantOK || key != tt.wantKey || val != tt.wantVal {
			t.Errorf("parseEnvLine(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.line, key, val, ok, tt.wantKey, tt.wantVal, tt.wantOK)
		}
	}
}
