package analytics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePagePath(t *testing.T) {
	tests := map[string]string{
		"":                                   "/",
		"/post/demo/?utm_source=x#comments": "/post/demo",
		"https://vantalens.com/archive/?q=1": "/archive",
		"post/demo/index.html":              "/post/demo/index.html",
	}
	for input, want := range tests {
		if got := normalizePagePath(input); got != want {
			t.Fatalf("normalizePagePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsTrackablePage(t *testing.T) {
	trackable := []string{"/", "/post/demo", "/archives", "/page/search"}
	for _, path := range trackable {
		if !isTrackablePage(path) {
			t.Fatalf("isTrackablePage(%q) = false, want true", path)
		}
	}

	blocked := []string{"/api/comments", "/platform/backend", "/preview/", "/scss/style.css", "/js/app.js", "/img/a.png"}
	for _, path := range blocked {
		if isTrackablePage(path) {
			t.Fatalf("isTrackablePage(%q) = true, want false", path)
		}
	}
}

func TestGetSiteStatisticsIncludesRegionAggregates(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "visits.db")
	t.Setenv("ANALYTICS_DB_PATH", dbPath)
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = Close()
		_ = os.Remove(dbPath)
	})
	conn, err := getDB()
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		ip      string
		country string
		region  string
		city    string
		path    string
		created string
	}{
		{"203.0.113.10", "United States", "California", "San Francisco", "/a", "2026-06-09T10:00:00Z"},
		{"203.0.113.11", "United States", "California", "San Francisco", "/b", "2026-06-09T10:02:00Z"},
		{"203.0.113.10", "United States", "California", "San Francisco", "/a", "2026-06-09T10:03:00Z"},
		{"198.51.100.20", "Japan", "Tokyo", "Tokyo", "/c", "2026-06-09T10:04:00Z"},
	}
	for index, row := range rows {
		if _, err := conn.Exec(`
INSERT INTO visits (
	session_id, page_path, page_title, referrer, ip, device_type, browser, os_name,
	region, country, city, dns_host, language, timezone, screen, user_agent,
	webrtc_json, is_page_view, created_at
) VALUES (?, ?, ?, '', ?, 'desktop', 'Chrome', 'Windows', ?, ?, ?, '', 'en', 'UTC', '1920x1080', 'test', '{}', 1, ?)`,
			"session-"+row.ip+"-"+row.created, row.path, "Page", row.ip, row.region, row.country, row.city, row.created); err != nil {
			t.Fatalf("insert row %d: %v", index, err)
		}
	}

	stats, err := GetSiteStatistics(20)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalViews != 4 || stats.UniqueIPs != 3 {
		t.Fatalf("totals = views %d unique_ips %d, want 4 and 3", stats.TotalViews, stats.UniqueIPs)
	}
	if len(stats.Regions) < 2 {
		t.Fatalf("regions = %+v, want at least two region groups", stats.Regions)
	}
	top := stats.Regions[0]
	if top.Label != "United States / California / San Francisco" || top.Views != 3 || top.UniqueIPs != 2 {
		t.Fatalf("top region = %+v, want SF aggregate with 3 views and 2 IPs", top)
	}
	if len(stats.Visitors) == 0 || stats.Visitors[0].Country == "" || stats.Visitors[0].City == "" {
		t.Fatalf("visitor geo fields missing: %+v", stats.Visitors)
	}
	if len(stats.RecentVisits) == 0 || stats.RecentVisits[0].Country == "" {
		t.Fatalf("recent visit geo fields missing: %+v", stats.RecentVisits)
	}
}
