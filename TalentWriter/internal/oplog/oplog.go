// Package oplog implements an append-only JSONL operation log for the
// TalentWriter platform. Every mutating admin operation (articles, comments,
// settings, publish) is recorded with an optional pre-change snapshot so the
// operation can be reviewed, rolled back, and reconciled between the local
// node and the remote authority backend.
package oplog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// HeaderOplogID propagates the logical operation id to the remote authority
// backend so both sides record the same entry id and id-based diffing works.
const HeaderOplogID = "X-Oplog-Id"

const (
	OriginLocal  = "local"
	OriginRemote = "remote"

	ResultSuccess = "success"

	TypePostSave       = "post.save"
	TypePostCreate     = "post.create"
	TypePostDelete     = "post.delete"
	TypePostRestore    = "post.restore"
	TypePostPurge      = "post.purge"
	TypeCommentApprove = "comment.approve"
	TypeCommentDelete  = "comment.delete"
	TypeSettingsSave   = "settings.save"
	TypePublish        = "publish"
	RollbackSuffix     = ".rollback"
)

// Entry is a single operation record.
type Entry struct {
	ID       string `json:"id"`
	Ts       string `json:"ts"`
	Origin   string `json:"origin"`
	Actor    string `json:"actor"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Summary  string `json:"summary"`
	Snapshot string `json:"snapshot,omitempty"`
	Result   string `json:"result"`
	Synced   bool   `json:"synced"`
}

// FailedResult builds the result string for a failed operation.
func FailedResult(err error) string {
	if err == nil {
		return ResultSuccess
	}
	return "failed: " + err.Error()
}

// Success reports whether the entry records a successful operation.
func (e Entry) Success() bool {
	return e.Result == ResultSuccess
}

// IsRollback reports whether the entry itself is a rollback record.
func (e Entry) IsRollback() bool {
	return strings.HasSuffix(e.Type, RollbackSuffix)
}

// NewID returns a fresh operation id.
func NewID() string {
	return uuid.NewString()
}

// Store is a mutex-guarded append-only JSONL log file. A single process
// writes to it, so an in-process mutex provides the required concurrency
// safety.
type Store struct {
	mu   sync.Mutex
	path string
}

// Open creates (if needed) the parent directory and returns a store bound to
// path. The file itself is created lazily on the first append.
func Open(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("oplog path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &Store{path: path}, nil
}

// Path returns the backing file path.
func (s *Store) Path() string {
	return s.path
}

// Append validates the entry, fills ID/Ts when empty, and appends one JSONL
// record. It returns the stored entry.
func (s *Store) Append(e Entry) (Entry, error) {
	if strings.TrimSpace(e.Type) == "" {
		return Entry{}, fmt.Errorf("oplog entry type is required")
	}
	if strings.TrimSpace(e.ID) == "" {
		e.ID = NewID()
	}
	if strings.TrimSpace(e.Ts) == "" {
		e.Ts = time.Now().UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(e.Origin) == "" {
		e.Origin = OriginLocal
	}
	if strings.TrimSpace(e.Result) == "" {
		e.Result = ResultSuccess
	}
	line, err := json.Marshal(e)
	if err != nil {
		return Entry{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return Entry{}, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// All returns every parseable entry in file (ascending) order. Corrupt lines
// are skipped so one bad record never blocks the log.
func (s *Store) All() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readAllLocked()
}

// List returns entries in reverse (newest first) order, optionally filtered
// by exact type and capped at limit (limit <= 0 means no cap).
func (s *Store) List(opType string, limit int) ([]Entry, error) {
	entries, err := s.All()
	if err != nil {
		return nil, err
	}
	opType = strings.TrimSpace(opType)
	out := make([]Entry, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		if opType != "" && entries[i].Type != opType {
			continue
		}
		out = append(out, entries[i])
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Get finds one entry by id.
func (s *Store) Get(id string) (Entry, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Entry{}, false, fmt.Errorf("oplog entry id is required")
	}
	entries, err := s.All()
	if err != nil {
		return Entry{}, false, err
	}
	for _, e := range entries {
		if e.ID == id {
			return e, true, nil
		}
	}
	return Entry{}, false, nil
}

// Has reports whether an entry with the given id exists.
func (s *Store) Has(id string) (bool, error) {
	_, ok, err := s.Get(id)
	return ok, err
}

// MarkSynced flips the synced flag of one entry. The file is rewritten
// atomically (temp file + rename); the log stays otherwise append-only.
func (s *Store) MarkSynced(id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, fmt.Errorf("oplog entry id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.readAllLocked()
	if err != nil {
		return false, err
	}
	found := false
	for i := range entries {
		if entries[i].ID == id {
			if entries[i].Synced {
				return true, nil
			}
			entries[i].Synced = true
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}
	return true, s.rewriteLocked(entries)
}

// Stats returns the total entry count and how many are not yet synced.
func (s *Store) Stats() (total int, pending int, err error) {
	entries, err := s.All()
	if err != nil {
		return 0, 0, err
	}
	for _, e := range entries {
		if !e.Synced {
			pending++
		}
	}
	return len(entries), pending, nil
}

func (s *Store) readAllLocked() ([]Entry, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entries := []Entry{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 8<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil || e.ID == "" {
			// tolerate corrupt lines
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) rewriteLocked(entries []Entry) error {
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return err
		}
		if _, err := writer.Write(append(line, '\n')); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, s.path)
}
