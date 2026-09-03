package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	Port     = 9090
	HTMLPort = 1313
)

const (
	MaxCommentNameLen    = 50
	MaxCommentEmailLen   = 100
	MaxCommentContentLen = 2000
	MaxCommentImages     = 5
	MaxImageSize         = 5 << 20
)

const (
	MaxEmailRetries  = 3
	EmailQueueSize   = 100
	EmailWorkerCount = 2
)

type Config struct {
	HugoPath      string
	HugoURL       string
	AdminUsername string
	AdminToken    string
	JWTSecret     []byte
}

var (
	Instance *Config
	configMu sync.RWMutex
)

func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return Instance
}

func SetConfig(cfg *Config) {
	configMu.Lock()
	Instance = cfg
	configMu.Unlock()
}

func GetEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func GetCommentSettingsPath(hugoPath string) string {
	if v := strings.TrimSpace(os.Getenv("COMMENT_SETTINGS_PATH")); v != "" {
		return v
	}
	return filepath.Join(hugoPath, "config", "comment_settings.json")
}

func GetAnalyticsDBPath(hugoPath string) string {
	if v := strings.TrimSpace(os.Getenv("ANALYTICS_DB_PATH")); v != "" {
		return v
	}
	return filepath.Join(hugoPath, ".talentwriter", "analytics", "visits.db")
}

func GetCommentsDBPath(hugoPath string) string {
	if v := strings.TrimSpace(os.Getenv("COMMENTS_DB_PATH")); v != "" {
		return v
	}
	return filepath.Join(hugoPath, ".talentwriter", "comments", "comments.db")
}

func GetArticlesDBPath(hugoPath string) string {
	if v := strings.TrimSpace(os.Getenv("ARTICLES_DB_PATH")); v != "" {
		return v
	}
	return filepath.Join(hugoPath, ".talentwriter", "articles", "articles.db")
}

func ResolveHugoPath(base string) string {
	base = filepath.Clean(base)
	candidates := []string{base}
	parent := filepath.Clean(filepath.Join(base, ".."))
	if parent != base {
		candidates = append(candidates, parent)
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "content")); err == nil && info.IsDir() {
			if abs, absErr := filepath.Abs(candidate); absErr == nil {
				return abs
			}
			return candidate
		}
	}

	if abs, err := filepath.Abs(base); err == nil {
		return abs
	}
	return base
}

func LocalhostURL(port int, path string) string {
	if port <= 0 {
		port = 80
	}
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	return "http://localhost:" + strconv.Itoa(port) + path
}
