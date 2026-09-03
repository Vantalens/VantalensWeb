package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Run("set value wins", func(t *testing.T) {
		t.Setenv("TW_TEST_GETENV_SET", "from-env")
		if got := GetEnv("TW_TEST_GETENV_SET", "fallback"); got != "from-env" {
			t.Fatalf("GetEnv = %q, want %q", got, "from-env")
		}
	})

	t.Run("unset returns default", func(t *testing.T) {
		if got := GetEnv("TW_TEST_GETENV_MISSING", "fallback"); got != "fallback" {
			t.Fatalf("GetEnv = %q, want %q", got, "fallback")
		}
	})

	t.Run("empty string counts as unset", func(t *testing.T) {
		t.Setenv("TW_TEST_GETENV_EMPTY", "")
		if got := GetEnv("TW_TEST_GETENV_EMPTY", "fallback"); got != "fallback" {
			t.Fatalf("GetEnv = %q, want %q for empty env value", got, "fallback")
		}
	})
}

func TestGetEnvAny(t *testing.T) {
	t.Run("first non-empty wins", func(t *testing.T) {
		t.Setenv("TW_TEST_ANY_A", "")
		t.Setenv("TW_TEST_ANY_B", "second")
		t.Setenv("TW_TEST_ANY_C", "third")
		if got := GetEnvAny([]string{"TW_TEST_ANY_A", "TW_TEST_ANY_B", "TW_TEST_ANY_C"}, "def"); got != "second" {
			t.Fatalf("GetEnvAny = %q, want %q", got, "second")
		}
	})

	t.Run("whitespace-only treated as empty", func(t *testing.T) {
		t.Setenv("TW_TEST_ANY_WS", "   ")
		if got := GetEnvAny([]string{"TW_TEST_ANY_WS"}, "def"); got != "def" {
			t.Fatalf("GetEnvAny = %q, want default for whitespace-only value", got)
		}
	})

	t.Run("all missing returns default", func(t *testing.T) {
		if got := GetEnvAny([]string{"TW_TEST_ANY_X", "TW_TEST_ANY_Y"}, "def"); got != "def" {
			t.Fatalf("GetEnvAny = %q, want %q", got, "def")
		}
	})

	t.Run("empty key list returns default", func(t *testing.T) {
		if got := GetEnvAny(nil, "def"); got != "def" {
			t.Fatalf("GetEnvAny = %q, want %q", got, "def")
		}
	})
}

func TestSetAndGetConfig(t *testing.T) {
	t.Cleanup(func() { SetConfig(nil) })

	SetConfig(nil)
	if GetConfig() != nil {
		t.Fatal("GetConfig after SetConfig(nil) must be nil")
	}

	cfg := &Config{HugoPath: "site", AdminUsername: "admin", AdminToken: "tok"}
	SetConfig(cfg)
	got := GetConfig()
	if got != cfg {
		t.Fatalf("GetConfig = %+v, want the exact pointer set via SetConfig", got)
	}
}

func TestCommentSettingsPath(t *testing.T) {
	t.Run("default under hugo path", func(t *testing.T) {
		got := GetCommentSettingsPath("/site")
		want := filepath.Join("/site", "config", "comment_settings.json")
		if got != want {
			t.Fatalf("GetCommentSettingsPath = %q, want %q", got, want)
		}
	})

	t.Run("env override wins", func(t *testing.T) {
		t.Setenv("COMMENT_SETTINGS_PATH", `D:\custom\settings.json`)
		if got := GetCommentSettingsPath("/site"); got != `D:\custom\settings.json` {
			t.Fatalf("GetCommentSettingsPath = %q, want env override", got)
		}
	})
}

func TestDatabasePathGetters(t *testing.T) {
	cases := []struct {
		name   string
		envKey string
		getter func(string) string
		rel    []string
	}{
		{"analytics", "ANALYTICS_DB_PATH", GetAnalyticsDBPath, []string{".talentwriter", "analytics", "visits.db"}},
		{"comments", "COMMENTS_DB_PATH", GetCommentsDBPath, []string{".talentwriter", "comments", "comments.db"}},
		{"articles", "ARTICLES_DB_PATH", GetArticlesDBPath, []string{".talentwriter", "articles", "articles.db"}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" default", func(t *testing.T) {
			want := filepath.Join(append([]string{"/site"}, tc.rel...)...)
			if got := tc.getter("/site"); got != want {
				t.Fatalf("getter = %q, want %q", got, want)
			}
		})
		t.Run(tc.name+" env override", func(t *testing.T) {
			t.Setenv(tc.envKey, `D:\tmp\override.db`)
			if got := tc.getter("/site"); got != `D:\tmp\override.db` {
				t.Fatalf("getter = %q, want env override", got)
			}
		})
		t.Run(tc.name+" blank env ignored", func(t *testing.T) {
			t.Setenv(tc.envKey, "   ")
			want := filepath.Join(append([]string{"/site"}, tc.rel...)...)
			if got := tc.getter("/site"); got != want {
				t.Fatalf("getter = %q, want default when env is blank", got)
			}
		})
	}
}

func TestResolveHugoPath(t *testing.T) {
	t.Run("base with content dir", func(t *testing.T) {
		base := t.TempDir()
		if err := os.MkdirAll(filepath.Join(base, "content"), 0o755); err != nil {
			t.Fatalf("create content dir: %v", err)
		}
		got := ResolveHugoPath(base)
		if !samePath(t, got, base) {
			t.Fatalf("ResolveHugoPath = %q, want abs of %q", got, base)
		}
	})

	t.Run("falls back to parent", func(t *testing.T) {
		parent := t.TempDir()
		if err := os.MkdirAll(filepath.Join(parent, "content"), 0o755); err != nil {
			t.Fatalf("create content dir: %v", err)
		}
		child := filepath.Join(parent, "subdir")
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatalf("create child dir: %v", err)
		}
		got := ResolveHugoPath(child)
		if !samePath(t, got, parent) {
			t.Fatalf("ResolveHugoPath(%q) = %q, want parent %q", child, got, parent)
		}
	})

	t.Run("no content dir returns absolute base", func(t *testing.T) {
		base := t.TempDir()
		got := ResolveHugoPath(base)
		if !filepath.IsAbs(got) {
			t.Fatalf("ResolveHugoPath = %q, want absolute path", got)
		}
		if !samePath(t, got, base) {
			t.Fatalf("ResolveHugoPath = %q, want abs of %q", got, base)
		}
	})

	t.Run("content as file does not count", func(t *testing.T) {
		base := t.TempDir()
		contentFile := filepath.Join(base, "content")
		if err := os.WriteFile(contentFile, []byte("not a dir"), 0o644); err != nil {
			t.Fatalf("create content file: %v", err)
		}
		// base has no content *directory* and neither does its parent,
		// so the result must fall back to the absolute base path.
		got := ResolveHugoPath(base)
		if !samePath(t, got, base) {
			t.Fatalf("ResolveHugoPath = %q, want abs of %q", got, base)
		}
	})
}

func TestLocalhostURL(t *testing.T) {
	cases := []struct {
		name string
		port int
		path string
		want string
	}{
		{"normal", 9090, "/api", "http://localhost:9090/api"},
		{"path without leading slash", 1313, "posts", "http://localhost:1313/posts"},
		{"empty path becomes root", 1313, "", "http://localhost:1313/"},
		{"zero port becomes 80", 0, "/", "http://localhost:80/"},
		{"negative port becomes 80", -1, "x", "http://localhost:80/x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LocalhostURL(tc.port, tc.path); got != tc.want {
				t.Fatalf("LocalhostURL(%d, %q) = %q, want %q", tc.port, tc.path, got, tc.want)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Guard the public limits against accidental changes.
	if Port != 9090 || HTMLPort != 1313 {
		t.Fatalf("ports changed: Port=%d HTMLPort=%d", Port, HTMLPort)
	}
	if MaxCommentNameLen != 50 || MaxCommentEmailLen != 100 || MaxCommentContentLen != 2000 {
		t.Fatalf("comment length limits changed: %d %d %d", MaxCommentNameLen, MaxCommentEmailLen, MaxCommentContentLen)
	}
	if MaxCommentImages != 5 || MaxImageSize != 5<<20 {
		t.Fatalf("image limits changed: %d %d", MaxCommentImages, MaxImageSize)
	}
	if MaxEmailRetries != 3 || EmailQueueSize != 100 || EmailWorkerCount != 2 {
		t.Fatalf("email queue constants changed: %d %d %d", MaxEmailRetries, EmailQueueSize, EmailWorkerCount)
	}
}

// samePath reports whether got resolves to the same location as want,
// tolerating symlink-resolution differences (e.g. macOS /var vs /private/var).
func samePath(t *testing.T, got, want string) bool {
	t.Helper()
	wantAbs, err := filepath.Abs(want)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", want, err)
	}
	if got == wantAbs {
		return true
	}
	resolvedWant, err := filepath.EvalSymlinks(wantAbs)
	if err != nil {
		return false
	}
	resolvedGot, err := filepath.EvalSymlinks(got)
	if err != nil {
		return false
	}
	return strings.EqualFold(resolvedGot, resolvedWant)
}
