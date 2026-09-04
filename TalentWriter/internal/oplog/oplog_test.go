package oplog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "oplog", "operations.jsonl"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return store
}

func TestAppendFillsDefaultsAndPersists(t *testing.T) {
	store := openTestStore(t)

	entry, err := store.Append(Entry{Type: TypePostSave, Target: "content/zh-cn/post/a/index.md", Summary: "保存文章"})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if entry.ID == "" || entry.Ts == "" {
		t.Fatalf("entry missing id/ts: %+v", entry)
	}
	if entry.Origin != OriginLocal || entry.Result != ResultSuccess {
		t.Fatalf("defaults not applied: %+v", entry)
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestAppendRespectsProvidedIDAndSnapshotRoundTrip(t *testing.T) {
	store := openTestStore(t)
	snapshot := "---\ntitle: \"旧标题\"\n---\n旧正文\n"

	if _, err := store.Append(Entry{ID: "fixed-id-1", Type: TypePostDelete, Target: "p", Snapshot: snapshot}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, ok, err := store.Get("fixed-id-1")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if got.Snapshot != snapshot {
		t.Fatalf("snapshot mismatch: %q", got.Snapshot)
	}
	if _, ok, _ := store.Get("missing"); ok {
		t.Fatal("Get returned ok for missing id")
	}
}

func TestListReverseOrderFilterAndLimit(t *testing.T) {
	store := openTestStore(t)
	types := []string{TypePostSave, TypeCommentApprove, TypePostSave, TypePublish}
	for _, typ := range types {
		if _, err := store.Append(Entry{Type: typ, Target: typ}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	all, err := store.List("", 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 4 || all[0].Type != TypePublish {
		t.Fatalf("expected newest-first order, got %+v", all)
	}

	saves, err := store.List(TypePostSave, 0)
	if err != nil || len(saves) != 2 {
		t.Fatalf("filter by type: n=%d err=%v", len(saves), err)
	}

	limited, err := store.List("", 2)
	if err != nil || len(limited) != 2 {
		t.Fatalf("limit: n=%d err=%v", len(limited), err)
	}
}

func TestCorruptLinesAreTolerated(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.Append(Entry{Type: TypePostSave, Target: "ok-1"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	f, err := os.OpenFile(store.Path(), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("this is not json\n{\"id\":\"\"}\n\n")
	_ = f.Close()
	if _, err := store.Append(Entry{Type: TypePostSave, Target: "ok-2"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, err := store.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != 2 || entries[0].Target != "ok-1" || entries[1].Target != "ok-2" {
		t.Fatalf("corrupt lines not skipped: %+v", entries)
	}
}

func TestMarkSyncedRewritesAtomically(t *testing.T) {
	store := openTestStore(t)
	first, _ := store.Append(Entry{Type: TypePostSave, Target: "a"})
	second, _ := store.Append(Entry{Type: TypeCommentDelete, Target: "c1", Synced: true})

	ok, err := store.MarkSynced(first.ID)
	if err != nil || !ok {
		t.Fatalf("MarkSynced: ok=%v err=%v", ok, err)
	}
	got, _, _ := store.Get(first.ID)
	if !got.Synced {
		t.Fatal("entry not marked synced")
	}

	// already-synced entry is a no-op but still found
	ok, err = store.MarkSynced(second.ID)
	if err != nil || !ok {
		t.Fatalf("MarkSynced existing synced: ok=%v err=%v", ok, err)
	}
	if ok, _ := store.MarkSynced("nope"); ok {
		t.Fatal("MarkSynced reported found for missing id")
	}

	total, pending, err := store.Stats()
	if err != nil || total != 2 || pending != 0 {
		t.Fatalf("Stats: total=%d pending=%d err=%v", total, pending, err)
	}
}

func TestConcurrentAppend(t *testing.T) {
	store := openTestStore(t)
	const workers = 8
	const perWorker = 25
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				if _, err := store.Append(Entry{Type: TypePostSave, Target: "w"}); err != nil {
					t.Errorf("Append: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	entries, err := store.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != workers*perWorker {
		t.Fatalf("entries = %d, want %d", len(entries), workers*perWorker)
	}
	ids := map[string]bool{}
	for _, e := range entries {
		if ids[e.ID] {
			t.Fatalf("duplicate id %s", e.ID)
		}
		ids[e.ID] = true
	}
}

func TestEmptyLogReadsAsEmpty(t *testing.T) {
	store := openTestStore(t)
	entries, err := store.List("", 0)
	if err != nil || len(entries) != 0 {
		t.Fatalf("List on missing file: n=%d err=%v", len(entries), err)
	}
	total, pending, err := store.Stats()
	if err != nil || total != 0 || pending != 0 {
		t.Fatalf("Stats on missing file: %d/%d err=%v", total, pending, err)
	}
}

func TestAppendRequiresType(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.Append(Entry{}); err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("expected type error, got %v", err)
	}
}
