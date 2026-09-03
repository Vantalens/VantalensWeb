package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

// initTestSecret installs a deterministic JWT secret for the test.
func initTestSecret(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "unit-test-secret")
	if err := InitJWTSecret(); err != nil {
		t.Fatalf("InitJWTSecret: %v", err)
	}
}

// unsetEnv removes a key for the duration of the test, restoring it afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	orig, had := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, orig)
		}
	})
}

func TestInitJWTSecretUsesConfiguredSecret(t *testing.T) {
	initTestSecret(t)
	if string(jwtSecret) != "unit-test-secret" {
		t.Fatalf("jwtSecret = %q, want configured secret", string(jwtSecret))
	}
}

func TestInitJWTSecretGeneratesRandomSecretInDev(t *testing.T) {
	unsetEnv(t, "JWT_SECRET")
	if err := InitJWTSecret(); err != nil {
		t.Fatalf("InitJWTSecret: %v", err)
	}
	if len(jwtSecret) == 0 {
		t.Fatal("jwtSecret must be generated when JWT_SECRET is unset in dev mode")
	}
}

func TestInitJWTSecretFailsInProductionWithoutSecret(t *testing.T) {
	unsetEnv(t, "JWT_SECRET")
	for _, key := range []string{"AUTHORITY_BACKEND", "BEHIND_PROXY", "REQUIRE_PERSISTENT_JWT_SECRET"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "true")
			if err := InitJWTSecret(); err == nil {
				t.Fatalf("InitJWTSecret with %s=true and no JWT_SECRET returned nil error", key)
			}
		})
	}
}

func TestIsProductionAuthRequiredValues(t *testing.T) {
	t.Setenv("REQUIRE_PERSISTENT_JWT_SECRET", "YES")
	if !isProductionAuthRequired() {
		t.Fatal(`isProductionAuthRequired with "YES" = false, want true`)
	}
	t.Setenv("REQUIRE_PERSISTENT_JWT_SECRET", "no")
	t.Setenv("AUTHORITY_BACKEND", "")
	t.Setenv("BEHIND_PROXY", "off")
	if isProductionAuthRequired() {
		t.Fatal("isProductionAuthRequired with only falsy values = true, want false")
	}
}

func TestCreateAndVerifyJWTRoundTrip(t *testing.T) {
	initTestSecret(t)
	before := time.Now().Unix()

	token, err := CreateJWT("alice", "access")
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}
	if got := len(strings.Split(token, ".")); got != 3 {
		t.Fatalf("token segment count = %d, want 3", got)
	}

	claims, err := VerifyJWT(token)
	if err != nil {
		t.Fatalf("VerifyJWT: %v", err)
	}
	if claims.Sub != "alice" {
		t.Fatalf("claims.Sub = %q, want %q", claims.Sub, "alice")
	}
	if claims.Typ != "access" {
		t.Fatalf("claims.Typ = %q, want %q", claims.Typ, "access")
	}
	if claims.Jti == "" {
		t.Fatal("claims.Jti must be non-empty")
	}
	if claims.Iat < before || claims.Iat > time.Now().Unix() {
		t.Fatalf("claims.Iat = %d, want within [%d, %d]", claims.Iat, before, time.Now().Unix())
	}
	if want := claims.Iat + int64(getJWTExpiry().Seconds()); claims.Exp != want {
		t.Fatalf("claims.Exp = %d, want iat + expiry = %d", claims.Exp, want)
	}

	// Header must declare HS256.
	headerJSON, err := base64URLDecode(strings.Split(token, ".")[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	var header map[string]string
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if header["alg"] != "HS256" || header["typ"] != "JWT" {
		t.Fatalf("header = %v, want alg=HS256 typ=JWT", header)
	}
}

func TestVerifyJWTRejectsMalformedTokens(t *testing.T) {
	initTestSecret(t)

	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"single segment", "abc"},
		{"two segments", "a.b"},
		{"four segments", "a.b.c.d"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := VerifyJWT(tc.token); err == nil || !strings.Contains(err.Error(), "invalid token format") {
				t.Fatalf("VerifyJWT(%q) err = %v, want invalid token format", tc.token, err)
			}
		})
	}
}

func TestVerifyJWTRejectsTamperedSignature(t *testing.T) {
	initTestSecret(t)

	token, err := CreateJWT("alice", "access")
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}
	parts := strings.Split(token, ".")
	parts[2] = base64URLEncode([]byte("forged-signature-value"))
	if _, err := VerifyJWT(strings.Join(parts, ".")); err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("VerifyJWT with forged signature err = %v, want invalid signature", err)
	}
}

func TestVerifyJWTRejectsTamperedPayload(t *testing.T) {
	initTestSecret(t)

	token, err := CreateJWT("alice", "access")
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}
	parts := strings.Split(token, ".")
	// Re-encode a different payload without re-signing.
	forged := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(`{"sub":"mallory","iat":1,"exp":9999999999,"jti":"x","typ":"access"}`))
	parts[1] = forged
	if _, err := VerifyJWT(strings.Join(parts, ".")); err == nil {
		t.Fatal("VerifyJWT accepted a token with tampered payload")
	}
}

func TestVerifyJWTRejectsTokenSignedWithDifferentSecret(t *testing.T) {
	initTestSecret(t)
	token, err := CreateJWT("alice", "access")
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}

	original := jwtSecret
	jwtSecret = []byte("another-secret")
	t.Cleanup(func() { jwtSecret = original })

	if _, err := VerifyJWT(token); err == nil {
		t.Fatal("VerifyJWT accepted a token signed with a different secret")
	}
}

func TestVerifyJWTRejectsExpiredToken(t *testing.T) {
	initTestSecret(t)

	now := time.Now().Unix()
	claims := models.JWTClaims{Sub: "alice", Iat: now - 7200, Exp: now - 3600, Jti: "x", Typ: "access"}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	headerPayload := header + "." + base64URLEncode(claimsJSON)
	token := headerPayload + "." + signJWT(headerPayload)

	if _, err := VerifyJWT(token); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("VerifyJWT expired token err = %v, want token expired", err)
	}
}

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"missing", "", ""},
		{"valid", "Bearer abc.def.ghi", "abc.def.ghi"},
		{"wrong scheme", "Basic abc", ""},
		{"lowercase scheme rejected", "bearer abc", ""},
		{"empty value", "Bearer ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			if got := ExtractBearerToken(req); got != tc.want {
				t.Fatalf("ExtractBearerToken = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractSessionToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := ExtractSessionToken(req); got != "" {
		t.Fatalf("ExtractSessionToken without cookie = %q, want empty", got)
	}

	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "  tok123  "})
	if got := ExtractSessionToken(req); got != "tok123" {
		t.Fatalf("ExtractSessionToken = %q, want trimmed %q", got, "tok123")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(&http.Cookie{Name: "other", Value: "tok123"})
	if got := ExtractSessionToken(req2); got != "" {
		t.Fatalf("ExtractSessionToken with wrong cookie name = %q, want empty", got)
	}
}

func TestSetSessionCookie(t *testing.T) {
	t.Run("plain http", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		rec := httptest.NewRecorder()
		SetSessionCookie(rec, req, "tok")

		cookies := rec.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("cookie count = %d, want 1", len(cookies))
		}
		c := cookies[0]
		if c.Name != SessionCookieName || c.Value != "tok" {
			t.Fatalf("cookie = %+v, want name %q value tok", c, SessionCookieName)
		}
		if !c.HttpOnly {
			t.Fatal("cookie must be HttpOnly")
		}
		if c.Secure {
			t.Fatal("cookie over plain http must not be Secure")
		}
		if c.SameSite != http.SameSiteStrictMode {
			t.Fatalf("cookie SameSite = %v, want Strict", c.SameSite)
		}
		if c.Path != "/" {
			t.Fatalf("cookie Path = %q, want /", c.Path)
		}
		if c.MaxAge != int(getJWTExpiry().Seconds()) {
			t.Fatalf("cookie MaxAge = %d, want %d", c.MaxAge, int(getJWTExpiry().Seconds()))
		}
	})

	t.Run("behind https proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		SetSessionCookie(rec, req, "tok")

		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].Secure {
			t.Fatalf("cookie = %+v, want Secure behind https proxy", cookies)
		}
	})

	t.Run("direct tls", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
		rec := httptest.NewRecorder()
		SetSessionCookie(rec, req, "tok")

		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].Secure {
			t.Fatalf("cookie = %+v, want Secure over TLS", cookies)
		}
	})
}

func TestClearSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	ClearSessionCookie(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != SessionCookieName || c.Value != "" {
		t.Fatalf("cookie = %+v, want emptied session cookie", c)
	}
	if c.MaxAge != -1 {
		t.Fatalf("cookie MaxAge = %d, want -1", c.MaxAge)
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie flags = HttpOnly %v SameSite %v, want HttpOnly+Strict", c.HttpOnly, c.SameSite)
	}
}

func TestRequestIsHTTPS(t *testing.T) {
	plain := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if requestIsHTTPS(plain) {
		t.Fatal("requestIsHTTPS(plain http) = true, want false")
	}
	tlsReq := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	if !requestIsHTTPS(tlsReq) {
		t.Fatal("requestIsHTTPS(tls) = false, want true")
	}
	proxied := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	proxied.Header.Set("X-Forwarded-Proto", "HTTPS")
	if !requestIsHTTPS(proxied) {
		t.Fatal("requestIsHTTPS(X-Forwarded-Proto: HTTPS) = false, want true")
	}
}

func TestRequestMutates(t *testing.T) {
	safe := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, m := range safe {
		if requestMutates(httptest.NewRequest(m, "/", nil)) {
			t.Fatalf("requestMutates(%s) = true, want false", m)
		}
	}
	mutating := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, m := range mutating {
		if !requestMutates(httptest.NewRequest(m, "/", nil)) {
			t.Fatalf("requestMutates(%s) = false, want true", m)
		}
	}
}

func TestSameOriginRequest(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		referer string
		want   bool
	}{
		{"no headers", "", "", true},
		{"matching origin", "http://example.com", "", true},
		{"matching origin different case", "http://EXAMPLE.com", "", true},
		{"mismatched origin", "http://evil.com", "", false},
		{"mismatched port", "http://example.com:9999", "", false},
		{"referer fallback match", "", "http://example.com/posts", true},
		{"referer fallback mismatch", "", "http://evil.com/posts", false},
		{"unparseable origin", "://not-a-url", "", false},
		{"origin without host", "/relative/path", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com/api", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			if got := sameOriginRequest(req); got != tc.want {
				t.Fatalf("sameOriginRequest = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAuthenticateRequest(t *testing.T) {
	initTestSecret(t)

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if _, _, err := AuthenticateRequest(req); err == nil || !strings.Contains(err.Error(), "missing token") {
			t.Fatalf("err = %v, want missing token", err)
		}
	})

	t.Run("bearer token", func(t *testing.T) {
		token, err := CreateJWT("alice", "access")
		if err != nil {
			t.Fatalf("CreateJWT: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		claims, source, err := AuthenticateRequest(req)
		if err != nil {
			t.Fatalf("AuthenticateRequest: %v", err)
		}
		if source != "bearer" {
			t.Fatalf("source = %q, want bearer", source)
		}
		if claims.Sub != "alice" {
			t.Fatalf("claims.Sub = %q, want alice", claims.Sub)
		}
	})

	t.Run("session cookie", func(t *testing.T) {
		token, err := CreateJWT("alice", "access")
		if err != nil {
			t.Fatalf("CreateJWT: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
		_, source, err := AuthenticateRequest(req)
		if err != nil {
			t.Fatalf("AuthenticateRequest: %v", err)
		}
		if source != "cookie" {
			t.Fatalf("source = %q, want cookie", source)
		}
	})

	t.Run("refresh token rejected", func(t *testing.T) {
		token, err := CreateJWT("alice", "refresh")
		if err != nil {
			t.Fatalf("CreateJWT: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if _, _, err := AuthenticateRequest(req); err == nil || !strings.Contains(err.Error(), "invalid token type") {
			t.Fatalf("err = %v, want invalid token type", err)
		}
	})

	t.Run("invalid token reports source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "garbage"})
		_, source, err := AuthenticateRequest(req)
		if err == nil {
			t.Fatal("AuthenticateRequest with garbage cookie returned nil error")
		}
		if source != "cookie" {
			t.Fatalf("source = %q, want cookie", source)
		}
	})
}

func requireAuthStatus(t *testing.T, req *http.Request) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	allowed := RequireAuth(rec, req)
	status := rec.Code
	if allowed {
		status = http.StatusOK
	}
	return status, rec.Body.String()
}

func TestRequireAuth(t *testing.T) {
	initTestSecret(t)
	config.SetConfig(&config.Config{AdminUsername: "vantalens", AdminToken: "secret"})
	t.Cleanup(func() { config.SetConfig(nil) })

	newAccessToken := func(username string) string {
		t.Helper()
		token, err := CreateJWT(username, "access")
		if err != nil {
			t.Fatalf("CreateJWT: %v", err)
		}
		return token
	}

	t.Run("missing token is 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		status, body := requireAuthStatus(t, req)
		if status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
		}
		if !strings.Contains(body, "Unauthorized") {
			t.Fatalf("body = %q, want Unauthorized message", body)
		}
	})

	t.Run("invalid token is 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer garbage")
		status, body := requireAuthStatus(t, req)
		if status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
		}
		if !strings.Contains(body, "Invalid token") {
			t.Fatalf("body = %q, want Invalid token message", body)
		}
	})

	t.Run("valid bearer for admin user passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer "+newAccessToken("vantalens"))
		status, _ := requireAuthStatus(t, req)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
	})

	t.Run("wrong username is 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer "+newAccessToken("mallory"))
		status, body := requireAuthStatus(t, req)
		if status != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
		}
		if !strings.Contains(body, "Forbidden") {
			t.Fatalf("body = %q, want Forbidden message", body)
		}
	})

	t.Run("cookie POST cross-site is 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: newAccessToken("vantalens")})
		req.Header.Set("Origin", "http://evil.com")
		status, body := requireAuthStatus(t, req)
		if status != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
		}
		if !strings.Contains(body, "Cross-site request rejected") {
			t.Fatalf("body = %q, want cross-site rejection message", body)
		}
	})

	t.Run("cookie POST same-origin passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: newAccessToken("vantalens")})
		req.Header.Set("Origin", "http://example.com")
		status, _ := requireAuthStatus(t, req)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
	})

	t.Run("cookie GET cross-origin passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: newAccessToken("vantalens")})
		req.Header.Set("Origin", "http://evil.com")
		status, _ := requireAuthStatus(t, req)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d (GET is not a mutation)", status, http.StatusOK)
		}
	})
}

func TestRequireAuthSkipsUsernameCheckWithoutAdminToken(t *testing.T) {
	initTestSecret(t)
	config.SetConfig(&config.Config{AdminUsername: "vantalens"}) // AdminToken empty
	t.Cleanup(func() { config.SetConfig(nil) })

	token, err := CreateJWT("anyone", "access")
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	if !RequireAuth(rec, req) {
		t.Fatalf("RequireAuth rejected request when AdminToken is empty, body=%s", rec.Body.String())
	}
}

func TestRequireAuthDefaultsExpectedUsername(t *testing.T) {
	initTestSecret(t)
	config.SetConfig(&config.Config{AdminToken: "secret"}) // AdminUsername empty -> "vantalens"
	t.Cleanup(func() { config.SetConfig(nil) })

	token, err := CreateJWT("vantalens", "access")
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	if !RequireAuth(rec, req) {
		t.Fatalf("RequireAuth rejected default admin username, body=%s", rec.Body.String())
	}
}

func TestWithAuth(t *testing.T) {
	initTestSecret(t)
	config.SetConfig(nil)

	called := false
	handler := WithAuth(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	// Unauthorized request must not reach the inner handler.
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if called {
		t.Fatal("inner handler called for unauthorized request")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	token, err := CreateJWT("alice", "access")
	if err != nil {
		t.Fatalf("CreateJWT: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler(rec, req)
	if !called {
		t.Fatal("inner handler not called for authorized request")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
