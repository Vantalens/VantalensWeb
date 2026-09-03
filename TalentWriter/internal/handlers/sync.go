package handlers

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/comment"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/dbsync"
	"vantalens/talentwriter/internal/models"
)

var (
	syncServiceMu sync.RWMutex
	syncService   *dbsync.Service
)

func SetSyncService(service *dbsync.Service) {
	syncServiceMu.Lock()
	syncService = service
	syncServiceMu.Unlock()
}

func getSyncService() *dbsync.Service {
	syncServiceMu.RLock()
	defer syncServiceMu.RUnlock()
	return syncService
}

func HandleSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	service := getSyncService()
	var cache interface{}
	if service != nil {
		cache = service.Status()
	} else {
		cache = map[string]interface{}{
			"enabled": false,
			"error":   "Database sync is not configured",
		}
	}
	remote := map[string]interface{}{
		"enabled": remoteAdminConfigured(),
		"base":    remoteAdminBase(),
	}
	if remoteAdminConfigured() {
		result, err := proxyRemoteAdmin(r, http.MethodGet, "/api/admin/data/status", nil)
		remote["http_status"] = result.StatusCode
		remote["duration"] = result.Duration.String()
		if err != nil {
			remote["reachable"] = false
			remote["error"] = err.Error()
			remote["response"] = result.Response
		} else {
			remote["reachable"] = true
			remote["response"] = result.Response.Data
		}
	}
	data := map[string]interface{}{
		"remote": remote,
		"cache":  cache,
	}
	if service != nil {
		status := service.Status()
		data["enabled"] = status.Enabled
		data["host"] = status.Host
		data["interval"] = status.Interval
		data["running"] = status.Running
		data["databases"] = status.Databases
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: data})
}

func HandleSyncRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	service := getSyncService()
	if service == nil {
		RespondJSON(w, http.StatusServiceUnavailable, models.APIResponse{Success: false, Message: "Database sync is not configured"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: service.RunOnce(ctx)})
}

type commentWritebackStage struct {
	Stage   string `json:"stage"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func runCommentWriteback(ctx context.Context, action func() error) ([]commentWritebackStage, error) {
	stages := []commentWritebackStage{}
	addStage := func(stage string, success bool, message string) {
		stages = append(stages, commentWritebackStage{Stage: stage, Success: success, Message: message})
	}

	service := getSyncService()
	if service == nil || !service.Status().Enabled {
		if err := action(); err != nil {
			addStage("local_update", false, err.Error())
			return stages, err
		}
		addStage("local_update", true, "DB sync is disabled or not configured; local database updated only")
		return stages, nil
	}

	pulled := service.RunDatabase(ctx, "comments")
	if err := statusError(pulled, "comments"); err != nil {
		addStage("pull_latest_comments", false, err.Error())
		return stages, err
	}
	addStage("pull_latest_comments", true, "latest server comments database downloaded")

	if err := action(); err != nil {
		addStage("local_update", false, err.Error())
		return stages, err
	}
	addStage("local_update", true, "local comments database updated")

	if err := comment.Close(); err != nil {
		addStage("close_comments_db", false, err.Error())
		return stages, err
	}
	addStage("close_comments_db", true, "local comments connection closed before upload")

	pushed := service.PushDatabase(ctx, "comments")
	if pushed.LastError != "" || !pushed.Success {
		if err := reinitComments(); err != nil {
			addStage("reopen_comments_db", false, err.Error())
		}
		err := statusToError(pushed, "push_comments_to_server")
		addStage("push_comments_to_server", false, err.Error())
		return stages, err
	}
	addStage("push_comments_to_server", true, "server comments database replaced after remote backup")

	if !strings.EqualFold(strings.TrimSpace(config.GetEnv("COMMENT_WRITEBACK_VERIFY", "false")), "true") {
		if err := reinitComments(); err != nil {
			addStage("reopen_comments_db", false, err.Error())
			return stages, err
		}
		addStage("reopen_comments_db", true, "local comments connection reopened without extra verification pull")
		return stages, nil
	}

	verified := service.RunDatabase(ctx, "comments")
	if err := statusError(verified, "comments"); err != nil {
		addStage("verify_server_comments", false, err.Error())
		return stages, err
	}
	addStage("verify_server_comments", true, "server comments database downloaded again for verification")
	return stages, nil
}

func runRemoteCommentMutation(ctx context.Context, sql string, localFallback func() error) ([]commentWritebackStage, error) {
	stages := []commentWritebackStage{}
	addStage := func(stage string, success bool, message string) {
		stages = append(stages, commentWritebackStage{Stage: stage, Success: success, Message: message})
	}

	if truthy(config.GetEnv("AUTHORITY_BACKEND", "false")) {
		if err := localFallback(); err != nil {
			addStage("authority_local_transaction", false, err.Error())
			return stages, err
		}
		addStage("authority_local_transaction", true, "authority backend updated comments database directly")
		return stages, nil
	}

	service := getSyncService()
	if service == nil || !service.Status().Enabled {
		if err := localFallback(); err != nil {
			addStage("local_update", false, err.Error())
			return stages, err
		}
		addStage("local_update", true, "DB sync is disabled or not configured; local database updated only")
		return stages, nil
	}

	remote := service.ApplyRemoteSQLite(ctx, "comments", sql)
	if remote.LastError != "" || !remote.Success {
		err := statusToError(remote, "remote_comments_update")
		addStage("remote_comments_update", false, err.Error())
		return stages, err
	}
	addStage("remote_comments_update", true, "server comments.db updated in place through sqlite transaction")

	pulled := service.RunDatabase(ctx, "comments")
	if err := statusError(pulled, "comments"); err != nil {
		addStage("pull_updated_comments", false, err.Error())
		return stages, nil
	}
	addStage("pull_updated_comments", true, "updated server comments.db downloaded locally")
	return stages, nil
}

func reinitComments() error {
	cfg := config.GetConfig()
	if cfg == nil {
		return nil
	}
	return comment.Init(cfg.HugoPath)
}

func statusError(status dbsync.Status, name string) error {
	for _, db := range status.Databases {
		if db.Name != name {
			continue
		}
		if db.LastError != "" {
			return statusToError(db, name)
		}
		if !db.Success {
			return statusToError(db, name)
		}
		return nil
	}
	return statusToError(dbsync.DatabaseStatus{Name: name, LastError: "database status not found"}, name)
}

func statusToError(status dbsync.DatabaseStatus, fallback string) error {
	message := status.LastError
	if message == "" {
		message = "operation did not report success"
	}
	return &syncStageError{stage: fallback, message: message}
}

type syncStageError struct {
	stage   string
	message string
}

func (e *syncStageError) Error() string {
	return e.stage + ": " + e.message
}
