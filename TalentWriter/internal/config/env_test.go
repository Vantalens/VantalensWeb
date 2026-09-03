package config

import (
	"os"
	"path/filepath"
	"testing"
)

// unsetEnv removes a key for the duration of the test, restoring it afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	orig, had := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, orig)
		}
	})
}

func TestLoadEnvFiles(t *testing.T) {
	for _, key := range []string{"TW_TEST_ENV_PLAIN", "TW_TEST_ENV_QUOTED", "TW_TEST_ENV_SPACED", "TW_TEST_ENV_EXISTING"} {
		unsetEnv(t, key)
	}

	envFile := filepath.Join(t.TempDir(), ".env")
	content := `# a comment line

TW_TEST_ENV_PLAIN=plain-value
TW_TEST_ENV_QUOTED="quoted value"
TW_TEST_ENV_SPACED   =   spaced value
TW_TEST_ENV_EXISTING=from-file
NO_EQUALS_SIGN_LINE
=no-key-value
`
	if err := os.WriteFile(envFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	// Pre-existing values must not be overridden.
	if err := os.Setenv("TW_TEST_ENV_EXISTING", "original"); err != nil {
		t.Fatalf("set existing env: %v", err)
	}

	LoadEnvFiles(envFile)

	cases := []struct {
		key  string
		want string
	}{
		{"TW_TEST_ENV_PLAIN", "plain-value"},
		{"TW_TEST_ENV_QUOTED", "quoted value"},
		{"TW_TEST_ENV_SPACED", "spaced value"},
		{"TW_TEST_ENV_EXISTING", "original"},
	}
	for _, tc := range cases {
		if got := os.Getenv(tc.key); got != tc.want {
			t.Fatalf("os.Getenv(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestLoadEnvFilesMultipleFilesLaterFileDoesNotOverride(t *testing.T) {
	unsetEnv(t, "TW_TEST_ENV_MULTI")

	dir := t.TempDir()
	first := filepath.Join(dir, "first.env")
	second := filepath.Join(dir, "second.env")
	if err := os.WriteFile(first, []byte("TW_TEST_ENV_MULTI=from-first\n"), 0o644); err != nil {
		t.Fatalf("write first env file: %v", err)
	}
	if err := os.WriteFile(second, []byte("TW_TEST_ENV_MULTI=from-second\n"), 0o644); err != nil {
		t.Fatalf("write second env file: %v", err)
	}

	LoadEnvFiles(first, second)
	if got := os.Getenv("TW_TEST_ENV_MULTI"); got != "from-first" {
		t.Fatalf("os.Getenv = %q, want %q (first file wins)", got, "from-first")
	}
}

func TestLoadEnvFilesMissingFileIsIgnored(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.env")
	// Must not panic or set anything.
	LoadEnvFiles(missing)
}

func TestGetEnvAnyFallsThroughEmptyValues(t *testing.T) {
	unsetEnv(t, "TW_TEST_ANY_UNSET")
	t.Setenv("TW_TEST_ANY_BLANK", "  ")

	if got := GetEnvAny([]string{"TW_TEST_ANY_UNSET", "TW_TEST_ANY_BLANK"}, "fallback"); got != "fallback" {
		t.Fatalf("GetEnvAny = %q, want %q", got, "fallback")
	}
}
