package dbsync

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vantalens/talentwriter/internal/config"
)

type Runner interface {
	Run(ctx context.Context, bin string, args ...string) error
}

type CommandRunner struct{}

func (CommandRunner) Run(ctx context.Context, bin string, args ...string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

type DatabaseSpec struct {
	Name       string
	RemotePath string
	LocalPath  string
}

type Config struct {
	Enabled    bool
	RemoteHost string
	SCPBin     string
	SCPArgs    []string
	SSHBin     string
	Interval   time.Duration
	Databases  []DatabaseSpec
	Timeout    time.Duration
}

type Hooks struct {
	BeforeReplace func() error
	AfterReplace  func() error
}

type DatabaseStatus struct {
	Name         string `json:"name"`
	RemotePath   string `json:"remote_path"`
	LocalPath    string `json:"local_path"`
	LastSyncAt   string `json:"last_sync_at,omitempty"`
	LastAttempt  string `json:"last_attempt,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	Success      bool   `json:"success"`
	LastError    string `json:"last_error,omitempty"`
	LastDuration string `json:"last_duration,omitempty"`
}

type Status struct {
	Enabled   bool             `json:"enabled"`
	Host      string           `json:"host"`
	Interval  string           `json:"interval"`
	Running   bool             `json:"running"`
	Databases []DatabaseStatus `json:"databases"`
}

type Service struct {
	cfg    Config
	runner Runner
	hooks  Hooks

	mu       sync.Mutex
	running  bool
	statuses map[string]DatabaseStatus
}

func DefaultConfig(hugoPath string) Config {
	remoteBase := strings.TrimRight(config.GetEnv("DB_SYNC_REMOTE_BASE", "/var/lib/vantalens"), "/")
	enabled := !strings.EqualFold(strings.TrimSpace(config.GetEnv("DB_SYNC_ENABLED", "true")), "false")
	cfg := Config{
		Enabled:    enabled,
		RemoteHost: config.GetEnv("DB_SYNC_REMOTE_HOST", "wj"),
		SCPBin:     config.GetEnv("DB_SYNC_SCP_BIN", "scp"),
		SCPArgs:    parseArgsEnv("DB_SYNC_SCP_ARGS", "-C"),
		SSHBin:     config.GetEnv("DB_SYNC_SSH_BIN", "ssh"),
		Interval:   parseDurationEnv("DB_SYNC_INTERVAL", 5*time.Minute),
		Timeout:    parseDurationEnv("DB_SYNC_TIMEOUT", 30*time.Second),
		Databases: []DatabaseSpec{
			{
				Name:       "comments",
				RemotePath: remoteBase + "/comments/comments.db",
				LocalPath:  config.GetCommentsDBPath(hugoPath),
			},
			{
				Name:       "analytics",
				RemotePath: remoteBase + "/analytics/visits.db",
				LocalPath:  config.GetAnalyticsDBPath(hugoPath),
			},
		},
	}
	if strings.EqualFold(strings.TrimSpace(config.GetEnv("DB_SYNC_INCLUDE_ARTICLES", "false")), "true") {
		cfg.Databases = append(cfg.Databases, DatabaseSpec{
			Name:       "articles",
			RemotePath: remoteBase + "/articles/articles.db",
			LocalPath:  config.GetArticlesDBPath(hugoPath),
		})
	}
	return cfg
}

func NewService(cfg Config, runner Runner) *Service {
	return NewServiceWithHooks(cfg, runner, Hooks{})
}

func NewServiceWithHooks(cfg Config, runner Runner, hooks Hooks) *Service {
	if runner == nil {
		runner = CommandRunner{}
	}
	if cfg.SCPBin == "" {
		cfg.SCPBin = "scp"
	}
	if cfg.SCPArgs == nil {
		cfg.SCPArgs = parseArgsEnv("DB_SYNC_SCP_ARGS", "-C")
	}
	if cfg.SSHBin == "" {
		cfg.SSHBin = "ssh"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	svc := &Service{
		cfg:      cfg,
		runner:   runner,
		hooks:    hooks,
		statuses: map[string]DatabaseStatus{},
	}
	for _, spec := range cfg.Databases {
		svc.statuses[spec.Name] = DatabaseStatus{
			Name:       spec.Name,
			RemotePath: spec.RemotePath,
			LocalPath:  spec.LocalPath,
		}
	}
	return svc
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || !s.cfg.Enabled || s.cfg.Interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(s.cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result := s.RunOnce(ctx)
				for _, db := range result.Databases {
					if db.LastError != "" {
						log.Printf("[DBSYNC] %s sync failed: %s", db.Name, db.LastError)
					}
				}
			}
		}
	}()
}

func (s *Service) RunOnce(ctx context.Context) Status {
	if s == nil {
		return Status{}
	}
	if !s.cfg.Enabled {
		return s.Status()
	}

	s.mu.Lock()
	if s.running {
		result := s.snapshotLocked()
		s.mu.Unlock()
		return result
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	replaceSucceeded := false
	for _, spec := range s.cfg.Databases {
		if s.syncDatabase(ctx, spec) {
			replaceSucceeded = true
		}
	}
	if replaceSucceeded {
		log.Printf("[DBSYNC] database sync completed")
	}
	return s.Status()
}

func (s *Service) RunDatabase(ctx context.Context, name string) Status {
	if s != nil {
		for {
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				break
			}
			select {
			case <-ctx.Done():
				return s.Status()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	return s.runDatabases(ctx, map[string]bool{strings.TrimSpace(name): true})
}

func (s *Service) runDatabases(ctx context.Context, names map[string]bool) Status {
	if s == nil {
		return Status{}
	}
	if !s.cfg.Enabled {
		return s.Status()
	}
	s.mu.Lock()
	if s.running {
		result := s.snapshotLocked()
		s.mu.Unlock()
		return result
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	replaceSucceeded := false
	found := false
	for _, spec := range s.cfg.Databases {
		if !names[spec.Name] {
			continue
		}
		found = true
		if s.syncDatabase(ctx, spec) {
			replaceSucceeded = true
		}
	}
	if !found {
		for name := range names {
			s.updateStatus(DatabaseStatus{
				Name:        name,
				LastAttempt: time.Now().UTC().Format(time.RFC3339),
				LastError:   "database sync spec not configured",
			})
		}
	}
	if replaceSucceeded {
		log.Printf("[DBSYNC] selected database sync completed")
	}
	return s.Status()
}

func (s *Service) Status() Status {
	if s == nil {
		return Status{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *Service) PushDatabase(ctx context.Context, name string) DatabaseStatus {
	start := time.Now()
	now := start.UTC().Format(time.RFC3339)
	if s == nil {
		return DatabaseStatus{Name: strings.TrimSpace(name), LastAttempt: now, LastError: "database sync service is not configured"}
	}
	spec, ok := s.FindDatabase(name)
	if !ok {
		status := DatabaseStatus{Name: strings.TrimSpace(name), LastAttempt: now, LastError: "database sync spec not configured"}
		s.updateStatus(status)
		return status
	}
	status := DatabaseStatus{
		Name:        spec.Name,
		RemotePath:  spec.RemotePath,
		LocalPath:   spec.LocalPath,
		LastAttempt: now,
	}
	if err := validateSpec(s.cfg, spec); err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}
	info, err := os.Stat(spec.LocalPath)
	if err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}
	if info.Size() == 0 {
		status.LastError = "local database is empty"
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}

	remoteTmp := spec.RemotePath + ".tmp-" + strconvTime(start)
	remoteBackup := spec.RemotePath + ".backup-" + time.Now().UTC().Format("20060102T150405Z")
	runCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	uploadArgs := append([]string{}, s.cfg.SCPArgs...)
	uploadArgs = append(uploadArgs, spec.LocalPath, s.cfg.RemoteHost+":"+remoteTmp)
	if err := s.runner.Run(runCtx, s.cfg.SCPBin, uploadArgs...); err != nil {
		status.LastError = "upload temp database: " + err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}
	cmd := "set -e; cp " + shellQuote(spec.RemotePath) + " " + shellQuote(remoteBackup) + "; mv " + shellQuote(remoteTmp) + " " + shellQuote(spec.RemotePath) + "; chmod 600 " + shellQuote(spec.RemotePath)
	if err := s.runner.Run(runCtx, s.cfg.SSHBin, s.cfg.RemoteHost, cmd); err != nil {
		status.LastError = "replace remote database: " + err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}
	status.Success = true
	status.LastSyncAt = time.Now().UTC().Format(time.RFC3339)
	status.SizeBytes = info.Size()
	status.LastDuration = time.Since(start).String()
	s.updateStatus(status)
	return status
}

func (s *Service) ApplyRemoteSQLite(ctx context.Context, name, sql string) DatabaseStatus {
	start := time.Now()
	now := start.UTC().Format(time.RFC3339)
	if s == nil {
		return DatabaseStatus{Name: strings.TrimSpace(name), LastAttempt: now, LastError: "database sync service is not configured"}
	}
	spec, ok := s.FindDatabase(name)
	if !ok {
		status := DatabaseStatus{Name: strings.TrimSpace(name), LastAttempt: now, LastError: "database sync spec not configured"}
		s.updateStatus(status)
		return status
	}
	status := DatabaseStatus{
		Name:        spec.Name,
		RemotePath:  spec.RemotePath,
		LocalPath:   spec.LocalPath,
		LastAttempt: now,
	}
	if err := validateSpec(s.cfg, spec); err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}
	sql = strings.TrimSpace(sql)
	if sql == "" {
		status.LastError = "remote sqlite statement is required"
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}
	backup := spec.RemotePath + ".backup-" + time.Now().UTC().Format("20060102T150405Z")
	remoteSQL := spec.RemotePath + ".sql-" + strconvTime(start)
	encodedSQL := base64.StdEncoding.EncodeToString([]byte(sql))
	cmd := "set -e; sql_file=" + shellQuote(remoteSQL) +
		"; trap 'rm -f \"$sql_file\"' EXIT" +
		"; printf %s " + shellQuote(encodedSQL) + " | base64 -d > \"$sql_file\"" +
		"; cp " + shellQuote(spec.RemotePath) + " " + shellQuote(backup) +
		"; changes=$(sqlite3 " + shellQuote(spec.RemotePath) + " < \"$sql_file\")" +
		"; test \"$changes\" != \"0\""
	runCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	if err := s.runner.Run(runCtx, s.cfg.SSHBin, s.cfg.RemoteHost, cmd); err != nil {
		status.LastError = "remote sqlite update: " + err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return status
	}
	status.Success = true
	status.LastSyncAt = time.Now().UTC().Format(time.RFC3339)
	status.LastDuration = time.Since(start).String()
	s.updateStatus(status)
	return status
}

func (s *Service) FindDatabase(name string) (DatabaseSpec, bool) {
	if s == nil {
		return DatabaseSpec{}, false
	}
	name = strings.TrimSpace(name)
	for _, spec := range s.cfg.Databases {
		if spec.Name == name {
			return spec, true
		}
	}
	return DatabaseSpec{}, false
}

func (s *Service) syncDatabase(ctx context.Context, spec DatabaseSpec) bool {
	start := time.Now()
	now := start.UTC().Format(time.RFC3339)
	status := DatabaseStatus{
		Name:        spec.Name,
		RemotePath:  spec.RemotePath,
		LocalPath:   spec.LocalPath,
		LastAttempt: now,
	}

	if err := validateSpec(s.cfg, spec); err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return false
	}

	if err := os.MkdirAll(filepath.Dir(spec.LocalPath), 0o755); err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return false
	}

	tmpPath := spec.LocalPath + "-" + strconvTime(start) + ".tmp"
	defer os.Remove(tmpPath)

	remote := s.cfg.RemoteHost + ":" + spec.RemotePath
	runCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	downloadArgs := append([]string{}, s.cfg.SCPArgs...)
	downloadArgs = append(downloadArgs, remote, tmpPath)
	if err := s.runner.Run(runCtx, s.cfg.SCPBin, downloadArgs...); err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return false
	}
	info, err := os.Stat(tmpPath)
	if err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return false
	}
	if info.Size() == 0 {
		status.LastError = "downloaded database is empty"
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		return false
	}
	if s.hooks.BeforeReplace != nil {
		if err := s.hooks.BeforeReplace(); err != nil {
			status.LastError = "before replace: " + err.Error()
			status.LastDuration = time.Since(start).String()
			s.updateStatus(status)
			if s.hooks.AfterReplace != nil {
				if afterErr := s.hooks.AfterReplace(); afterErr != nil {
					s.recordHookError("after_replace", afterErr)
				}
			}
			return false
		}
	}
	if err := replaceFile(tmpPath, spec.LocalPath); err != nil {
		status.LastError = err.Error()
		status.LastDuration = time.Since(start).String()
		s.updateStatus(status)
		if s.hooks.AfterReplace != nil {
			if afterErr := s.hooks.AfterReplace(); afterErr != nil {
				s.recordHookError("after_replace", afterErr)
			}
		}
		return false
	}
	if s.hooks.AfterReplace != nil {
		if err := s.hooks.AfterReplace(); err != nil {
			status.LastError = "after replace: " + err.Error()
			status.LastDuration = time.Since(start).String()
			s.updateStatus(status)
			return false
		}
	}

	status.Success = true
	status.LastSyncAt = time.Now().UTC().Format(time.RFC3339)
	status.SizeBytes = info.Size()
	status.LastDuration = time.Since(start).String()
	s.updateStatus(status)
	return true
}

func (s *Service) updateStatus(status DatabaseStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[status.Name] = status
}

func (s *Service) recordHookError(name string, err error) {
	if err == nil {
		return
	}
	s.updateStatus(DatabaseStatus{
		Name:        name,
		LastAttempt: time.Now().UTC().Format(time.RFC3339),
		LastError:   err.Error(),
	})
}

func (s *Service) snapshotLocked() Status {
	result := Status{
		Enabled:  s.cfg.Enabled,
		Host:     s.cfg.RemoteHost,
		Interval: s.cfg.Interval.String(),
		Running:  s.running,
	}
	for _, spec := range s.cfg.Databases {
		status, ok := s.statuses[spec.Name]
		if !ok {
			status = DatabaseStatus{Name: spec.Name, RemotePath: spec.RemotePath, LocalPath: spec.LocalPath}
		}
		result.Databases = append(result.Databases, status)
	}
	for name, status := range s.statuses {
		found := false
		for _, spec := range s.cfg.Databases {
			if spec.Name == name {
				found = true
				break
			}
		}
		if !found {
			result.Databases = append(result.Databases, status)
		}
	}
	return result
}

func validateSpec(cfg Config, spec DatabaseSpec) error {
	if strings.TrimSpace(cfg.RemoteHost) == "" {
		return errors.New("remote host is required")
	}
	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("database name is required")
	}
	if strings.TrimSpace(spec.RemotePath) == "" {
		return errors.New("remote path is required")
	}
	if strings.TrimSpace(spec.LocalPath) == "" {
		return errors.New("local path is required")
	}
	return nil
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	duration, err := time.ParseDuration(raw)
	if err == nil && duration > 0 {
		return duration
	}
	return fallback
}

func parseArgsEnv(key, fallback string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		raw = strings.TrimSpace(fallback)
	}
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}

func replaceFile(src, dst string) error {
	backup := dst + "-" + strconvTime(time.Now())
	hasExisting := false
	if _, err := os.Stat(dst); err == nil {
		hasExisting = true
		if err := os.Rename(dst, backup); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(src, dst); err != nil {
		if hasExisting {
			_ = os.Rename(backup, dst)
		}
		return err
	}
	if hasExisting {
		_ = os.Remove(backup)
	}
	return nil
}

func strconvTime(t time.Time) string {
	return fmt.Sprintf("%d", t.UnixNano())
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
