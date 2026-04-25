package comment

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetCommentsMatchesPathVariants(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMMENTS_DB_PATH", filepath.Join(dir, "comments.db"))
	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() {
		dbMu.Lock()
		if db != nil {
			_ = db.Close()
			db = nil
		}
		dbMu.Unlock()
	}()

	conn, err := getDB()
	if err != nil {
		t.Fatalf("getDB() error = %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = conn.Exec(`
INSERT INTO comments (
	id, post_path, author, phone, email, content, parent_id, approved, ip_address, user_agent,
	fingerprint, captcha_score, vpn_suspected, risk_reasons_json, images_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, 100, 0, '[]', '[]', ?, ?)`,
		"test-comment", "post/high-precision-calculations-in-c/index.md", "tester", "+8613800138000",
		"tester@example.com", "hello", "", "127.0.0.1", "go-test", "fingerprint", now, now)
	if err != nil {
		t.Fatalf("insert comment error = %v", err)
	}

	comments, err := GetComments("content/zh-cn/post/high-precision-calculations-in-c/index.md")
	if err != nil {
		t.Fatalf("GetComments() error = %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("GetComments() len = %d, want 1", len(comments))
	}
	if comments[0].ID != "test-comment" {
		t.Fatalf("GetComments()[0].ID = %q", comments[0].ID)
	}
}

func TestApproveAndDeleteCommentByIDAcrossDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMMENTS_DB_PATH", filepath.Join(dir, "comments.db"))
	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() {
		dbMu.Lock()
		if db != nil {
			_ = db.Close()
			db = nil
		}
		dbMu.Unlock()
	}()

	conn, err := getDB()
	if err != nil {
		t.Fatalf("getDB() error = %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = conn.Exec(`
INSERT INTO comments (
	id, post_path, author, phone, email, content, parent_id, approved, ip_address, user_agent,
	fingerprint, captcha_score, vpn_suspected, risk_reasons_json, images_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, 100, 0, '[]', '[]', ?, ?)`,
		"global-comment", "post/example/index.md", "tester", "+8613800138000",
		"tester@example.com", "hello", "", "127.0.0.1", "go-test", "fingerprint", now, now)
	if err != nil {
		t.Fatalf("insert comment error = %v", err)
	}

	if err := ApproveComment("", "global-comment"); err != nil {
		t.Fatalf("ApproveComment() error = %v", err)
	}
	comments, err := GetAllComments()
	if err != nil {
		t.Fatalf("GetAllComments() error = %v", err)
	}
	if len(comments) != 1 || !comments[0].Approved {
		t.Fatalf("approved comments = %+v", comments)
	}

	if err := DeleteComment("", "global-comment"); err != nil {
		t.Fatalf("DeleteComment() error = %v", err)
	}
	comments, err = GetAllComments()
	if err != nil {
		t.Fatalf("GetAllComments() after delete error = %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments after delete len = %d, want 0", len(comments))
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	dbMu.Lock()
	if db != nil {
		_ = db.Close()
		db = nil
	}
	dbMu.Unlock()
	os.Exit(code)
}
