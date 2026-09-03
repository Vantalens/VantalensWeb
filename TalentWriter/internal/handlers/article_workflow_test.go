package handlers

import (
	"bytes"
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
)

func TestArticleWorkflowInIsolatedSite(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content", "zh-cn", "post"), 0o755); err != nil {
		t.Fatal(err)
	}

	oldConfig := config.GetConfig()
	defer config.SetConfig(oldConfig)
	config.SetConfig(&config.Config{HugoPath: root, AdminUsername: "audit", AdminToken: "secret"})
	t.Setenv("ARTICLES_DB_PATH", filepath.Join(root, "data", "articles.db"))
	t.Setenv("ARTICLE_TRASH_PATH", filepath.Join(root, "data", "trash"))
	t.Setenv("JWT_SECRET", strings.Repeat("a", 64))
	if err := article.Init(root); err != nil {
		t.Fatal(err)
	}
	defer article.Close()
	if err := auth.InitJWTSecret(); err != nil {
		t.Fatal(err)
	}
	token, err := auth.CreateJWT("audit", "access")
	if err != nil {
		t.Fatal(err)
	}

	first, err := createArticle("Duplicate title", "audit", "initial body", true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := createArticle("Duplicate title", "audit", "second body", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{first, second} {
		if !strings.HasPrefix(filepath.ToSlash(path), "content/zh-cn/post/") {
			t.Fatalf("article path escaped canonical content tree: %q", path)
		}
	}
	if first == second {
		t.Fatal("duplicate titles must receive distinct paths")
	}

	original, err := readArticle(first)
	if err != nil {
		t.Fatal(err)
	}
	original = strings.Replace(original, "draft: true", "draft: true\ncustom_field: keep-me", 1)
	if err := writeArticle(first, original); err != nil {
		t.Fatal(err)
	}
	doc, err := parseArticleDocument(original)
	if err != nil {
		t.Fatal(err)
	}

	if err := writeArticle(first, strings.Replace(original, "initial body", "external edit", 1)); err != nil {
		t.Fatal(err)
	}
	conflictBody := map[string]any{
		"path": first, "body": "stale edit", "metadata": doc.Metadata, "revision": doc.Revision,
	}
	if status := callArticleHandler(t, HandleSaveContent, http.MethodPost, "/api/save_content", conflictBody, token).Code; status != http.StatusConflict {
		t.Fatalf("stale save status = %d, want %d", status, http.StatusConflict)
	}

	current, err := readArticle(first)
	if err != nil {
		t.Fatal(err)
	}
	currentDoc, err := parseArticleDocument(current)
	if err != nil {
		t.Fatal(err)
	}
	currentDoc.Metadata.Draft = false
	saveBody := map[string]any{
		"path": first, "body": "published body", "metadata": currentDoc.Metadata, "revision": currentDoc.Revision,
	}
	if status := callArticleHandler(t, HandleSaveContent, http.MethodPost, "/api/save_content", saveBody, token).Code; status != http.StatusOK {
		t.Fatalf("save status = %d, want %d", status, http.StatusOK)
	}
	saved, err := readArticle(first)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saved, "custom_field: keep-me") || !strings.Contains(saved, "draft: false") || !strings.Contains(saved, "published body") {
		t.Fatalf("structured save did not preserve unknown front matter or update draft/body:\n%s", saved)
	}

	record, err := trashArticle(first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(first))); !os.IsNotExist(err) {
		t.Fatalf("trashed article still exists, stat error = %v", err)
	}
	if _, err := restoreTrashedArticle(record.ID); err != nil {
		t.Fatal(err)
	}
	if restored, err := readArticle(first); err != nil || !strings.Contains(restored, "published body") {
		t.Fatalf("restored article invalid: err=%v content=%q", err, restored)
	}

	record, err = trashArticle(first)
	if err != nil {
		t.Fatal(err)
	}
	if err := purgeTrashedArticle(record.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := readTrashRecord(record.ID); err == nil {
		t.Fatal("purged article is still present in trash")
	}
}

func callArticleHandler(t *testing.T, handler http.HandlerFunc, method, target string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}
