package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With,X-Admin-Token")
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
	expectedUsername := "vantalens"
	if cfg != nil && strings.TrimSpace(cfg.AdminUsername) != "" {
		expectedUsername = strings.TrimSpace(cfg.AdminUsername)
	}
	if cfg == nil || username != expectedUsername {
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
	accessToken, _ := auth.CreateJWT(expectedUsername, "access")
	refreshToken, _ := auth.CreateJWT(expectedUsername, "refresh")
	auth.SetSessionCookie(w, r, accessToken)
	RespondJSON(w, 200, models.APIResponse{Success: true, Data: map[string]string{"token": accessToken, "access_token": accessToken, "refresh_token": refreshToken}})
}

func HandlePlatformSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if claims, _, err := auth.AuthenticateRequest(r); err == nil {
		RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]interface{}{"authenticated": true, "user": claims.Sub, "expires_at": claims.Exp}})
		return
	}
	username, password, ok := r.BasicAuth()
	if !ok {
		RespondJSON(w, http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Basic auth credentials unavailable"})
		return
	}
	cfg := config.GetConfig()
	expectedUsername := "vantalens"
	if cfg != nil && strings.TrimSpace(cfg.AdminUsername) != "" {
		expectedUsername = strings.TrimSpace(cfg.AdminUsername)
	}
	if cfg == nil || strings.TrimSpace(username) != expectedUsername {
		RespondJSON(w, http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Unauthorized"})
		return
	}
	password = strings.TrimSpace(password)
	if cfg.AdminToken != "" && password != cfg.AdminToken {
		RespondJSON(w, http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Unauthorized"})
		return
	}
	if cfg.AdminToken == "" && password == "" {
		RespondJSON(w, http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Unauthorized"})
		return
	}
	accessToken, _ := auth.CreateJWT(expectedUsername, "access")
	refreshToken, _ := auth.CreateJWT(expectedUsername, "refresh")
	auth.SetSessionCookie(w, r, accessToken)
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]string{"token": accessToken, "access_token": accessToken, "refresh_token": refreshToken}})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	auth.ClearSessionCookie(w, r)
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Logged out"})
}

func HandleGetComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	path := r.URL.Query().Get("path")
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("all")), "1") {
		if !auth.RequireAuth(w, r) {
			return
		}
		if remoteAdminConfigured() {
			result, err := proxyRemoteAdmin(r, http.MethodGet, "/api/admin/comments?all=1", nil)
			if err == nil {
				result.Response.Message = "已从服务器权威后端实时读取评论"
				RespondJSON(w, http.StatusOK, result.Response)
				return
			}
			if !localCacheEnabled() {
				RespondJSON(w, http.StatusBadGateway, models.APIResponse{
					Success: false,
					Message: "服务器权威后端不可达，且本地缓存兜底已关闭",
					Data:    remoteErrorData(result, err, "disabled"),
				})
				return
			}
		}
		comments, err := comment.GetAllComments()
		if err != nil {
			RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
			return
		}
		message := ""
		if remoteAdminConfigured() {
			message = "服务器权威后端不可达，当前显示本地缓存评论"
		}
		RespondJSON(w, 200, models.APIResponse{Success: true, Message: message, Data: comments})
		return
	}
	if isAuthenticated(r) {
		if strings.TrimSpace(path) == "" {
			if remoteAdminConfigured() {
				result, err := proxyRemoteAdmin(r, http.MethodGet, "/api/admin/comments?all=1", nil)
				if err == nil {
					result.Response.Message = "已从服务器权威后端实时读取评论"
					RespondJSON(w, http.StatusOK, result.Response)
					return
				}
				if !localCacheEnabled() {
					RespondJSON(w, http.StatusBadGateway, models.APIResponse{
						Success: false,
						Message: "服务器权威后端不可达，且本地缓存兜底已关闭",
						Data:    remoteErrorData(result, err, "disabled"),
					})
					return
				}
			}
			comments, err := comment.GetAllComments()
			if err != nil {
				RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
				return
			}
			message := ""
			if remoteAdminConfigured() {
				message = "服务器权威后端不可达，当前显示本地缓存评论"
			}
			RespondJSON(w, 200, models.APIResponse{Success: true, Message: message, Data: comments})
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
	snapshot := snapshotCommentByID(id)
	ensureOplogID(r)
	if remoteAdminConfigured() {
		remotePath := "/api/admin/comments/" + url.PathEscape(id) + "/approve"
		result, err := proxyRemoteAdmin(r, http.MethodPost, remotePath, nil)
		if err != nil {
			recordOp(r, "comment.approve", id, "审核评论 "+id, snapshot, err, false)
			RespondJSON(w, http.StatusBadGateway, models.APIResponse{
				Success: false,
				Message: "远端评论审核失败，未执行本地写入兜底: " + err.Error(),
				Data:    remoteErrorData(result, err, "write_not_fallback"),
			})
			return
		}
		recordOp(r, "comment.approve", id, "审核评论 "+id, snapshot, nil, true)
		result.Response.Message = "评论已在服务器权威后端审核"
		RespondJSON(w, http.StatusOK, result.Response)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	now := time.Now().UTC().Format(time.RFC3339)
	sql := "BEGIN IMMEDIATE; UPDATE comments SET approved = 1, updated_at = " + sqliteLiteral(now) + " WHERE id = " + sqliteLiteral(id) + "; SELECT changes(); COMMIT;"
	stages, err := runRemoteCommentMutation(ctx, sql, func() error {
		return comment.ApproveComment(path, id)
	})
	if err != nil {
		recordOp(r, "comment.approve", id, "审核评论 "+id, snapshot, err, opSyncedByDefault())
		RespondJSON(w, commentMutationStatus(err), models.APIResponse{Success: false, Message: err.Error(), Data: stages})
		return
	}
	recordOp(r, "comment.approve", id, "审核评论 "+id, snapshot, nil, opSyncedByDefault())
	RespondJSON(w, 200, models.APIResponse{Success: true, Message: "评论已审核并写回服务器", Data: stages})
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
	snapshot := snapshotCommentByID(id)
	ensureOplogID(r)
	if remoteAdminConfigured() {
		remotePath := "/api/admin/comments/" + url.PathEscape(id) + "/delete"
		result, err := proxyRemoteAdmin(r, http.MethodPost, remotePath, nil)
		if err != nil {
			recordOp(r, "comment.delete", id, "删除评论 "+id, snapshot, err, false)
			RespondJSON(w, http.StatusBadGateway, models.APIResponse{
				Success: false,
				Message: "远端评论删除失败，未执行本地写入兜底: " + err.Error(),
				Data:    remoteErrorData(result, err, "write_not_fallback"),
			})
			return
		}
		recordOp(r, "comment.delete", id, "删除评论 "+id, snapshot, nil, true)
		result.Response.Message = "评论已在服务器权威后端删除"
		RespondJSON(w, http.StatusOK, result.Response)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	sql := "BEGIN IMMEDIATE; DELETE FROM comments WHERE id = " + sqliteLiteral(id) + "; SELECT changes(); COMMIT;"
	stages, err := runRemoteCommentMutation(ctx, sql, func() error {
		return comment.DeleteComment(path, id)
	})
	if err != nil {
		recordOp(r, "comment.delete", id, "删除评论 "+id, snapshot, err, opSyncedByDefault())
		RespondJSON(w, commentMutationStatus(err), models.APIResponse{Success: false, Message: err.Error(), Data: stages})
		return
	}
	recordOp(r, "comment.delete", id, "删除评论 "+id, snapshot, nil, opSyncedByDefault())
	RespondJSON(w, 200, models.APIResponse{Success: true, Message: "评论已删除并写回服务器", Data: stages})
}

func commentMutationStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
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
	snapshot := snapshotCommentSettings()
	if err := comment.SaveSettings(settings); err != nil {
		recordOp(r, "settings.save", "comment_settings", "保存评论设置", snapshot, err, opSyncedByDefault())
		RespondJSON(w, 500, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	recordOp(r, "settings.save", "comment_settings", "保存评论设置", snapshot, nil, opSyncedByDefault())
	RespondJSON(w, 200, models.APIResponse{Success: true})
}

func sqliteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
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
