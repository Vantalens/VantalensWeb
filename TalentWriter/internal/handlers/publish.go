package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

type PublishStatus struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	OutputPath string `json:"output_path,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

var (
	publishMu      sync.RWMutex
	currentPublish = PublishStatus{State: "idle"}
)

func HandlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}

	hugoPath := articleRootDir()
	outputPath := strings.TrimSpace(config.GetEnv("PUBLISH_OUTPUT_PATH", filepath.Join(hugoPath, "public")))
	hugoPath, outputPath, err := validatePublishPaths(hugoPath, outputPath)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}

	publishMu.Lock()
	if currentPublish.State == "running" {
		status := currentPublish
		publishMu.Unlock()
		RespondJSON(w, http.StatusConflict, models.APIResponse{Success: false, Message: "A publish is already running", Data: status})
		return
	}
	status := PublishStatus{
		ID:         time.Now().UTC().Format("20060102T150405.000000000Z"),
		State:      "running",
		OutputPath: outputPath,
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	currentPublish = status
	publishMu.Unlock()

	go runPublish(status.ID, hugoPath, outputPath)
	RespondJSON(w, http.StatusAccepted, models.APIResponse{Success: true, Message: "Publish started", Data: status})
}

func HandlePublishStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	publishMu.RLock()
	status := currentPublish
	publishMu.RUnlock()
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: status})
}

func runPublish(id, hugoPath, outputPath string) {
	command := resolveHugoCommand(hugoPath)
	result := runCommand(hugoPath, 3*time.Minute, command[0], append(command[1:], "--minify", "--cleanDestinationDir", "--destination", outputPath)...)

	publishMu.Lock()
	defer publishMu.Unlock()
	if currentPublish.ID != id {
		return
	}
	currentPublish.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	currentPublish.Output = limitPublishOutput(result.Output, 32<<10)
	if result.Success {
		currentPublish.State = "succeeded"
		return
	}
	currentPublish.State = "failed"
	currentPublish.Error = "Hugo build failed"
}

func validatePublishPaths(hugoPath, outputPath string) (string, string, error) {
	root, err := filepath.Abs(strings.TrimSpace(hugoPath))
	if err != nil || strings.TrimSpace(hugoPath) == "" {
		return "", "", fmt.Errorf("invalid Hugo path")
	}
	output, err := filepath.Abs(strings.TrimSpace(outputPath))
	if err != nil || strings.TrimSpace(outputPath) == "" {
		return "", "", fmt.Errorf("invalid publish output path")
	}
	if filepath.Clean(root) == filepath.Clean(output) {
		return "", "", fmt.Errorf("publish output path must not equal the Hugo source root")
	}
	if output == filepath.VolumeName(output)+string(filepath.Separator) {
		return "", "", fmt.Errorf("publish output path must not be a filesystem root")
	}
	return root, output, nil
}

func limitPublishOutput(output string, limit int) string {
	output = strings.TrimSpace(output)
	if len(output) <= limit {
		return output
	}
	return output[:limit] + "\n...[truncated]"
}
