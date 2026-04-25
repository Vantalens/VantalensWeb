package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

type geoRecord struct {
	Country string
	Region  string
	City    string
}

var (
	db   *sql.DB
	dbMu sync.RWMutex
)

func Init(hugoPath string) error {
	path := config.GetAnalyticsDBPath(hugoPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if err := ensureSchema(conn); err != nil {
		_ = conn.Close()
		return err
	}

	dbMu.Lock()
	if db != nil {
		_ = db.Close()
	}
	db = conn
	dbMu.Unlock()
	return nil
}

func ensureSchema(conn *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS visits (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL,
	page_path TEXT NOT NULL,
	page_title TEXT NOT NULL,
	referrer TEXT NOT NULL,
	ip TEXT NOT NULL,
	device_type TEXT NOT NULL,
	browser TEXT NOT NULL,
	os_name TEXT NOT NULL,
	region TEXT NOT NULL,
	country TEXT NOT NULL,
	city TEXT NOT NULL,
	dns_host TEXT NOT NULL,
	language TEXT NOT NULL,
	timezone TEXT NOT NULL,
	screen TEXT NOT NULL,
	user_agent TEXT NOT NULL,
	webrtc_json TEXT NOT NULL,
	is_page_view INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_visits_created_at ON visits(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_visits_page_path ON visits(page_path);
CREATE INDEX IF NOT EXISTS idx_visits_ip ON visits(ip);
CREATE INDEX IF NOT EXISTS idx_visits_session ON visits(session_id);

CREATE TABLE IF NOT EXISTS geo_cache (
	ip TEXT PRIMARY KEY,
	country TEXT NOT NULL,
	region TEXT NOT NULL,
	city TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`
	_, err := conn.Exec(schema)
	return err
}

func TrackVisit(r *http.Request, payload models.AnalyticsCollectRequest) (*models.VisitRecord, error) {
	conn, err := getDB()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ip := extractIP(r)
	if ip == "" {
		ip = "unknown"
	}

	deviceType, browser, osName := parseUserAgent(r.UserAgent())
	geo := lookupGeo(conn, ip)

	if payload.SessionID == "" {
		payload.SessionID = fmt.Sprintf("anon-%d", time.Now().UnixNano())
	}
	payload.Path = normalizePagePath(payload.Path)
	if !isTrackablePage(payload.Path) {
		return &models.VisitRecord{
			SessionID: payload.SessionID,
			Path:      payload.Path,
			Title:     payload.Title,
			IP:        ip,
			CreatedAt: now,
			PageView:  false,
		}, nil
	}
	payload.Title = normalizeTitle(payload.Title, payload.Path)
	payload.Referrer = normalizeReferrer(payload.Referrer)
	payload.PageView = true
	if payload.Title == "" {
		payload.Title = payload.Path
	}

	webrtcJSON := "{}"
	if payload.WebRTC != nil {
		if data, marshalErr := json.Marshal(payload.WebRTC); marshalErr == nil {
			webrtcJSON = string(data)
		}
	}

	result, err := conn.Exec(`
INSERT INTO visits (
	session_id, page_path, page_title, referrer, ip, device_type, browser, os_name,
	region, country, city, dns_host, language, timezone, screen, user_agent,
	webrtc_json, is_page_view, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		limit(payload.SessionID, 96),
		limit(payload.Path, 512),
		limit(payload.Title, 300),
		limit(payload.Referrer, 512),
		limit(ip, 64),
		limit(deviceType, 32),
		limit(browser, 64),
		limit(osName, 64),
		limit(geo.Region, 64),
		limit(geo.Country, 64),
		limit(geo.City, 64),
		limit(payload.DNSHost, 255),
		limit(payload.Language, 32),
		limit(payload.Timezone, 64),
		limit(payload.Screen, 32),
		limit(r.UserAgent(), 512),
		webrtcJSON,
		boolToInt(payload.PageView),
		now,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &models.VisitRecord{
		ID:        id,
		SessionID: payload.SessionID,
		Path:      payload.Path,
		Title:     payload.Title,
		Referrer:  payload.Referrer,
		IP:        ip,
		Device:    deviceType,
		Browser:   browser,
		OS:        osName,
		Region:    geo.Region,
		Country:   geo.Country,
		City:      geo.City,
		DNSHost:   payload.DNSHost,
		Language:  payload.Language,
		Timezone:  payload.Timezone,
		Screen:    payload.Screen,
		UserAgent: r.UserAgent(),
		WebRTC:    payload.WebRTC,
		CreatedAt: now,
		PageView:  payload.PageView,
	}, nil
}

func GetSiteStatistics(limit int) (models.SiteStatistics, error) {
	conn, err := getDB()
	if err != nil {
		return models.SiteStatistics{}, err
	}
	if limit <= 0 {
		limit = 100
	}

	stats := models.SiteStatistics{}
	_ = conn.QueryRow(`SELECT COUNT(*) FROM visits WHERE is_page_view = 1`).Scan(&stats.TotalViews)
	_ = conn.QueryRow(`SELECT COUNT(DISTINCT ip) FROM visits WHERE is_page_view = 1`).Scan(&stats.UniqueIPs)
	_ = conn.QueryRow(`SELECT COUNT(DISTINCT session_id) FROM visits WHERE is_page_view = 1`).Scan(&stats.UniqueSessions)
	_ = conn.QueryRow(`SELECT COUNT(DISTINCT page_path) FROM visits WHERE is_page_view = 1`).Scan(&stats.TotalPages)

	pageRows, err := conn.Query(`
SELECT page_path, MAX(page_title), COUNT(*) AS views, COUNT(DISTINCT ip) AS uv, MAX(created_at) AS last_seen
FROM visits
WHERE is_page_view = 1
GROUP BY page_path
ORDER BY views DESC, last_seen DESC
LIMIT ?`, limit)
	if err != nil {
		return stats, err
	}
	defer pageRows.Close()
	for pageRows.Next() {
		var item models.PageStatistics
		if err := pageRows.Scan(&item.Path, &item.Title, &item.Views, &item.UV, &item.LastSeen); err == nil {
			stats.Pages = append(stats.Pages, item)
		}
	}

	visitorRows, err := conn.Query(`
SELECT ip, COUNT(*) AS visit_count, MIN(created_at) AS first_seen, MAX(created_at) AS last_seen,
       MAX(region) AS region, MAX(device_type) AS device
FROM visits
WHERE is_page_view = 1
GROUP BY ip
ORDER BY visit_count DESC, last_seen DESC
LIMIT 100`)
	if err != nil {
		return stats, err
	}
	defer visitorRows.Close()
	for visitorRows.Next() {
		var item models.VisitorIP
		if err := visitorRows.Scan(&item.IP, &item.VisitCount, &item.FirstSeen, &item.LastSeen, &item.Region, &item.Device); err == nil {
			stats.Visitors = append(stats.Visitors, item)
		}
	}

	recentRows, err := conn.Query(`
SELECT id, session_id, page_path, page_title, referrer, ip, device_type, browser, os_name,
       region, country, city, dns_host, language, timezone, screen, user_agent, webrtc_json, is_page_view, created_at
FROM visits
WHERE is_page_view = 1
ORDER BY id DESC
LIMIT ?`, limit)
	if err != nil {
		return stats, err
	}
	defer recentRows.Close()
	for recentRows.Next() {
		var item models.VisitRecord
		var webrtcJSON string
		var isPageView int
		if err := recentRows.Scan(
			&item.ID, &item.SessionID, &item.Path, &item.Title, &item.Referrer, &item.IP, &item.Device, &item.Browser, &item.OS,
			&item.Region, &item.Country, &item.City, &item.DNSHost, &item.Language, &item.Timezone, &item.Screen,
			&item.UserAgent, &webrtcJSON, &isPageView, &item.CreatedAt,
		); err == nil {
			item.PageView = isPageView == 1
			if webrtcJSON != "" && webrtcJSON != "{}" {
				var report models.WebRTCReport
				if jsonErr := json.Unmarshal([]byte(webrtcJSON), &report); jsonErr == nil {
					item.WebRTC = &report
				}
			}
			stats.RecentVisits = append(stats.RecentVisits, item)
		}
	}

	return stats, nil
}

func getDB() (*sql.DB, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	if db == nil {
		return nil, errors.New("analytics database not initialized")
	}
	return db, nil
}

func extractIP(r *http.Request) string {
	for _, key := range []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
		raw := strings.TrimSpace(r.Header.Get(key))
		if raw == "" {
			continue
		}
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			ip := cleanIP(part)
			if ip != "" {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return cleanIP(host)
	}
	return cleanIP(r.RemoteAddr)
}

func cleanIP(raw string) string {
	raw = strings.TrimSpace(strings.Trim(raw, "[]"))
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func normalizePagePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/"
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		if req, err := http.NewRequest(http.MethodGet, raw, nil); err == nil && req.URL != nil {
			raw = req.URL.Path
		}
	}
	if idx := strings.IndexAny(raw, "?#"); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.ReplaceAll(raw, "\\", "/")
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	for strings.Contains(raw, "//") {
		raw = strings.ReplaceAll(raw, "//", "/")
	}
	if raw != "/" {
		raw = strings.TrimRight(raw, "/")
	}
	return raw
}

func isTrackablePage(path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	if path == "" || path == "/" {
		return true
	}
	blockedPrefixes := []string{
		"/api",
		"/platform",
		"/preview",
		"/scss",
		"/css",
		"/js",
		"/img",
		"/images",
		"/favicon",
		"/robots.txt",
		"/sitemap",
		"/index.xml",
	}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	blockedSuffixes := []string{".css", ".js", ".map", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".woff", ".woff2", ".ttf"}
	for _, suffix := range blockedSuffixes {
		if strings.HasSuffix(path, suffix) {
			return false
		}
	}
	return true
}

func normalizeTitle(title, path string) string {
	title = strings.TrimSpace(title)
	title = strings.TrimSuffix(title, " | Vantalens")
	title = strings.TrimSuffix(title, " - Vantalens")
	if title != "" {
		return limit(title, 300)
	}
	if path == "" || path == "/" {
		return "Home"
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return limit(strings.ReplaceAll(parts[len(parts)-1], "-", " "), 300)
}

func normalizeReferrer(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if req, err := http.NewRequest(http.MethodGet, raw, nil); err == nil && req.URL != nil {
		if req.URL.Host != "" {
			return limit(req.URL.Host+normalizePagePath(req.URL.Path), 512)
		}
		return limit(normalizePagePath(req.URL.Path), 512)
	}
	return limit(raw, 512)
}

func parseUserAgent(ua string) (deviceType string, browser string, osName string) {
	lower := strings.ToLower(ua)
	deviceType = "desktop"
	browser = "unknown"
	osName = "unknown"

	switch {
	case strings.Contains(lower, "iphone"), strings.Contains(lower, "android") && strings.Contains(lower, "mobile"):
		deviceType = "mobile"
	case strings.Contains(lower, "ipad"), strings.Contains(lower, "tablet"):
		deviceType = "tablet"
	case strings.Contains(lower, "bot"), strings.Contains(lower, "spider"), strings.Contains(lower, "crawl"):
		deviceType = "bot"
	}

	switch {
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "chrome/") && !strings.Contains(lower, "edg/"):
		browser = "Chrome"
	case strings.Contains(lower, "safari/") && !strings.Contains(lower, "chrome/"):
		browser = "Safari"
	case strings.Contains(lower, "firefox/"):
		browser = "Firefox"
	case strings.Contains(lower, "qqbrowser"):
		browser = "QQBrowser"
	case strings.Contains(lower, "micromessenger"):
		browser = "WeChat"
	}

	switch {
	case strings.Contains(lower, "windows nt"):
		osName = "Windows"
	case strings.Contains(lower, "mac os x"), strings.Contains(lower, "macintosh"):
		osName = "macOS"
	case strings.Contains(lower, "android"):
		osName = "Android"
	case strings.Contains(lower, "iphone"), strings.Contains(lower, "ipad"), strings.Contains(lower, "ios"):
		osName = "iOS"
	case strings.Contains(lower, "linux"):
		osName = "Linux"
	}

	return deviceType, browser, osName
}

func lookupGeo(conn *sql.DB, ip string) geoRecord {
	if ip == "" || ip == "unknown" {
		return geoRecord{}
	}

	var record geoRecord
	var updatedAt string
	err := conn.QueryRow(`SELECT country, region, city, updated_at FROM geo_cache WHERE ip = ?`, ip).
		Scan(&record.Country, &record.Region, &record.City, &updatedAt)
	if err == nil {
		if t, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil && time.Since(t) < 7*24*time.Hour {
			return record
		}
	}

	fetched := fetchGeo(ip)
	if fetched.Country == "" && fetched.Region == "" && fetched.City == "" {
		return record
	}
	_, _ = conn.Exec(`INSERT INTO geo_cache(ip, country, region, city, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(ip) DO UPDATE SET country=excluded.country, region=excluded.region, city=excluded.city, updated_at=excluded.updated_at`,
		ip, fetched.Country, fetched.Region, fetched.City, time.Now().UTC().Format(time.RFC3339))
	return fetched
}

func fetchGeo(ip string) geoRecord {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipwho.is/"+ip, nil)
	if err != nil {
		return geoRecord{}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return geoRecord{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return geoRecord{}
	}

	var payload struct {
		Success bool   `json:"success"`
		Country string `json:"country"`
		Region  string `json:"region"`
		City    string `json:"city"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || !payload.Success {
		return geoRecord{}
	}
	return geoRecord{
		Country: payload.Country,
		Region:  payload.Region,
		City:    payload.City,
	}
}

func limit(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
