package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

var jwtSecret []byte

const SessionCookieName = "tw_session"

func InitJWTSecret() error {
	secret := strings.TrimSpace(config.GetEnv("JWT_SECRET", ""))
	if secret == "" {
		if isProductionAuthRequired() {
			return errors.New("JWT_SECRET is required in production")
		}
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err == nil {
			secret = base64URLEncode(buf)
		} else {
			secret = generateRandomString(32)
		}
	}
	jwtSecret = []byte(secret)
	return nil
}

func isProductionAuthRequired() bool {
	for _, key := range []string{"AUTHORITY_BACKEND", "BEHIND_PROXY", "REQUIRE_PERSISTENT_JWT_SECRET"} {
		switch strings.ToLower(strings.TrimSpace(config.GetEnv(key, ""))) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}

func base64URLEncode(data []byte) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(s)
}

func getJWTExpiry() time.Duration {
	return 24 * time.Hour
}

func signJWT(headerPayload string) string {
	h := hmac.New(sha256.New, jwtSecret)
	h.Write([]byte(headerPayload))
	return base64URLEncode(h.Sum(nil))
}

func CreateJWT(username string, tokenType string) (string, error) {
	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().Unix()
	claims := models.JWTClaims{
		Sub: username,
		Iat: now,
		Exp: now + int64(getJWTExpiry().Seconds()),
		Jti: generateRandomString(16),
		Typ: tokenType,
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64URLEncode(claimsJSON)
	headerPayload := header + "." + payload
	signature := signJWT(headerPayload)
	return headerPayload + "." + signature, nil
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return base64URLEncode([]byte(time.Now().Format(time.RFC3339Nano)))[:length]
	}
	return base64URLEncode(b)[:length]
}

func VerifyJWT(token string) (*models.JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	headerPayload := parts[0] + "." + parts[1]
	expectedSig := signJWT(headerPayload)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, errors.New("invalid signature")
	}
	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, err
	}
	var claims models.JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, err
	}
	if claims.Exp < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func ExtractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func ExtractSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: requestIsHTTPS(r), SameSite: http.SameSiteStrictMode,
		MaxAge: int(getJWTExpiry().Seconds()), Expires: time.Now().Add(getJWTExpiry()),
	})
}

func ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: requestIsHTTPS(r), SameSite: http.SameSiteStrictMode,
		MaxAge: -1, Expires: time.Unix(1, 0),
	})
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func AuthenticateRequest(r *http.Request) (*models.JWTClaims, string, error) {
	token := ExtractBearerToken(r)
	source := "bearer"
	if token == "" {
		token = ExtractSessionToken(r)
		source = "cookie"
	}
	if token == "" {
		return nil, "", errors.New("missing token")
	}
	claims, err := VerifyJWT(token)
	if err != nil {
		return nil, source, err
	}
	if claims.Typ != "access" {
		return nil, source, errors.New("invalid token type")
	}
	return claims, source, nil
}

func RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	claims, source, err := AuthenticateRequest(r)
	if err != nil {
		if strings.Contains(err.Error(), "missing token") {
			writeAuthError(w, http.StatusUnauthorized, "Unauthorized")
		} else {
			writeAuthError(w, http.StatusUnauthorized, "Invalid token")
		}
		return false
	}
	if source == "cookie" && requestMutates(r) && !sameOriginRequest(r) {
		writeAuthError(w, http.StatusForbidden, "Cross-site request rejected")
		return false
	}
	cfg := config.GetConfig()
	if cfg != nil && cfg.AdminToken != "" {
		expectedUsername := "vantalens"
		if strings.TrimSpace(cfg.AdminUsername) != "" {
			expectedUsername = strings.TrimSpace(cfg.AdminUsername)
		}
		if claims.Sub != expectedUsername {
			writeAuthError(w, http.StatusForbidden, "Forbidden")
			return false
		}
	}
	return true
}

func requestMutates(r *http.Request) bool {
	return r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
}

func sameOriginRequest(r *http.Request) bool {
	raw := strings.TrimSpace(r.Header.Get("Origin"))
	if raw == "" {
		raw = strings.TrimSpace(r.Header.Get("Referer"))
	}
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.APIResponse{Success: false, Message: message})
}

func WithAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !RequireAuth(w, r) {
			return
		}
		handler(w, r)
	}
}
