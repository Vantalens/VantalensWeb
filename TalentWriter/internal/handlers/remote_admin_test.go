package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

func testAdminToken(t *testing.T) string {
	t.Helper()
	config.SetConfig(&config.Config{AdminUsername: "vantalens", AdminToken: "secret"})
	auth.InitJWTSecret()
	token, err := auth.CreateJWT("vantalens", "access")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestHandleAnalyticsStatsUsesRemoteAuthority(t *testing.T) {
	token := testAdminToken(t)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/admin/analytics/stats" {
			t.Fatalf("remote path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "25" {
			t.Fatalf("limit = %q, want 25", got)
		}
		RespondJSON(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data:    models.SiteStatistics{TotalViews: 42, UniqueIPs: 7},
		})
	}))
	defer remote.Close()
	t.Setenv("REMOTE_ADMIN_BASE", remote.URL)
	t.Setenv("LOCAL_CACHE_ENABLED", "false")

	req := httptest.NewRequest(http.MethodGet, "/api/analytics/stats?limit=25", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	HandleAnalyticsStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data := resp.Data.(map[string]interface{})
	if data["total_views"].(float64) != 42 {
		t.Fatalf("data = %#v, want remote stats", data)
	}
}

func TestHandleApproveCommentProxiesRemoteAuthority(t *testing.T) {
	token := testAdminToken(t)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/admin/comments/c-1/approve" {
			t.Fatalf("remote path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "approved"})
	}))
	defer remote.Close()
	t.Setenv("REMOTE_ADMIN_BASE", remote.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/comments/approve?id=c-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	HandleApproveComment(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || !strings.Contains(resp.Message, "服务器权威后端") {
		t.Fatalf("response = %#v", resp)
	}
}

func TestHandleDeleteCommentRemoteFailureDoesNotFakeSuccess(t *testing.T) {
	token := testAdminToken(t)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: "sqlite locked"})
	}))
	defer remote.Close()
	t.Setenv("REMOTE_ADMIN_BASE", remote.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/comments/delete?id=c-2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	HandleDeleteComment(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Success || !strings.Contains(resp.Message, "未执行本地写入兜底") {
		t.Fatalf("response = %#v", resp)
	}
}
