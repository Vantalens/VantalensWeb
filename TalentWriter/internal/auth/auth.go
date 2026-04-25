package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

var jwtSecret []byte

func InitJWTSecret() {
	secret := strings.TrimSpace(config.GetEnv("JWT_SECRET", ""))
	if secret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err == nil {
			secret = base64URLEncode(buf)
		} else {
			secret = generateRandomString(32)
		}
	}
	jwtSecret = []byte(secret)
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

func RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	token := ExtractBearerToken(r)
	if token == "" {
		writeAuthError(w, http.StatusUnauthorized, "Unauthorized")
		return false
	}
	claims, err := VerifyJWT(token)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "Invalid token")
		return false
	}
	if claims.Typ != "access" {
		writeAuthError(w, http.StatusUnauthorized, "Invalid token type")
		return false
	}
	cfg := config.GetConfig()
	if cfg != nil && cfg.AdminToken != "" {
		if claims.Sub != "admin" {
			writeAuthError(w, http.StatusForbidden, "Forbidden")
			return false
		}
	}
	return true
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
