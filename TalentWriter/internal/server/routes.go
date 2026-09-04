package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"vantalens/talentwriter/internal/handlers"
)

type BuildInfo struct {
	Version   string `json:"version"`
	GitSHA    string `json:"git_sha"`
	BuildTime string `json:"build_time"`
	Dirty     string `json:"dirty"`
}

type Mode string

const (
	ModeAll Mode = "all"
)

func BuildMux(mode Mode, version string) *http.ServeMux {
	return BuildMuxWithInfo(mode, BuildInfo{Version: version})
}

func BuildMuxWithInfo(mode Mode, build BuildInfo) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/login", handlers.WithCORS(handlers.HandleLogin))
	mux.HandleFunc("/api/logout", handlers.WithCORS(handlers.HandleLogout))
	mux.HandleFunc("/health", healthHandler(build))
	mux.HandleFunc("/api/health", healthHandler(build))
	mux.HandleFunc("/api", apiInfoHandler(mode, build))

	registerControlRoutes(mux)
	registerWriterRoutes(mux)
	mux.HandleFunc("/", rootHandler("/platform", mode, build))

	return mux
}

func registerControlRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/control/status", handlers.WithCORS(handlers.HandleControlStatus))
	mux.HandleFunc("/api/control/command", handlers.WithCORS(handlers.HandleControlCommand))
	mux.HandleFunc("/api/analytics/stats", handlers.WithCORS(handlers.HandleAnalyticsStats))
	mux.HandleFunc("/api/admin/analytics/stats", handlers.WithCORS(handlers.HandleAdminAnalyticsStats))
	mux.HandleFunc("/api/admin/comments", handlers.WithCORS(handlers.HandleAdminComments))
	mux.HandleFunc("/api/admin/comments/", handlers.WithCORS(handlers.HandleAdminCommentAction))
	mux.HandleFunc("/api/admin/data/status", handlers.WithCORS(handlers.HandleAdminDataStatus))
	mux.HandleFunc("/api/sync/status", handlers.WithCORS(handlers.HandleSyncStatus))
	mux.HandleFunc("/api/sync/run", handlers.WithCORS(handlers.HandleSyncRun))
	mux.HandleFunc("/platform/session", handlers.HandlePlatformSession)
	mux.HandleFunc("/platform", handlers.HandlePlatformPage)
	mux.HandleFunc("/platform/", handlers.HandlePlatformSlash)
	mux.HandleFunc("/platform/control", handlers.HandleControlPage)
	mux.HandleFunc("/platform/analytics", handlers.HandleAnalyticsPage)
	mux.HandleFunc("/vendor/world-map-equirectangular.svg", handlers.HandleVendorWorldMap)
	mux.HandleFunc("/vendor/world-map-equirectangular.LICENSE.txt", handlers.HandleVendorWorldMapLicense)
}

func registerWriterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/analytics/collect", handlers.WithCORS(handlers.HandleAnalyticsCollect))
	mux.HandleFunc("/api/posts", handlers.WithCORS(handlers.HandleGetPosts))
	mux.HandleFunc("/api/get_content", handlers.WithCORS(handlers.HandleGetContent))
	mux.HandleFunc("/api/save_content", handlers.WithCORS(handlers.HandleSaveContent))
	mux.HandleFunc("/api/delete_post", handlers.WithCORS(handlers.HandleDeletePost))
	mux.HandleFunc("/api/create_post", handlers.WithCORS(handlers.HandleCreatePost))
	mux.HandleFunc("/api/trash/posts", handlers.WithCORS(handlers.HandleTrashPosts))
	mux.HandleFunc("/api/restore_post", handlers.WithCORS(handlers.HandleRestorePost))
	mux.HandleFunc("/api/purge_post", handlers.WithCORS(handlers.HandlePurgePost))
	mux.HandleFunc("/api/publish", handlers.WithCORS(handlers.HandlePublish))
	mux.HandleFunc("/api/publish/status", handlers.WithCORS(handlers.HandlePublishStatus))
	mux.HandleFunc("/api/comments", handlers.WithCORS(handlers.HandleGetComments))
	mux.HandleFunc("/api/comments/challenge", handlers.WithCORS(handlers.HandleCommentChallenge))
	mux.HandleFunc("/api/comments/email-code", handlers.WithCORS(handlers.HandleCommentEmailCode))
	mux.HandleFunc("/api/comments/add", handlers.WithCORS(handlers.HandleAddComment))
	mux.HandleFunc("/api/comments/approve", handlers.WithCORS(handlers.HandleApproveComment))
	mux.HandleFunc("/api/comments/delete", handlers.WithCORS(handlers.HandleDeleteComment))
	mux.HandleFunc("/api/settings", handlers.WithCORS(handlers.HandleGetSettings))
	mux.HandleFunc("/api/settings/save", handlers.WithCORS(handlers.HandleSaveSettings))
	mux.HandleFunc("/api/oplog/list", handlers.WithCORS(handlers.HandleOplogList))
	mux.HandleFunc("/api/oplog/entry", handlers.WithCORS(handlers.HandleOplogEntry))
	mux.HandleFunc("/api/oplog/rollback", handlers.WithCORS(handlers.HandleOplogRollback))
	mux.HandleFunc("/api/oplog/compare", handlers.WithCORS(handlers.HandleOplogCompare))
	mux.HandleFunc("/api/oplog/sync", handlers.WithCORS(handlers.HandleOplogSync))
	mux.HandleFunc("/platform/posts", handlers.HandlePostsPage)
	mux.HandleFunc("/platform/comments", handlers.HandleCommentsPage)
	mux.HandleFunc("/platform/history", handlers.HandleHistoryPage)
}

func rootHandler(defaultPath string, mode Mode, build BuildInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if wantsJSON(r) {
			apiInfoHandler(mode, build)(w, r)
			return
		}
		http.Redirect(w, r, defaultPath, http.StatusTemporaryRedirect)
	}
}

func healthHandler(build BuildInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Status string `json:"status"`
			BuildInfo
		}{Status: "ok", BuildInfo: build})
	}
}

func apiInfoHandler(mode Mode, build BuildInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		endpoints := []string{
			"/api/login",
			"/health",
			"/api/health",
			"/api",
			"/api/control/status",
			"/api/control/command",
			"/api/analytics/stats",
			"/api/admin/analytics/stats",
			"/api/admin/comments",
			"/api/admin/comments/{id}/approve",
			"/api/admin/comments/{id}/delete",
			"/api/admin/data/status",
			"/api/sync/status",
			"/api/sync/run",
			"/api/posts",
			"/api/get_content",
			"/api/save_content",
			"/api/delete_post",
			"/api/create_post",
			"/api/trash/posts",
			"/api/restore_post",
			"/api/purge_post",
			"/api/publish",
			"/api/publish/status",
			"/api/comments",
			"/api/comments/add",
			"/api/comments/approve",
			"/api/comments/delete",
			"/api/settings",
			"/api/settings/save",
			"/api/oplog/list",
			"/api/oplog/entry",
			"/api/oplog/rollback",
			"/api/oplog/compare",
			"/api/oplog/sync",
			"/platform",
			"/platform/control",
			"/platform/posts",
			"/platform/comments",
			"/platform/history",
			"/platform/analytics",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":       "Vantalens Writer API",
			"mode":       string(mode),
			"version":    build.Version,
			"git_sha":    build.GitSHA,
			"build_time": build.BuildTime,
			"dirty":      build.Dirty,
			"endpoints":  endpoints,
		})
	}
}

func wantsJSON(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "application/json") || r.URL.Query().Get("format") == "json"
}
