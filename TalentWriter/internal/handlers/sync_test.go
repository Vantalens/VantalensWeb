package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/dbsync"
	"vantalens/talentwriter/internal/models"
)

func TestHandleSyncStatusRequiresAuth(t *testing.T) {
	SetSyncService(dbsync.NewService(dbsync.Config{
		Enabled:    true,
		RemoteHost: "wj",
		Interval:   5 * time.Minute,
	}, nil))
	t.Cleanup(func() { SetSyncService(nil) })

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	rr := httptest.NewRecorder()

	HandleSyncStatus(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleSyncStatusReturnsConfiguredServiceState(t *testing.T) {
	config.SetConfig(&config.Config{AdminUsername: "vantalens", AdminToken: "secret"})
	auth.InitJWTSecret()
	token, err := auth.CreateJWT("vantalens", "access")
	if err != nil {
		t.Fatal(err)
	}
	SetSyncService(dbsync.NewService(dbsync.Config{
		Enabled:    true,
		RemoteHost: "wj",
		Interval:   5 * time.Minute,
		Databases: []dbsync.DatabaseSpec{{
			Name:       "analytics",
			RemotePath: "/var/lib/vantalens/analytics/visits.db",
			LocalPath:  "local/visits.db",
		}},
	}, nil))
	t.Cleanup(func() { SetSyncService(nil) })

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	HandleSyncStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("success = false, body=%s", rr.Body.String())
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok || data["host"] != "wj" {
		t.Fatalf("data = %#v, want host wj", resp.Data)
	}
}

func TestHandleSyncRunTriggersManualSync(t *testing.T) {
	config.SetConfig(&config.Config{AdminUsername: "vantalens", AdminToken: "secret"})
	auth.InitJWTSecret()
	token, err := auth.CreateJWT("vantalens", "access")
	if err != nil {
		t.Fatal(err)
	}
	SetSyncService(dbsync.NewService(dbsync.Config{
		Enabled:    true,
		RemoteHost: "wj",
		Interval:   5 * time.Minute,
	}, nil))
	t.Cleanup(func() { SetSyncService(nil) })

	req := httptest.NewRequest(http.MethodPost, "/api/sync/run", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	HandleSyncRun(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}
