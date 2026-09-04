package handlers

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vantalens/talentwriter/internal/analytics"
	"vantalens/talentwriter/internal/comment"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

func HandleAdminComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	comments, err := comment.GetAllComments()
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "authority comments loaded", Data: comments})
}

func HandleAdminCommentAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	id, action, ok := parseAdminCommentAction(r.URL.Path)
	if !ok {
		RespondJSON(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "admin comment action not found"})
		return
	}
	stages := []commentWritebackStage{}
	addStage := func(stage string, success bool, message string) {
		stages = append(stages, commentWritebackStage{Stage: stage, Success: success, Message: message})
	}

	cfg := config.GetConfig()
	hugoPath := ""
	if cfg != nil {
		hugoPath = cfg.HugoPath
	}
	backup, err := comment.BackupDatabase(hugoPath)
	if err != nil {
		addStage("backup_remote_comments_db", false, err.Error())
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error(), Data: stages})
		return
	}
	addStage("backup_remote_comments_db", true, backup)

	snapshot := snapshotCommentByID(id)
	switch action {
	case "approve":
		err = comment.ApproveComment("", id)
	case "delete":
		err = comment.DeleteComment("", id)
	default:
		err = http.ErrNotSupported
	}
	if err != nil {
		recordOp(r, "comment."+action, id, "评论"+action+" "+id, snapshot, err, opSyncedByDefault())
		addStage("execute_remote_transaction", false, err.Error())
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		RespondJSON(w, status, models.APIResponse{Success: false, Message: err.Error(), Data: stages})
		return
	}
	recordOp(r, "comment."+action, id, "评论"+action+" "+id, snapshot, nil, opSyncedByDefault())
	addStage("execute_remote_transaction", true, action+" succeeded")
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "remote comment " + action + " succeeded", Data: stages})
}

func parseAdminCommentAction(path string) (string, string, bool) {
	rest := strings.TrimPrefix(path, "/api/admin/comments/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		return "", "", false
	}
	id, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", false
	}
	action := strings.TrimSpace(parts[1])
	return id, action, action == "approve" || action == "delete"
}

func HandleAdminAnalyticsStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	stats, err := analytics.GetSiteStatistics(limit)
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "authority analytics loaded", Data: stats})
}

func HandleAdminDataStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	cfg := config.GetConfig()
	hugoPath := ""
	if cfg != nil {
		hugoPath = cfg.HugoPath
	}
	data := map[string]interface{}{
		"role":      "authority",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"databases": map[string]interface{}{
			"comments": databaseFileStatus(config.GetCommentsDBPath(hugoPath)),
			"visits":   databaseFileStatus(config.GetAnalyticsDBPath(hugoPath)),
			"articles": databaseFileStatus(config.GetArticlesDBPath(hugoPath)),
		},
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: data})
}

func databaseFileStatus(path string) map[string]interface{} {
	info, err := os.Stat(path)
	status := map[string]interface{}{
		"path": filepath.Clean(path),
	}
	if err != nil {
		status["exists"] = false
		status["error"] = err.Error()
		return status
	}
	status["exists"] = true
	status["size"] = info.Size()
	status["modified_at"] = info.ModTime().UTC().Format(time.RFC3339)
	return status
}
