package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

func HandleControlPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/platform/control" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cfg := config.GetConfig()
	writerURL := "/platform/backend"
	if cfg != nil && cfg.LauncherMode != "all" && cfg.WriterPort > 0 {
		writerURL = config.LocalhostURL(cfg.WriterPort, "/platform/backend")
	}
	_, _ = w.Write([]byte(ControlHTML(writerURL)))
}

func HandleWriterPageRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/platform/backend" {
		http.NotFound(w, r)
		return
	}
	cfg := config.GetConfig()
	writerURL := "/platform/backend"
	if cfg != nil && cfg.LauncherMode != "all" && cfg.WriterPort > 0 {
		writerURL = config.LocalhostURL(cfg.WriterPort, "/platform/backend")
	}
	http.Redirect(w, r, writerURL, http.StatusTemporaryRedirect)
}

func HandleBackendPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/platform/backend" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cfg := config.GetConfig()
	controlURL := "/platform/control"
	if cfg != nil && cfg.LauncherMode != "all" && cfg.ControlPort > 0 {
		controlURL = config.LocalhostURL(cfg.ControlPort, "/platform/control")
	}
	_, _ = w.Write([]byte(DashboardHTML("2.0.0", controlURL)))
}

func HandleControlStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	cfg := config.GetConfig()
	hugoPath := ""
	if cfg != nil {
		hugoPath = cfg.HugoPath
	}

	hugoCommand := resolveHugoCommand(hugoPath)
	frontend := runCommand(hugoPath, 25*time.Second, hugoCommand[0], append(hugoCommand[1:], "version")...)
	backend := map[string]interface{}{
		"service":       "online",
		"platform":      runtime.GOOS,
		"hugo_path":     hugoPath,
		"launcher_mode": cfgMode(cfg),
	}
	backend["writer_embedded"] = cfgMode(cfg) == "all"
	backend["preview_url"] = previewPublicURL()
	backend["preview_running"] = checkHTTPReady(previewInternalURL(), 900*time.Millisecond)

	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]interface{}{
		"frontend": frontend,
		"backend":  backend,
	}})
}

type controlCommandRequest struct {
	Scope  string `json:"scope"`
	Action string `json:"action"`
}

func HandleControlCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}

	var req controlCommandRequest
	if err := decodeJSONBody(w, r, &req, 16<<10); err != nil {
		return
	}

	req.Scope = strings.ToLower(strings.TrimSpace(req.Scope))
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	if req.Scope == "" || req.Action == "" {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "scope and action are required"})
		return
	}

	cfg := config.GetConfig()
	hugoPath := "."
	if cfg != nil && strings.TrimSpace(cfg.HugoPath) != "" {
		hugoPath = cfg.HugoPath
	}

	result, err := executeControlCommand(cfg, hugoPath, req.Scope, req.Action)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error(), Data: result})
		return
	}

	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: result})
}

func executeControlCommand(cfg *config.Config, hugoPath, scope, action string) (map[string]interface{}, error) {
	switch scope {
	case "frontend":
		switch action {
		case "check":
			hugoCommand := resolveHugoCommand(hugoPath)
			res := runCommand(hugoPath, 25*time.Second, hugoCommand[0], append(hugoCommand[1:], "version")...)
			if !res.Success {
				return map[string]interface{}{"scope": scope, "action": action, "result": res}, fmt.Errorf("hugo unavailable: %s", res.Output)
			}
			return map[string]interface{}{"scope": scope, "action": action, "result": res}, nil
		case "build":
			hugoCommand := resolveHugoCommand(hugoPath)
			res := runCommand(hugoPath, 2*time.Minute, hugoCommand[0], append(hugoCommand[1:], "--minify")...)
			if !res.Success {
				return map[string]interface{}{"scope": scope, "action": action, "result": res}, fmt.Errorf("frontend build failed")
			}
			return map[string]interface{}{"scope": scope, "action": action, "result": res}, nil
		case "preview":
			hugoCommand := resolveHugoCommand(hugoPath)
			previewPort := 1313
			internalURL := config.LocalhostURL(previewPort, "/")
			publicURL := previewPublicURL()
			if checkHTTPReady(internalURL, 900*time.Millisecond) {
				return map[string]interface{}{"scope": scope, "action": action, "result": map[string]interface{}{"preview_url": publicURL, "output": "preview already running"}}, nil
			}
			args := append(hugoCommand[1:], "server", "--bind", "127.0.0.1", "--port", strconv.Itoa(previewPort), "--appendPort=false", "--baseURL", publicURL)
			res := startHugoPreviewCommand(hugoPath, hugoCommand[0], args...)
			if !res.Success {
				return map[string]interface{}{"scope": scope, "action": action, "result": res}, fmt.Errorf("frontend preview failed")
			}
			if !waitForHTTPReady(internalURL, 6*time.Second) {
				return map[string]interface{}{
					"scope":  scope,
					"action": action,
					"result": map[string]interface{}{
						"preview_url": publicURL,
						"warning":     "preview process started; readiness check is still pending",
						"result":      res,
					},
				}, nil
			}
			return map[string]interface{}{"scope": scope, "action": action, "result": map[string]interface{}{"preview_url": publicURL, "result": res}}, nil
		default:
			return nil, fmt.Errorf("unsupported frontend action: %s", action)
		}
	case "backend":
		switch action {
		case "check":
			return map[string]interface{}{
				"scope":  scope,
				"action": action,
				"result": map[string]interface{}{"service": "online", "time": time.Now().Format(time.RFC3339)},
			}, nil
		case "routes":
			return map[string]interface{}{
				"scope":  scope,
				"action": action,
				"result": []string{"/platform/control", "/platform/backend", "/api/posts", "/api/comments", "/api/settings", "/health", "/api/health"},
			}, nil
		case "stop_writer":
			if cfgMode(cfg) == "all" {
				return map[string]interface{}{
					"scope":  scope,
					"action": action,
					"result": map[string]interface{}{
						"message": "writer is embedded in unified mode; nothing to stop separately",
						"mode":    "all",
					},
				}, nil
			}
			writerPort := 9091
			if cfg != nil && cfg.WriterPort > 0 {
				writerPort = cfg.WriterPort
			}
			res := stopListenerOnPort(writerPort)
			if !res.Success {
				return map[string]interface{}{"scope": scope, "action": action, "result": res}, fmt.Errorf("failed to stop writer listener")
			}
			return map[string]interface{}{"scope": scope, "action": action, "result": res}, nil
		case "stop_control":
			if cfgMode(cfg) == "all" {
				return map[string]interface{}{
					"scope":  scope,
					"action": action,
					"result": map[string]interface{}{
						"message": "control and writer share one process in unified mode; close the program window instead",
						"mode":    "all",
					},
				}, nil
			}
			controlPort := 9090
			if cfg != nil && cfg.ControlPort > 0 {
				controlPort = cfg.ControlPort
			}
			result := map[string]interface{}{
				"scope":  scope,
				"action": action,
				"result": map[string]interface{}{
					"message": "control service will stop shortly",
					"port":    controlPort,
				},
			}
			go func() {
				time.Sleep(400 * time.Millisecond)
				os.Exit(0)
			}()
			return result, nil
		default:
			return nil, fmt.Errorf("unsupported backend action: %s", action)
		}
	default:
		return nil, fmt.Errorf("unsupported scope: %s", scope)
	}
}

func stopListenerOnPort(port int) commandResult {
	if port <= 0 {
		return commandResult{Success: false, Output: "invalid port"}
	}
	if runtime.GOOS == "windows" {
		script := "$conns = Get-NetTCPConnection -LocalPort " + strconv.Itoa(port) + " -State Listen -ErrorAction SilentlyContinue; if ($conns) { $pids = $conns | Select-Object -ExpandProperty OwningProcess -Unique; Stop-Process -Id $pids -Force; Write-Output ('Stopped PIDs: ' + ($pids -join ', ')) } else { Write-Output 'No listener found.' }"
		return runCommand(".", 12*time.Second, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	}
	cmd := "pids=$(lsof -t -iTCP:" + strconv.Itoa(port) + " -sTCP:LISTEN 2>/dev/null); if [ -n \"$pids\" ]; then kill -9 $pids; echo \"Stopped PIDs: $pids\"; else echo \"No listener found.\"; fi"
	return runCommand(".", 12*time.Second, "sh", "-c", cmd)
}

func resolveHugoCommand(hugoPath string) []string {
	if hugoPath != "" {
		if runtime.GOOS == "windows" {
			candidate := filepath.Join(hugoPath, "hugo.exe")
			if _, err := os.Stat(candidate); err == nil {
				if abs, absErr := filepath.Abs(candidate); absErr == nil {
					return []string{abs}
				}
				return []string{candidate}
			}
		}
		if info, err := os.Stat(hugoPath); err == nil && !info.IsDir() {
			if abs, absErr := filepath.Abs(hugoPath); absErr == nil {
				return []string{abs}
			}
			return []string{hugoPath}
		}
		if runtime.GOOS == "windows" {
			candidate := filepath.Join(hugoPath, "tools", "hugo", "hugo.exe")
			if _, err := os.Stat(candidate); err == nil {
				if abs, absErr := filepath.Abs(candidate); absErr == nil {
					return []string{abs}
				}
				return []string{candidate}
			}
			candidate = filepath.Join(hugoPath, "tools", "hugo", "hugo")
			if _, err := os.Stat(candidate); err == nil {
				if abs, absErr := filepath.Abs(candidate); absErr == nil {
					return []string{abs}
				}
				return []string{candidate}
			}
		}
	}
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(filepath.Join("tools", "hugo", "hugo.exe")); err == nil {
			candidate := filepath.Join("tools", "hugo", "hugo.exe")
			if abs, absErr := filepath.Abs(candidate); absErr == nil {
				return []string{abs}
			}
			return []string{candidate}
		}
	}
	return []string{"hugo"}
}

func startCommand(dir string, name string, args ...string) commandResult {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return commandResult{Success: false, Output: err.Error()}
	}
	return commandResult{Success: true, Output: fmt.Sprintf("started pid %d", cmd.Process.Pid)}
}

func startHugoPreviewCommand(dir string, name string, args ...string) commandResult {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	cacheDir := filepath.Join(dir, "build", "preview-cache")
	resourceDir := filepath.Join(dir, "build", "preview-resources")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return commandResult{Success: false, Output: err.Error()}
	}
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		return commandResult{Success: false, Output: err.Error()}
	}

	cmd.Env = append(os.Environ(),
		"HUGO_CACHEDIR="+cacheDir,
		"HUGO_RESOURCEDIR="+resourceDir,
	)

	if err := cmd.Start(); err != nil {
		return commandResult{Success: false, Output: err.Error()}
	}
	return commandResult{Success: true, Output: fmt.Sprintf("started pid %d", cmd.Process.Pid)}
}

type commandResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
}

func runCommand(dir string, timeout time.Duration, name string, args ...string) commandResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if output == "" && err != nil {
		output = err.Error()
	}

	if ctx.Err() != nil {
		return commandResult{Success: false, Output: "command timeout"}
	}
	if err != nil {
		return commandResult{Success: false, Output: output}
	}
	return commandResult{Success: true, Output: output}
}

func cfgMode(cfg *config.Config) string {
	if cfg == nil || strings.TrimSpace(cfg.LauncherMode) == "" {
		return "all"
	}
	return strings.TrimSpace(cfg.LauncherMode)
}

func previewInternalURL() string {
	return config.LocalhostURL(1313, "/preview/")
}

func previewPublicURL() string {
	raw := strings.TrimSpace(config.GetEnv("PREVIEW_PUBLIC_URL", ""))
	if raw == "" {
		return previewInternalURL()
	}
	if !strings.HasSuffix(raw, "/") {
		raw += "/"
	}
	return raw
}

func waitForHTTPReady(rawURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if checkHTTPReady(rawURL, 900*time.Millisecond) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

func checkHTTPReady(rawURL string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}
