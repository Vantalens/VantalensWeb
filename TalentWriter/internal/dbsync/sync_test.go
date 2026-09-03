package dbsync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	files    map[string]string
	uploads  map[string]string
	commands []string
	err      error
}

func (r *fakeRunner) Run(ctx context.Context, bin string, args ...string) error {
	if r.err != nil {
		return r.err
	}
	r.commands = append(r.commands, bin+" "+strings.Join(args, " "))
	if len(args) < 2 {
		return nil
	}
	if bin == "ssh" {
		return nil
	}
	srcArg := args[len(args)-2]
	dstArg := args[len(args)-1]
	if isRemoteArg(dstArg) {
		if r.uploads == nil {
			r.uploads = map[string]string{}
		}
		data, err := os.ReadFile(srcArg)
		if err != nil {
			return err
		}
		r.uploads[dstArg] = string(data)
		return nil
	}
	src := r.files[srcArg]
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstArg), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dstArg, data, 0o600)
}

func isRemoteArg(arg string) bool {
	return strings.Contains(arg, ":") && filepath.VolumeName(arg) == ""
}

func TestRunOnceDownloadsToTempAndReplacesLocalDatabase(t *testing.T) {
	dir := t.TempDir()
	remote := filepath.Join(dir, "remote.db")
	local := filepath.Join(dir, "local", "visits.db")
	if err := os.WriteFile(remote, []byte("new-db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("old-db"), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := NewService(Config{
		Enabled:    true,
		RemoteHost: "wj",
		SCPBin:     "scp",
		Interval:   time.Minute,
		Databases: []DatabaseSpec{{
			Name:       "visits",
			RemotePath: "/var/lib/vantalens/analytics/visits.db",
			LocalPath:  local,
		}},
	}, &fakeRunner{files: map[string]string{"wj:/var/lib/vantalens/analytics/visits.db": remote}})

	result := svc.RunOnce(context.Background())
	if len(result.Databases) != 1 || !result.Databases[0].Success {
		t.Fatalf("RunOnce() status = %+v, want one successful database", result)
	}
	data, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-db" {
		t.Fatalf("local database content = %q, want new-db", data)
	}
	if result.Databases[0].SizeBytes != int64(len("new-db")) {
		t.Fatalf("SizeBytes = %d, want %d", result.Databases[0].SizeBytes, len("new-db"))
	}
	entries, err := os.ReadDir(filepath.Dir(local))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Fatalf("temporary file was not cleaned up: %s", entry.Name())
		}
	}
}

func TestRunOnceFailureKeepsExistingDatabaseAndRecordsError(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "comments.db")
	if err := os.WriteFile(local, []byte("old-db"), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := NewService(Config{
		Enabled:    true,
		RemoteHost: "wj",
		SCPBin:     "scp",
		Interval:   time.Minute,
		Databases: []DatabaseSpec{{
			Name:       "comments",
			RemotePath: "/var/lib/vantalens/comments/comments.db",
			LocalPath:  local,
		}},
	}, &fakeRunner{err: os.ErrNotExist})

	result := svc.RunOnce(context.Background())
	if len(result.Databases) != 1 || result.Databases[0].Success {
		t.Fatalf("RunOnce() status = %+v, want failed database", result)
	}
	if result.Databases[0].LastError == "" {
		t.Fatalf("LastError is empty, want sync error")
	}
	data, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old-db" {
		t.Fatalf("local database content = %q, want old-db", data)
	}
}

func TestDefaultConfigSyncsCommentsAndAnalyticsEveryFiveMinutes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DB_SYNC_REMOTE_HOST", "")
	t.Setenv("DB_SYNC_INTERVAL", "")
	t.Setenv("DB_SYNC_SCP_ARGS", "")

	cfg := DefaultConfig(dir)
	if !cfg.Enabled {
		t.Fatalf("Enabled = false, want true")
	}
	if cfg.RemoteHost != "wj" {
		t.Fatalf("RemoteHost = %q, want wj", cfg.RemoteHost)
	}
	if cfg.Interval != 5*time.Minute {
		t.Fatalf("Interval = %s, want 5m", cfg.Interval)
	}
	if got := strings.Join(cfg.SCPArgs, " "); got != "-C" {
		t.Fatalf("SCPArgs = %q, want -C compression", got)
	}
	names := map[string]bool{}
	for _, spec := range cfg.Databases {
		names[spec.Name] = true
		if !strings.HasPrefix(spec.LocalPath, dir) {
			t.Fatalf("%s local path = %q, want under %q", spec.Name, spec.LocalPath, dir)
		}
	}
	for _, name := range []string{"comments", "analytics"} {
		if !names[name] {
			t.Fatalf("missing database spec %q in %+v", name, cfg.Databases)
		}
	}
	if names["articles"] {
		t.Fatalf("articles database should not be synced by default because local Hugo content is the source")
	}
}

type hookTimingRunner struct {
	t      *testing.T
	closed *bool
	source string
}

func (r hookTimingRunner) Run(ctx context.Context, bin string, args ...string) error {
	if *r.closed {
		r.t.Fatalf("BeforeReplace hook ran before download completed")
	}
	dst := args[len(args)-1]
	data, err := os.ReadFile(r.source)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

func TestRunOnceClosesDatabaseOnlyAfterDownloadCompletes(t *testing.T) {
	dir := t.TempDir()
	remote := filepath.Join(dir, "remote.db")
	local := filepath.Join(dir, "local.db")
	if err := os.WriteFile(remote, []byte("new-db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("old-db"), 0o600); err != nil {
		t.Fatal(err)
	}
	closed := false
	svc := NewServiceWithHooks(Config{
		Enabled:    true,
		RemoteHost: "wj",
		SCPBin:     "scp",
		Interval:   time.Minute,
		Databases: []DatabaseSpec{{
			Name:       "comments",
			RemotePath: "/var/lib/vantalens/comments/comments.db",
			LocalPath:  local,
		}},
	}, hookTimingRunner{t: t, closed: &closed, source: remote}, Hooks{
		BeforeReplace: func() error {
			closed = true
			return nil
		},
		AfterReplace: func() error {
			closed = false
			return nil
		},
	})
	result := svc.RunOnce(context.Background())
	if len(result.Databases) != 1 || !result.Databases[0].Success {
		t.Fatalf("RunOnce() status = %+v, want success", result)
	}
	if closed {
		t.Fatalf("database connection stayed closed after sync")
	}
}

func TestDefaultConfigIncludesArticlesOnlyWhenExplicitlyEnabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DB_SYNC_INCLUDE_ARTICLES", "true")
	cfg := DefaultConfig(dir)
	found := false
	for _, spec := range cfg.Databases {
		if spec.Name == "articles" {
			found = true
		}
	}
	if !found {
		t.Fatalf("articles database spec missing when DB_SYNC_INCLUDE_ARTICLES=true")
	}
}

func TestPushDatabaseUploadsTempAndRunsRemoteBackupReplace(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "comments.db")
	if err := os.WriteFile(local, []byte("comment-db"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	svc := NewService(Config{
		Enabled:    true,
		RemoteHost: "wj",
		SCPBin:     "scp",
		SSHBin:     "ssh",
		Interval:   time.Minute,
		Databases: []DatabaseSpec{{
			Name:       "comments",
			RemotePath: "/var/lib/vantalens/comments/comments.db",
			LocalPath:  local,
		}},
	}, runner)

	status := svc.PushDatabase(context.Background(), "comments")
	if !status.Success || status.LastError != "" {
		t.Fatalf("PushDatabase() status = %+v, want success", status)
	}
	foundUpload := false
	for remote, payload := range runner.uploads {
		if strings.HasPrefix(remote, "wj:/var/lib/vantalens/comments/comments.db.tmp-") && payload == "comment-db" {
			foundUpload = true
		}
	}
	if !foundUpload {
		t.Fatalf("temporary upload not recorded: %+v", runner.uploads)
	}
	joined := strings.Join(runner.commands, "\n")
	if !strings.Contains(joined, "cp '/var/lib/vantalens/comments/comments.db'") || !strings.Contains(joined, "mv '/var/lib/vantalens/comments/comments.db.tmp-") {
		t.Fatalf("remote backup/replace command missing from:\n%s", joined)
	}
}

func TestApplyRemoteSQLiteBacksUpAndRunsChangeCheckedStatement(t *testing.T) {
	runner := &fakeRunner{}
	svc := NewService(Config{
		Enabled:    true,
		RemoteHost: "wj",
		SSHBin:     "ssh",
		Interval:   time.Minute,
		Databases: []DatabaseSpec{{
			Name:       "comments",
			RemotePath: "/var/lib/vantalens/comments/comments.db",
			LocalPath:  "local-comments.db",
		}},
	}, runner)

	status := svc.ApplyRemoteSQLite(context.Background(), "comments", "BEGIN IMMEDIATE; UPDATE comments SET approved = 1 WHERE id = 'abc'; SELECT changes(); COMMIT;")
	if !status.Success || status.LastError != "" {
		t.Fatalf("ApplyRemoteSQLite() status = %+v, want success", status)
	}
	joined := strings.Join(runner.commands, "\n")
	if !strings.Contains(joined, "cp '/var/lib/vantalens/comments/comments.db'") {
		t.Fatalf("remote backup command missing from:\n%s", joined)
	}
	if !strings.Contains(joined, "sqlite3 '/var/lib/vantalens/comments/comments.db'") {
		t.Fatalf("sqlite3 command missing from:\n%s", joined)
	}
	if !strings.Contains(joined, "test \"$changes\" != \"0\"") {
		t.Fatalf("change check missing from:\n%s", joined)
	}
}
