package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegacyBackendRouteIsRemoved(t *testing.T) {
	mux := BuildMux(ModeAll, "test")

	req := httptest.NewRequest(http.MethodGet, "/platform/backend", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /platform/backend status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAPIInfoDoesNotAdvertiseRemovedRoutes(t *testing.T) {
	mux := BuildMux(ModeAll, "test")

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Endpoints []string `json:"endpoints"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api response: %v", err)
	}

	removed := map[string]bool{
		"/platform/backend": true,
		"/api/create_sync":  true,
	}
	for _, endpoint := range payload.Endpoints {
		if removed[endpoint] {
			t.Fatalf("/api still advertises removed endpoint %q", endpoint)
		}
		if strings.Contains(endpoint, "backend") {
			t.Fatalf("/api still advertises backend legacy endpoint %q", endpoint)
		}
	}
}
