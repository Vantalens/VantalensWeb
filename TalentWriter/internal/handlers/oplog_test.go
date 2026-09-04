package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vantalens/talentwriter/internal/article"
	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
	"vantalens/talentwriter/internal/oplog"
)

func setupOplogTestEnv(t *testing.T) (*oplog.Store, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	config.SetConfig(&config.Config{HugoPath: root, AdminUsername: "vantalens", AdminToken: "secret"})
	if err := auth.InitJWTSecret(); err != nil {
		t.Fatal(err)
	}
	if err := article.Init(root); err != nil {
		t.Fatal(err)
	}
	store, err := oplog.Open(filepath.Join(t.TempDir(), "oplog", "operations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	SetOpLogStore(store)
	t.Cleanup(func() {
		SetOpLogStore(nil)
		_ = article.Close()
		config.SetConfig(nil)
	})
	return store, root
}

func oplogTestToken(t *testing.T) string {
	t.Helper()
	token, err := auth.CreateJWT("vantalens", "access")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestOplogAPIsRequireAuth(t *testing.T) {
	setupOplogTestEnv(t)
	for _, tc := range []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodGet, "/api/oplog/list", HandleOplogList},
		{http.MethodGet, "/api/oplog/entry?id=x", HandleOplogEntry},
		{http.MethodGet, "/api/oplog/compare", HandleOplogCompare},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		tc.handler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s status = %d, want %d", tc.path, rr.Code, http.StatusUnauthorized)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/oplog/rollback", strings.NewReader(`{"id":"x"}`))
	rr := httptest.NewRecorder()
	HandleOplogRollback(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("rollback status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/oplog/sync", strings.NewReader(`{"direction":"push"}`))
	rr = httptest.NewRecorder()
	HandleOplogSync(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sync status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleOplogListAndEntry(t *testing.T) {
	store, _ := setupOplogTestEnv(t)
	token := oplogTestToken(t)
	if _, err := store.Append(oplog.Entry{Type: oplog.TypePostSave, Target: "content/zh-cn/post/a/index.md", Summary: "保存文章", Snapshot: "old content"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(oplog.Entry{Type: oplog.TypeCommentApprove, Target: "c1", Summary: "审核评论"}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/oplog/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	HandleOplogList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(resp.Data)
	var list oplogListData
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Entries) != 2 || list.Total != 2 || list.Pending != 2 {
		t.Fatalf("list data = %+v", list)
	}
	if list.Entries[0].Type != oplog.TypeCommentApprove {
		t.Fatalf("expected newest first, got %+v", list.Entries[0])
	}

	// type filter
	req = httptest.NewRequest(http.MethodGet, "/api/oplog/list?type=post.save", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	HandleOplogList(rr, req)
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	raw, _ = json.Marshal(resp.Data)
	list = oplogListData{}
	_ = json.Unmarshal(raw, &list)
	if len(list.Entries) != 1 || list.Entries[0].Type != oplog.TypePostSave {
		t.Fatalf("filtered list = %+v", list.Entries)
	}

	// entry detail with snapshot
	req = httptest.NewRequest(http.MethodGet, "/api/oplog/entry?id="+list.Entries[0].ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	HandleOplogEntry(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("entry status = %d, body=%s", rr.Code, rr.Body.String())
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	raw, _ = json.Marshal(resp.Data)
	var entry oplog.Entry
	_ = json.Unmarshal(raw, &entry)
	if entry.Snapshot != "old content" {
		t.Fatalf("entry snapshot = %q", entry.Snapshot)
	}

	// missing entry
	req = httptest.NewRequest(http.MethodGet, "/api/oplog/entry?id=nope", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	HandleOplogEntry(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing entry status = %d", rr.Code)
	}
}

func TestHandleSaveContentRecordsOplog(t *testing.T) {
	store, root := setupOplogTestEnv(t)
	token := oplogTestToken(t)

	rel := "content/zh-cn/post/demo/index.md"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	old := "---\ntitle: \"旧标题\"\ndraft: true\n---\n旧正文\n"
	if err := os.WriteFile(abs, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"path":"` + rel + `","content":"---\ntitle: \"新标题\"\n---\n新正文\n"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/save_content", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleSaveContent(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("save status = %d, body=%s", rr.Code, rr.Body.String())
	}

	entries, err := store.List(oplog.TypePostSave, 0)
	if err != nil || len(entries) != 1 {
		t.Fatalf("oplog entries = %d, err=%v", len(entries), err)
	}
	entry := entries[0]
	if entry.Target != rel || entry.Result != oplog.ResultSuccess || entry.Snapshot != old {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.Synced {
		t.Fatal("local save should not be marked synced")
	}

	// failed save (missing file) also gets logged
	req = httptest.NewRequest(http.MethodPost, "/api/save_content", strings.NewReader(`{"path":"content/zh-cn/post/ghost/index.md","content":"x"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	HandleSaveContent(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatal("expected save failure for missing file")
	}
	entries, _ = store.List(oplog.TypePostSave, 0)
	if len(entries) != 2 || entries[0].Result == oplog.ResultSuccess {
		t.Fatalf("failed save not logged: %+v", entries)
	}
}

func TestHandleOplogRollbackRestoresSnapshot(t *testing.T) {
	store, root := setupOplogTestEnv(t)
	token := oplogTestToken(t)

	rel := "content/zh-cn/post/demo/index.md"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	old := "---\ntitle: \"回溯前\"\n---\n旧内容\n"
	if err := os.WriteFile(abs, []byte("---\ntitle: \"当前\"\n---\n当前内容\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	logged, err := store.Append(oplog.Entry{Type: oplog.TypePostSave, Target: rel, Summary: "保存文章", Snapshot: old})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/oplog/rollback", strings.NewReader(`{"id":"`+logged.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleOplogRollback(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("rollback status = %d, body=%s", rr.Code, rr.Body.String())
	}
	restored, err := os.ReadFile(abs)
	if err != nil || string(restored) != old {
		t.Fatalf("restored content = %q, err=%v", string(restored), err)
	}

	// rollback itself is logged
	entries, _ := store.List(oplog.TypePostSave+oplog.RollbackSuffix, 0)
	if len(entries) != 1 {
		t.Fatalf("rollback entry not logged: %+v", entries)
	}

	// rollback entry cannot be rolled back again
	req = httptest.NewRequest(http.MethodPost, "/api/oplog/rollback", strings.NewReader(`{"id":"`+entries[0].ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	HandleOplogRollback(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("re-rollback status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleOplogRollbackRejectsIrreversible(t *testing.T) {
	store, _ := setupOplogTestEnv(t)
	token := oplogTestToken(t)

	publish, _ := store.Append(oplog.Entry{Type: oplog.TypePublish, Target: "public", Summary: "发布"})
	req := httptest.NewRequest(http.MethodPost, "/api/oplog/rollback", strings.NewReader(`{"id":"`+publish.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleOplogRollback(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("publish rollback status = %d, want %d, body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	// comment.delete without snapshot is irreversible
	del, _ := store.Append(oplog.Entry{Type: oplog.TypeCommentDelete, Target: "c1", Summary: "删除评论"})
	req = httptest.NewRequest(http.MethodPost, "/api/oplog/rollback", strings.NewReader(`{"id":"`+del.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	HandleOplogRollback(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("snapshot-less comment delete rollback status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleOplogCompareRequiresRemote(t *testing.T) {
	setupOplogTestEnv(t)
	token := oplogTestToken(t)
	// REMOTE_ADMIN_BASE not configured -> compare/sync should fail cleanly
	req := httptest.NewRequest(http.MethodGet, "/api/oplog/compare", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	HandleOplogCompare(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("compare status = %d, want %d, body=%s", rr.Code, http.StatusBadGateway, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/oplog/sync", strings.NewReader(`{"direction":"push"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	HandleOplogSync(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("sync status = %d, want %d", rr.Code, http.StatusBadGateway)
	}

	// invalid direction
	req = httptest.NewRequest(http.MethodPost, "/api/oplog/sync", strings.NewReader(`{"direction":"sideways"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	HandleOplogSync(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad direction status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestRecordOpNeverPanicsWithoutStore(t *testing.T) {
	SetOpLogStore(nil)
	t.Cleanup(func() { SetOpLogStore(nil) })
	config.SetConfig(&config.Config{HugoPath: filepath.Join(t.TempDir(), "nonexistent-root")})
	t.Cleanup(func() { config.SetConfig(nil) })
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	// store open may succeed or fail depending on path resolution; either way no panic
	recordOp(req, oplog.TypePublish, "public", "发布", "", nil, false)
}
