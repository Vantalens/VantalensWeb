package handlers

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

type remoteAdminResult struct {
	StatusCode int
	Response   models.APIResponse
	Duration   time.Duration
	Body       string
}

func remoteAdminConfigured() bool {
	if truthy(config.GetEnv("AUTHORITY_BACKEND", "false")) {
		return false
	}
	return remoteAdminBase() != ""
}

func remoteAdminBase() string {
	return strings.TrimRight(strings.TrimSpace(config.GetEnv("REMOTE_ADMIN_BASE", "")), "/")
}

func localCacheEnabled() bool {
	return !falsy(config.GetEnv("LOCAL_CACHE_ENABLED", "true"))
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func falsy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

func requireAdminAccess(w http.ResponseWriter, r *http.Request) bool {
	cfg := config.GetConfig()
	adminToken := ""
	if cfg != nil {
		adminToken = strings.TrimSpace(cfg.AdminToken)
	}
	if adminToken != "" {
		for _, candidate := range []string{
			strings.TrimSpace(r.Header.Get("X-Admin-Token")),
			strings.TrimSpace(auth.ExtractBearerToken(r)),
		} {
			if candidate != "" && subtle.ConstantTimeCompare([]byte(candidate), []byte(adminToken)) == 1 {
				return true
			}
		}
	}
	return auth.RequireAuth(w, r)
}

func proxyRemoteAdmin(r *http.Request, method, remotePath string, body []byte) (remoteAdminResult, error) {
	base := remoteAdminBase()
	if base == "" {
		return remoteAdminResult{}, fmt.Errorf("REMOTE_ADMIN_BASE is not configured")
	}
	target, err := url.JoinPath(base, remotePath)
	if err != nil {
		return remoteAdminResult{}, err
	}
	if r.URL.RawQuery != "" && !strings.Contains(remotePath, "?") {
		target += "?" + r.URL.RawQuery
	}
	if strings.Contains(remotePath, "?") {
		target = base + remotePath
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(r.Context(), method, target, reader)
	if err != nil {
		return remoteAdminResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := strings.TrimSpace(config.GetEnv("REMOTE_ADMIN_TOKEN", "")); token != "" {
		req.Header.Set("X-Admin-Token", token)
	} else if authHeader := strings.TrimSpace(r.Header.Get("Authorization")); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return remoteAdminResult{Duration: time.Since(start)}, err
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	result := remoteAdminResult{StatusCode: resp.StatusCode, Duration: time.Since(start), Body: string(raw)}
	if readErr != nil {
		return result, readErr
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &result.Response); err != nil {
			return result, fmt.Errorf("remote returned non-json response: http %d", resp.StatusCode)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !result.Response.Success {
		msg := result.Response.Message
		if msg == "" {
			msg = strings.TrimSpace(result.Body)
		}
		if msg == "" {
			msg = "remote request failed"
		}
		return result, fmt.Errorf("remote admin request failed: http %d: %s", resp.StatusCode, msg)
	}
	return result, nil
}

func remoteErrorData(result remoteAdminResult, err error, fallback string) map[string]interface{} {
	data := map[string]interface{}{
		"stage":           "remote_admin_request",
		"http_status":     result.StatusCode,
		"duration":        result.Duration.String(),
		"remote_response": result.Response,
		"fallback":        fallback,
	}
	if err != nil {
		data["error"] = err.Error()
	}
	return data
}
