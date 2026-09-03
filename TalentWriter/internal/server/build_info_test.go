package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthExposesBuildInformation(t *testing.T) {
	want := BuildInfo{Version: "2.0.0", GitSHA: "abc123", BuildTime: "2026-08-12T00:00:00Z", Dirty: "false"}
	mux := BuildMuxWithInfo(ModeAll, want)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("health status = %d", recorder.Code)
	}
	var payload struct {
		Status string `json:"status"`
		BuildInfo
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "ok" || payload.BuildInfo != want {
		t.Fatalf("health payload = %+v, want build %+v", payload, want)
	}
}
