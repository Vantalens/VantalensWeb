package models

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// marshalToMap marshals v and decodes the result into a generic map so tests
// can assert on exact JSON keys and values.
func marshalToMap(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal marshaled output: %v", err)
	}
	return out
}

func TestPostJSONFieldNames(t *testing.T) {
	post := Post{
		Title:       "你好，世界",
		Lang:        "zh",
		Path:        "content/posts/hello.md",
		Date:        "2024-01-02",
		Status:      "published",
		StatusColor: "green",
		Pinned:      true,
	}
	m := marshalToMap(t, post)

	want := map[string]interface{}{
		"title":        "你好，世界",
		"lang":         "zh",
		"path":         "content/posts/hello.md",
		"date":         "2024-01-02",
		"status":       "published",
		"status_color": "green",
		"pinned":       true,
	}
	for key, wantVal := range want {
		got, ok := m[key]
		if !ok {
			t.Fatalf("missing JSON key %q in %v", key, m)
		}
		if got != wantVal {
			t.Fatalf("key %q = %v, want %v", key, got, wantVal)
		}
	}
	if len(m) != len(want) {
		t.Fatalf("unexpected extra keys: %v", m)
	}
}

func TestArticleRecordOmitEmpty(t *testing.T) {
	empty := ArticleRecord{Post: Post{Title: "t", Path: "p"}}
	m := marshalToMap(t, empty)
	for _, key := range []string{"content", "created_at", "updated_at"} {
		if _, ok := m[key]; ok {
			t.Fatalf("key %q must be omitted when empty, got %v", key, m)
		}
	}

	full := ArticleRecord{
		Post:      Post{Title: "t", Path: "p"},
		Content:   "body",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-02T00:00:00Z",
	}
	m = marshalToMap(t, full)
	for _, key := range []string{"content", "created_at", "updated_at"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("key %q must be present when set, got %v", key, m)
		}
	}
}

func TestArticleRecordEmbedsPostFields(t *testing.T) {
	raw := `{"title":"嵌入","lang":"zh","path":"a.md","date":"","status":"","status_color":"","pinned":false,"content":"c"}`
	var record ArticleRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record.Title != "嵌入" || record.Content != "c" {
		t.Fatalf("record = %+v, want embedded Post fields decoded", record)
	}
}

func TestCommentOmitEmpty(t *testing.T) {
	minimal := Comment{ID: "1", Author: "a", Email: "e", Content: "c", Timestamp: "ts", PostPath: "p", IPAddress: "ip", UserAgent: "ua"}
	m := marshalToMap(t, minimal)

	omitted := []string{"phone", "parent_id", "images", "fingerprint", "captcha_score", "vpn_suspected", "risk_reasons"}
	for _, key := range omitted {
		if _, ok := m[key]; ok {
			t.Fatalf("key %q must be omitted for zero value, got %v", key, m)
		}
	}

	full := Comment{
		ID: "1", Author: "a", Email: "e", Content: "c", Timestamp: "ts", PostPath: "p",
		Phone: "123", ParentID: "0", Images: []string{"i1"}, Fingerprint: "fp",
		CaptchaScore: 90, VPNSuspected: true, RiskReasons: []string{"blacklist"},
	}
	m = marshalToMap(t, full)
	for _, key := range omitted {
		if _, ok := m[key]; !ok {
			t.Fatalf("key %q must be present when set, got %v", key, m)
		}
	}
}

func TestCommentRoundTrip(t *testing.T) {
	original := Comment{
		ID:           "abc",
		Author:       "读者",
		Email:        "reader@example.com",
		Content:      "写得不错 👍",
		Approved:     true,
		PostPath:     "content/posts/a.md",
		RiskReasons:  []string{"重复内容", "黑名单词"},
		CaptchaScore: 42,
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Comment
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Author != original.Author || decoded.Content != original.Content {
		t.Fatalf("unicode fields corrupted: %+v", decoded)
	}
	if len(decoded.RiskReasons) != 2 || decoded.RiskReasons[1] != "黑名单词" {
		t.Fatalf("risk_reasons = %v, want round-trip preserved", decoded.RiskReasons)
	}
	if decoded.CaptchaScore != 42 || !decoded.Approved {
		t.Fatalf("decoded = %+v, want captcha_score 42 and approved true", decoded)
	}
}

func TestCommentSettingsSnakeCaseKeys(t *testing.T) {
	raw := `{
		"mail_provider": "smtp",
		"smtp_enabled": true,
		"smtp_host": "smtp.example.com",
		"smtp_port": 465,
		"smtp_user": "u",
		"smtp_pass": "p",
		"smtp_from": "from@example.com",
		"smtp_to": ["a@example.com", "b@example.com"],
		"microsoft_tenant": "common",
		"microsoft_client_id": "cid",
		"microsoft_client_secret": "sec",
		"microsoft_refresh_token": "rt",
		"microsoft_sender": "sender@example.com",
		"notify_on_pending": true,
		"blacklist_ips": ["1.2.3.4"],
		"blacklist_keywords": ["spam"]
	}`
	var settings CommentSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if settings.SMTPHost != "smtp.example.com" || settings.SMTPPort != 465 {
		t.Fatalf("settings = %+v, want smtp host/port decoded", settings)
	}
	if len(settings.SMTPTo) != 2 || settings.SMTPTo[1] != "b@example.com" {
		t.Fatalf("smtp_to = %v, want two recipients", settings.SMTPTo)
	}
	if settings.BlacklistWords[0] != "spam" || settings.BlacklistIPs[0] != "1.2.3.4" {
		t.Fatalf("blacklists = %v / %v", settings.BlacklistWords, settings.BlacklistIPs)
	}

	// Marshaling must use the same snake_case keys.
	m := marshalToMap(t, settings)
	for _, key := range []string{"mail_provider", "smtp_enabled", "smtp_host", "smtp_port", "smtp_to", "notify_on_pending", "blacklist_ips", "blacklist_keywords"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("marshaled settings missing key %q: %v", key, m)
		}
	}
}

func TestAPIResponseOmitEmpty(t *testing.T) {
	minimal := APIResponse{Success: false}
	raw, err := json.Marshal(minimal)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Success has no omitempty: false must still be serialized.
	if !bytes.Contains(raw, []byte(`"success":false`)) {
		t.Fatalf("marshaled %s, want success:false present", raw)
	}
	for _, key := range []string{"message", "content", "data"} {
		if strings.Contains(string(raw), `"`+key+`"`) {
			t.Fatalf("key %q must be omitted when empty in %s", key, raw)
		}
	}

	full := APIResponse{Success: true, Message: "ok", Content: "body", Data: map[string]int{"n": 1}}
	m := marshalToMap(t, full)
	if m["success"] != true || m["message"] != "ok" || m["content"] != "body" {
		t.Fatalf("marshaled = %v, want all fields present", m)
	}
}

func TestJWTClaimsFieldNames(t *testing.T) {
	claims := JWTClaims{Sub: "alice", Iat: 100, Exp: 200, Jti: "id", Typ: "access"}
	m := marshalToMap(t, claims)
	want := map[string]interface{}{"sub": "alice", "iat": float64(100), "exp": float64(200), "jti": "id", "typ": "access"}
	for key, wantVal := range want {
		if got := m[key]; got != wantVal {
			t.Fatalf("key %q = %v, want %v", key, got, wantVal)
		}
	}
	if len(m) != len(want) {
		t.Fatalf("unexpected extra keys: %v", m)
	}
}

func TestCommentsFileUnmarshal(t *testing.T) {
	raw := `{"comments":[{"id":"1","author":"a","email":"e","content":"c","timestamp":"t","approved":false,"post_path":"p","ip_address":"i","user_agent":"u"}]}`
	var file CommentsFile
	if err := json.Unmarshal([]byte(raw), &file); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(file.Comments) != 1 || file.Comments[0].ID != "1" {
		t.Fatalf("file = %+v, want one comment decoded", file)
	}
}

func TestAnalyticsTypesFieldNames(t *testing.T) {
	visit := VisitRecord{ID: 7, SessionID: "s", Path: "/p", CreatedAt: "2024-01-01T00:00:00Z", PageView: true}
	m := marshalToMap(t, visit)
	for _, key := range []string{"id", "session_id", "path", "created_at", "page_view", "user_agent", "dns_host"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("VisitRecord missing key %q: %v", key, m)
		}
	}
	// webrtc is optional and must be omitted when nil.
	if _, ok := m["webrtc"]; ok {
		t.Fatalf("webrtc must be omitted when nil: %v", m)
	}

	stats := SiteStatistics{TotalPages: 1, TotalViews: 2, TotalComments: 3, PendingComments: 4, UniqueIPs: 5, UniqueSessions: 6}
	m = marshalToMap(t, stats)
	for _, key := range []string{"total_pages", "total_views", "total_comments", "pending_comments", "unique_ips", "unique_sessions"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("SiteStatistics missing key %q: %v", key, m)
		}
	}

	region := RegionStatistics{Label: "成都", Views: 10, UniqueIPs: 3, Latitude: 30.57, Longitude: 104.07}
	m = marshalToMap(t, region)
	if m["label"] != "成都" || m["unique_ips"] != float64(3) {
		t.Fatalf("RegionStatistics = %v, want label/unique_ips decoded", m)
	}
}
