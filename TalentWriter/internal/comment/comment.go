package comment

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

type Challenge struct {
	Token    string `json:"token"`
	Question string `json:"question"`
	Expires  string `json:"expires"`
}

type VerificationCode struct {
	Code    string
	Email   string
	Expires time.Time
}

type SubmitMeta struct {
	Phone           string
	Fingerprint     string
	CaptchaToken    string
	CaptchaAnswer   string
	EmailCode       string
	Honeypot        string
	WebRTCPublicIPs []string
}

var (
	db   *sql.DB
	dbMu sync.RWMutex

	challengeMu sync.Mutex
	challenges  = map[string]struct {
		Answer  string
		Expires time.Time
	}{}

	codeMu sync.Mutex
	codes  = map[string]VerificationCode{}
)

func Init(hugoPath string) error {
	path := config.GetCommentsDBPath(hugoPath)
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
CREATE TABLE IF NOT EXISTS comments (
	id TEXT PRIMARY KEY,
	post_path TEXT NOT NULL,
	author TEXT NOT NULL,
	phone TEXT NOT NULL,
	email TEXT NOT NULL,
	content TEXT NOT NULL,
	parent_id TEXT NOT NULL,
	approved INTEGER NOT NULL DEFAULT 0,
	ip_address TEXT NOT NULL,
	user_agent TEXT NOT NULL,
	fingerprint TEXT NOT NULL,
	captcha_score INTEGER NOT NULL DEFAULT 0,
	vpn_suspected INTEGER NOT NULL DEFAULT 0,
	risk_reasons_json TEXT NOT NULL DEFAULT '[]',
	images_json TEXT NOT NULL DEFAULT '[]',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_comments_post_approved ON comments(post_path, approved, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_email ON comments(email);
CREATE INDEX IF NOT EXISTS idx_comments_ip ON comments(ip_address);
CREATE INDEX IF NOT EXISTS idx_comments_fingerprint ON comments(fingerprint);
`
	_, err := conn.Exec(schema)
	return err
}

func GetComments(postPath string) ([]models.Comment, error) {
	conn, err := getDB()
	if err != nil {
		return nil, err
	}
	variants := normalizePostPathVariants(postPath)
	if len(variants) == 0 {
		return []models.Comment{}, fmt.Errorf("invalid post path")
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(variants)), ",")
	args := make([]interface{}, 0, len(variants))
	for _, variant := range variants {
		args = append(args, variant)
	}
	rows, err := conn.Query(fmt.Sprintf(`
SELECT id, author, phone, email, content, created_at, approved, post_path, ip_address, user_agent,
       parent_id, images_json, fingerprint, captcha_score, vpn_suspected, risk_reasons_json
FROM comments
WHERE post_path IN (%s)
ORDER BY created_at ASC`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanComments(rows)
}

func GetAllComments() ([]models.Comment, error) {
	conn, err := getDB()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(`
SELECT id, author, phone, email, content, created_at, approved, post_path, ip_address, user_agent,
       parent_id, images_json, fingerprint, captcha_score, vpn_suspected, risk_reasons_json
FROM comments
ORDER BY approved ASC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

func scanComments(rows *sql.Rows) ([]models.Comment, error) {
	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		var approved, vpn int
		var imagesJSON, reasonsJSON string
		if err := rows.Scan(&c.ID, &c.Author, &c.Phone, &c.Email, &c.Content, &c.Timestamp, &approved, &c.PostPath,
			&c.IPAddress, &c.UserAgent, &c.ParentID, &imagesJSON, &c.Fingerprint, &c.CaptchaScore, &vpn, &reasonsJSON); err != nil {
			return nil, err
		}
		c.Approved = approved == 1
		c.VPNSuspected = vpn == 1
		_ = json.Unmarshal([]byte(imagesJSON), &c.Images)
		_ = json.Unmarshal([]byte(reasonsJSON), &c.RiskReasons)
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

func AddComment(postPath, author, email, content, ipAddress, userAgent, parentID string, meta SubmitMeta, r *http.Request) (models.Comment, error) {
	conn, err := getDB()
	if err != nil {
		return models.Comment{}, err
	}

	postPath = normalizePostPath(postPath)
	author = strings.TrimSpace(author)
	email = strings.ToLower(strings.TrimSpace(email))
	content = strings.TrimSpace(content)
	parentID = strings.TrimSpace(parentID)
	meta.Phone = strings.TrimSpace(meta.Phone)
	meta.Fingerprint = strings.TrimSpace(meta.Fingerprint)

	if meta.Honeypot != "" {
		return models.Comment{}, fmt.Errorf("comment blocked")
	}
	if postPath == "" {
		return models.Comment{}, fmt.Errorf("invalid post path")
	}
	if author == "" || len(author) > config.MaxCommentNameLen {
		return models.Comment{}, fmt.Errorf("invalid author")
	}
	if meta.Phone == "" || !isLikelyInternationalPhone(meta.Phone) {
		return models.Comment{}, fmt.Errorf("invalid phone")
	}
	if email == "" || len(email) > config.MaxCommentEmailLen || !isLikelyEmail(email) {
		return models.Comment{}, fmt.Errorf("invalid email")
	}
	if content == "" || len(content) > config.MaxCommentContentLen {
		return models.Comment{}, fmt.Errorf("invalid comment content")
	}
	if meta.Fingerprint == "" || len(meta.Fingerprint) > 128 {
		return models.Comment{}, fmt.Errorf("invalid browser fingerprint")
	}
	if !VerifyChallenge(meta.CaptchaToken, meta.CaptchaAnswer) {
		return models.Comment{}, fmt.Errorf("captcha verification failed")
	}
	if !VerifyEmailCode(email, meta.EmailCode) {
		return models.Comment{}, fmt.Errorf("email verification failed")
	}

	settings := LoadSettings()
	if IsBlacklisted(settings, ipAddress, author, email, content) {
		return models.Comment{}, fmt.Errorf("comment blocked")
	}

	reasons := riskReasons(ipAddress, userAgent, meta, r)
	now := time.Now().UTC().Format(time.RFC3339)
	comment := models.Comment{
		ID:           generateCommentID(),
		Author:       author,
		Phone:        meta.Phone,
		Email:        email,
		Content:      content,
		Timestamp:    now,
		Approved:     false,
		PostPath:     postPath,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ParentID:     parentID,
		Fingerprint:  hashFingerprint(meta.Fingerprint),
		CaptchaScore: 100,
		VPNSuspected: containsRisk(reasons, "vpn") || containsRisk(reasons, "proxy"),
		RiskReasons:  reasons,
	}
	reasonsJSON, _ := json.Marshal(comment.RiskReasons)
	imagesJSON, _ := json.Marshal(comment.Images)

	_, err = conn.Exec(`
INSERT INTO comments (
	id, post_path, author, phone, email, content, parent_id, approved, ip_address, user_agent,
	fingerprint, captcha_score, vpn_suspected, risk_reasons_json, images_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		comment.ID, comment.PostPath, comment.Author, comment.Phone, comment.Email, comment.Content, comment.ParentID,
		comment.IPAddress, comment.UserAgent, comment.Fingerprint, comment.CaptchaScore, boolToInt(comment.VPNSuspected),
		string(reasonsJSON), string(imagesJSON), now, now)
	if err != nil {
		return models.Comment{}, err
	}
	return comment, nil
}

func ApproveComment(postPath, commentID string) error {
	conn, err := getDB()
	if err != nil {
		return err
	}
	commentID = strings.TrimSpace(commentID)
	result, err := conn.Exec(`UPDATE comments SET approved = 1, updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), commentID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("comment not found")
	}
	return nil
}

func DeleteComment(postPath, commentID string) error {
	conn, err := getDB()
	if err != nil {
		return err
	}
	commentID = strings.TrimSpace(commentID)
	result, err := conn.Exec(`DELETE FROM comments WHERE id = ?`, commentID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("comment not found")
	}
	return nil
}

func NewChallenge() Challenge {
	a := randomInt(2, 9)
	b := randomInt(2, 9)
	token := randomString(24)
	expires := time.Now().Add(10 * time.Minute)
	challengeMu.Lock()
	challenges[token] = struct {
		Answer  string
		Expires time.Time
	}{Answer: fmt.Sprintf("%d", a+b), Expires: expires}
	pruneChallenges()
	challengeMu.Unlock()
	return Challenge{
		Token:    token,
		Question: fmt.Sprintf("%d + %d = ?", a, b),
		Expires:  expires.Format(time.RFC3339),
	}
}

func VerifyChallenge(token, answer string) bool {
	token = strings.TrimSpace(token)
	answer = strings.TrimSpace(answer)
	if token == "" || answer == "" {
		return false
	}
	challengeMu.Lock()
	defer challengeMu.Unlock()
	item, ok := challenges[token]
	if !ok || time.Now().After(item.Expires) || item.Answer != answer {
		return false
	}
	delete(challenges, token)
	return true
}

func CreateEmailCode(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !isLikelyEmail(email) {
		return "", fmt.Errorf("invalid email")
	}
	code := fmt.Sprintf("%06d", randomInt(0, 999999))
	codeMu.Lock()
	codes[email] = VerificationCode{Code: code, Email: email, Expires: time.Now().Add(10 * time.Minute)}
	pruneCodes()
	codeMu.Unlock()
	return code, nil
}

func VerifyEmailCode(email, code string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)
	if email == "" || code == "" {
		return false
	}
	codeMu.Lock()
	defer codeMu.Unlock()
	item, ok := codes[email]
	if !ok || time.Now().After(item.Expires) || item.Code != code {
		return false
	}
	delete(codes, email)
	return true
}

func IsBlacklisted(settings models.CommentSettings, ip, author, email, content string) bool {
	ip = strings.TrimSpace(strings.ToLower(ip))
	text := strings.ToLower(strings.Join([]string{author, email, content}, " "))
	for _, b := range settings.BlacklistIPs {
		if strings.TrimSpace(strings.ToLower(b)) != "" && ip != "" && strings.Contains(ip, strings.TrimSpace(strings.ToLower(b))) {
			return true
		}
	}
	for _, w := range settings.BlacklistWords {
		keyword := strings.TrimSpace(strings.ToLower(w))
		if keyword != "" && strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func LoadSettings() models.CommentSettings {
	cfg := config.GetConfig()
	if cfg == nil {
		return models.CommentSettings{}
	}
	path := config.GetCommentSettingsPath(cfg.HugoPath)
	settings := models.CommentSettings{
		SMTPEnabled:     false,
		SMTPPort:        587,
		NotifyOnPending: true,
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return settings
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}
	_ = json.Unmarshal(data, &settings)
	return settings
}

func SaveSettings(settings models.CommentSettings) error {
	cfg := config.GetConfig()
	if cfg == nil {
		return nil
	}
	path := config.GetCommentSettingsPath(cfg.HugoPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(settings, "", " ")
	return os.WriteFile(path, data, 0o600)
}

func getDB() (*sql.DB, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	if db == nil {
		return nil, errors.New("comment database not initialized")
	}
	return db, nil
}

func riskReasons(ipAddress, userAgent string, meta SubmitMeta, r *http.Request) []string {
	var reasons []string
	lowerUA := strings.ToLower(userAgent)
	if lowerUA == "" || strings.Contains(lowerUA, "bot") || strings.Contains(lowerUA, "spider") || strings.Contains(lowerUA, "crawl") || strings.Contains(lowerUA, "curl") {
		reasons = append(reasons, "bot-like user agent")
	}
	if r != nil {
		for _, key := range []string{"Via", "Forwarded", "X-Proxy-ID", "Proxy-Connection", "X-Forwarded-Host"} {
			if strings.TrimSpace(r.Header.Get(key)) != "" {
				reasons = append(reasons, "proxy header: "+key)
			}
		}
		if strings.EqualFold(r.Header.Get("CF-IPCountry"), "T1") {
			reasons = append(reasons, "vpn country signal")
		}
	}
	for _, publicIP := range meta.WebRTCPublicIPs {
		publicIP = strings.TrimSpace(publicIP)
		if publicIP != "" && cleanIP(publicIP) != "" && cleanIP(publicIP) != cleanIP(ipAddress) {
			reasons = append(reasons, "vpn/webrtc ip mismatch")
			break
		}
	}
	return uniqueStrings(reasons)
}

func normalizePostPath(postPath string) string {
	normalized := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(strings.TrimSpace(postPath), "\\", "/")))
	if normalized == "." || normalized == "" || filepath.IsAbs(normalized) {
		return ""
	}
	return strings.ReplaceAll(normalized, "\\", "/")
}

func normalizePostPathVariants(postPath string) []string {
	base := normalizePostPath(postPath)
	if base == "" {
		return nil
	}
	candidates := []string{base}
	trimmed := strings.TrimPrefix(base, "content/")
	if trimmed != base {
		candidates = append(candidates, trimmed)
	}
	for _, lang := range []string{"zh-cn/", "zh/", "en/"} {
		if strings.HasPrefix(trimmed, lang) {
			candidates = append(candidates, strings.TrimPrefix(trimmed, lang))
		}
	}
	if !strings.HasSuffix(base, "/index.md") && strings.HasSuffix(base, ".md") {
		noExt := strings.TrimSuffix(base, ".md")
		candidates = append(candidates, noExt+"/index.md")
	}
	if strings.HasSuffix(base, "/index.md") {
		candidates = append(candidates, strings.TrimSuffix(base, "/index.md")+".md")
	}

	seen := map[string]bool{}
	var out []string
	for _, candidate := range candidates {
		candidate = normalizePostPath(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

func generateCommentID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		raw = []byte(time.Now().Format("150405000000"))
	}
	for i := range b {
		b[i] = charset[int(raw[i%len(raw)])%len(charset)]
	}
	return string(b)
}

func randomInt(min, max int) int {
	if max <= min {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return min + int(time.Now().UnixNano()%int64(max-min+1))
	}
	return min + int(n.Int64())
}

var (
	emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
)

func isLikelyEmail(v string) bool {
	return emailPattern.MatchString(v)
}

func isLikelyInternationalPhone(v string) bool {
	v = strings.ReplaceAll(strings.TrimSpace(v), " ", "")
	v = strings.ReplaceAll(v, "-", "")
	return phonePattern.MatchString(v)
}

func hashFingerprint(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func pruneChallenges() {
	now := time.Now()
	for key, item := range challenges {
		if now.After(item.Expires) {
			delete(challenges, key)
		}
	}
}

func pruneCodes() {
	now := time.Now()
	for key, item := range codes {
		if now.After(item.Expires) {
			delete(codes, key)
		}
	}
}

func cleanIP(raw string) string {
	raw = strings.TrimSpace(strings.Trim(raw, "[]"))
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func containsRisk(reasons []string, needle string) bool {
	for _, reason := range reasons {
		if strings.Contains(strings.ToLower(reason), needle) {
			return true
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
