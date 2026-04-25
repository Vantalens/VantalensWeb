package handlers

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/comment"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/email"
	"vantalens/talentwriter/internal/models"
)

type rateBucket struct {
	Count    int
	ResetAt  time.Time
	LastSeen time.Time
}

var (
	loginRateMu   sync.Mutex
	loginRateHits = map[string]rateBucket{}

	commentRateMu   sync.Mutex
	commentRateHits = map[string]rateBucket{}
)

func RespondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func WithCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")
		for _, o := range allowedOrigins() {
			if o == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func allowedOrigins() []string {
	allowed := []string{
		"http://localhost:1313",
		"http://localhost:9090",
		"http://localhost:9091",
		"http://127.0.0.1:1313",
		"http://127.0.0.1:9090",
		"http://127.0.0.1:9091",
		"https://vantalens.com",
		"https://www.vantalens.com",
		"https://w2343419-del.github.io",
	}
	for _, raw := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if value := strings.TrimSpace(raw); value != "" {
			allowed = append(allowed, value)
		}
	}
	return allowed
}

type loginRequest struct {
	User     string `json:"user"`
	Pass     string `json:"pass"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	clientIP := requestIP(r)
	if !allowRate(&loginRateMu, loginRateHits, clientIP, 8, 10*time.Minute) {
		RespondJSON(w, http.StatusTooManyRequests, models.APIResponse{Success: false, Message: "Too many login attempts"})
		return
	}
	var req loginRequest
	if err := decodeJSONBody(w, r, &req, 8<<10); err != nil {
		return
	}
	username := strings.TrimSpace(req.User)
	if username == "" {
		username = strings.TrimSpace(req.Username)
	}
	password := strings.TrimSpace(req.Pass)
	if password == "" {
		password = strings.TrimSpace(req.Password)
	}
	cfg := config.GetConfig()
	if cfg == nil || username != "admin" {
		RespondJSON(w, 401, models.APIResponse{Success: false, Message: "Unauthorized"})
		return
	}
	if cfg.AdminToken != "" && password != cfg.AdminToken {
		RespondJSON(w, 401, models.APIResponse{Success: false, Message: "Unauthorized"})
		return
	}
	if cfg.AdminToken == "" && strings.TrimSpace(password) == "" {
		RespondJSON(w, 401, models.APIResponse{Success: false, Message: "Unauthorized"})
		return
	}
	accessToken, _ := auth.CreateJWT("admin", "access")
	refreshToken, _ := auth.CreateJWT("admin", "refresh")
	RespondJSON(w, 200, models.APIResponse{Success: true, Data: map[string]string{"token": accessToken, "access_token": accessToken, "refresh_token": refreshToken}})
}

func HandleGetComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	path := r.URL.Query().Get("path")
	if isAuthenticated(r) {
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("all")), "1") || strings.TrimSpace(path) == "" {
			comments, err := comment.GetAllComments()
			if err != nil {
				RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
				return
			}
			RespondJSON(w, 200, models.APIResponse{Success: true, Data: comments})
			return
		}
		comments, err := comment.GetComments(path)
		if err != nil {
			RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
			return
		}
		RespondJSON(w, 200, models.APIResponse{Success: true, Data: comments})
		return
	}

	comments, err := comment.GetComments(path)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	publicComments := make([]models.Comment, 0, len(comments))
	for _, item := range comments {
		if !item.Approved {
			continue
		}
		item.Email = ""
		item.IPAddress = ""
		item.UserAgent = ""
		publicComments = append(publicComments, item)
	}
	RespondJSON(w, 200, models.APIResponse{Success: true, Data: publicComments})
}

type addCommentRequest struct {
	PostPath        string   `json:"post_path"`
	Author          string   `json:"author"`
	Phone           string   `json:"phone"`
	Email           string   `json:"email"`
	Content         string   `json:"content"`
	Parent          string   `json:"parent"`
	ParentID        string   `json:"parent_id"`
	Fingerprint     string   `json:"fingerprint"`
	CaptchaToken    string   `json:"captcha_token"`
	CaptchaAnswer   string   `json:"captcha_answer"`
	EmailCode       string   `json:"email_code"`
	Website         string   `json:"website"`
	WebRTCPublicIPs []string `json:"webrtc_public_ips"`
}

func HandleCommentChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: comment.NewChallenge()})
}

func HandleCommentEmailCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	clientIP := requestIP(r)
	if !allowRate(&commentRateMu, commentRateHits, "email:"+clientIP, 5, 15*time.Minute) {
		RespondJSON(w, http.StatusTooManyRequests, models.APIResponse{Success: false, Message: "Too many verification attempts"})
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSONBody(w, r, &req, 8<<10); err != nil {
		return
	}
	code, err := comment.CreateEmailCode(req.Email)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	settings := comment.LoadSettings()
	if err := email.SendVerificationCode(settings, req.Email, code); err != nil {
		RespondJSON(w, http.StatusServiceUnavailable, models.APIResponse{Success: false, Message: "Email verification is not configured"})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Verification code sent"})
}

func HandleAddComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	clientIP := requestIP(r)
	if !allowRate(&commentRateMu, commentRateHits, clientIP, 12, 15*time.Minute) {
		RespondJSON(w, http.StatusTooManyRequests, models.APIResponse{Success: false, Message: "Too many comment attempts"})
		return
	}
	path := r.URL.Query().Get("path")
	var req addCommentRequest
	if err := decodeJSONBody(w, r, &req, 64<<10); err != nil {
		return
	}
	if strings.TrimSpace(path) == "" {
		path = req.PostPath
	}
	parentID := req.ParentID
	if parentID == "" {
		parentID = req.Parent
	}
	c, err := comment.AddComment(path, req.Author, req.Email, req.Content, clientIP, r.UserAgent(), parentID, comment.SubmitMeta{
		Phone:           req.Phone,
		Fingerprint:     req.Fingerprint,
		CaptchaToken:    req.CaptchaToken,
		CaptchaAnswer:   req.CaptchaAnswer,
		EmailCode:       req.EmailCode,
		Honeypot:        req.Website,
		WebRTCPublicIPs: req.WebRTCPublicIPs,
	}, r)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(err.Error()), "blocked") {
			status = http.StatusForbidden
		}
		RespondJSON(w, status, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	email.QueueNotification(comment.LoadSettings(), c, path)
	RespondJSON(w, 200, models.APIResponse{Success: true, Data: c})
}

func HandleApproveComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	path := r.URL.Query().Get("path")
	id := r.URL.Query().Get("id")
	if err := comment.ApproveComment(path, id); err != nil {
		RespondJSON(w, 500, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, 200, models.APIResponse{Success: true})
}

func HandleDeleteComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	path := r.URL.Query().Get("path")
	id := r.URL.Query().Get("id")
	if err := comment.DeleteComment(path, id); err != nil {
		RespondJSON(w, 500, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, 200, models.APIResponse{Success: true})
}

func HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	settings := comment.LoadSettings()
	RespondJSON(w, 200, models.APIResponse{Success: true, Data: settings})
}

func HandleSaveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	var settings models.CommentSettings
	if err := decodeJSONBody(w, r, &settings, 32<<10); err != nil {
		return
	}
	if err := comment.SaveSettings(settings); err != nil {
		RespondJSON(w, 500, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, 200, models.APIResponse{Success: true})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}, maxBytes int64) error {
	if r.Body == nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Request body is required"})
		return io.EOF
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid JSON payload"})
		return err
	}
	return nil
}

func isAuthenticated(r *http.Request) bool {
	token := auth.ExtractBearerToken(r)
	if token == "" {
		return false
	}
	_, err := auth.VerifyJWT(token)
	return err == nil
}

func allowRate(mu *sync.Mutex, buckets map[string]rateBucket, key string, limit int, window time.Duration) bool {
	if strings.TrimSpace(key) == "" {
		key = "unknown"
	}
	now := time.Now()
	mu.Lock()
	defer mu.Unlock()

	bucket, ok := buckets[key]
	if !ok || now.After(bucket.ResetAt) {
		buckets[key] = rateBucket{
			Count:    1,
			ResetAt:  now.Add(window),
			LastSeen: now,
		}
		pruneBuckets(buckets, now.Add(-24*time.Hour))
		return true
	}
	if bucket.Count >= limit {
		bucket.LastSeen = now
		buckets[key] = bucket
		return false
	}
	bucket.Count++
	bucket.LastSeen = now
	buckets[key] = bucket
	return true
}

func pruneBuckets(buckets map[string]rateBucket, before time.Time) {
	for key, bucket := range buckets {
		if bucket.LastSeen.Before(before) {
			delete(buckets, key)
		}
	}
}

func requestIP(r *http.Request) string {
	for _, key := range []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
		raw := strings.TrimSpace(r.Header.Get(key))
		if raw == "" {
			continue
		}
		parts := strings.Split(raw, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	raw := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		return host
	}
	return raw
}
