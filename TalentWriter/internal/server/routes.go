package server

import (
    "encoding/json"
    "net/http"
    "strings"

    "vantalens/talentwriter/internal/handlers"
)

type Mode string

const (
    ModeAll     Mode = "all"
    ModeControl Mode = "control"
    ModeWriter  Mode = "writer"
)

func BuildMux(mode Mode, version string) *http.ServeMux {
    mux := http.NewServeMux()

    mux.HandleFunc("/api/login", handlers.WithCORS(handlers.HandleLogin))
    mux.HandleFunc("/health", healthHandler(version))
    mux.HandleFunc("/api/health", healthHandler(version))
    mux.HandleFunc("/api", apiInfoHandler(mode, version))

    switch mode {
    case ModeControl:
        registerControlRoutes(mux)
        mux.HandleFunc("/platform/backend", handlers.HandleWriterPageRedirect)
        mux.HandleFunc("/", rootHandler("/platform/control", mode, version))
    case ModeWriter:
        registerControlRoutes(mux)
        registerWriterRoutes(mux)
        mux.HandleFunc("/", rootHandler("/platform/backend", mode, version))
    default:
        registerControlRoutes(mux)
        registerWriterRoutes(mux)
        mux.HandleFunc("/", rootHandler("/platform/control", mode, version))
    }

    return mux
}

func registerControlRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/api/control/status", handlers.WithCORS(handlers.HandleControlStatus))
    mux.HandleFunc("/api/control/command", handlers.WithCORS(handlers.HandleControlCommand))
    mux.HandleFunc("/api/analytics/stats", handlers.WithCORS(handlers.HandleAnalyticsStats))
    mux.HandleFunc("/platform/control", handlers.HandleControlPage)
    mux.HandleFunc("/platform/analytics", handlers.HandleAnalyticsPage)
}

func registerWriterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/api/analytics/collect", handlers.WithCORS(handlers.HandleAnalyticsCollect))
    mux.HandleFunc("/api/posts", handlers.WithCORS(handlers.HandleGetPosts))
    mux.HandleFunc("/api/get_content", handlers.WithCORS(handlers.HandleGetContent))
    mux.HandleFunc("/api/save_content", handlers.WithCORS(handlers.HandleSaveContent))
    mux.HandleFunc("/api/delete_post", handlers.WithCORS(handlers.HandleDeletePost))
    mux.HandleFunc("/api/create_post", handlers.WithCORS(handlers.HandleCreatePost))
    mux.HandleFunc("/api/create_sync", handlers.WithCORS(handlers.HandleCreatePost))
    mux.HandleFunc("/api/comments", handlers.WithCORS(handlers.HandleGetComments))
    mux.HandleFunc("/api/comments/challenge", handlers.WithCORS(handlers.HandleCommentChallenge))
    mux.HandleFunc("/api/comments/email-code", handlers.WithCORS(handlers.HandleCommentEmailCode))
    mux.HandleFunc("/api/comments/add", handlers.WithCORS(handlers.HandleAddComment))
    mux.HandleFunc("/api/comments/approve", handlers.WithCORS(handlers.HandleApproveComment))
    mux.HandleFunc("/api/comments/delete", handlers.WithCORS(handlers.HandleDeleteComment))
    mux.HandleFunc("/api/settings", handlers.WithCORS(handlers.HandleGetSettings))
    mux.HandleFunc("/api/settings/save", handlers.WithCORS(handlers.HandleSaveSettings))
    mux.HandleFunc("/platform/backend", handlers.HandleBackendPage)
}

func rootHandler(defaultPath string, mode Mode, version string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.NotFound(w, r)
            return
        }
        if wantsJSON(r) {
            apiInfoHandler(mode, version)(w, r)
            return
        }
        http.Redirect(w, r, defaultPath, http.StatusTemporaryRedirect)
    }
}

func healthHandler(version string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]string{
            "status":  "ok",
            "version": version,
        })
    }
}

func apiInfoHandler(mode Mode, version string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        endpoints := []string{"/api/login", "/health", "/api/health", "/api"}

        if mode == ModeAll || mode == ModeControl || mode == ModeWriter {
            endpoints = append(endpoints,
                "/api/control/status",
                "/api/control/command",
                "/platform/control",
            )
        }
        if mode == ModeAll || mode == ModeWriter {
            endpoints = append(endpoints,
                "/api/posts",
                "/api/get_content",
                "/api/save_content",
                "/api/delete_post",
                "/api/create_post",
                "/api/create_sync",
                "/api/comments",
                "/api/comments/add",
                "/api/comments/approve",
                "/api/comments/delete",
                "/api/settings",
                "/api/settings/save",
                "/platform/backend",
            )
        }

        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]interface{}{
            "name":      "Vantalens Writer API",
            "mode":      string(mode),
            "version":   version,
            "endpoints": endpoints,
        })
    }
}

func wantsJSON(r *http.Request) bool {
    accept := strings.ToLower(r.Header.Get("Accept"))
    return strings.Contains(accept, "application/json") || r.URL.Query().Get("format") == "json"
}
