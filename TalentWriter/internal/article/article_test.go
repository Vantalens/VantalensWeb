package article

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vantalens/talentwriter/internal/models"
)

func TestNormalizePath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"dot", ".", ""},
		{"dot with spaces", " . ", ""},
		{"windows absolute", `C:\posts\a.md`, ""},
		{"simple", "content/posts/a.md", "content/posts/a.md"},
		{"backslashes converted", `content\posts\a.md`, "content/posts/a.md"},
		{"mixed separators", `content\posts/a.md`, "content/posts/a.md"},
		{"redundant segments", "content//posts/./a.md", "content/posts/a.md"},
		{"leading and trailing spaces", "  content/posts/a.md  ", "content/posts/a.md"},
		{"unicode path", "content/文章/你好.md", "content/文章/你好.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizePath(tc.in); got != tc.want {
				t.Fatalf("normalizePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBoolToInt(t *testing.T) {
	if got := boolToInt(true); got != 1 {
		t.Fatalf("boolToInt(true) = %d, want 1", got)
	}
	if got := boolToInt(false); got != 0 {
		t.Fatalf("boolToInt(false) = %d, want 0", got)
	}
}

// setupTestDB initializes the article store against a temporary database file
// and registers cleanup that closes it again.
func setupTestDB(t *testing.T) {
	t.Helper()
	t.Setenv("ARTICLES_DB_PATH", filepath.Join(t.TempDir(), "articles.db"))
	if err := Init(""); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		if err := Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
}

func TestOperationsFailBeforeInit(t *testing.T) {
	// Ensure the package-level db handle is unset regardless of test order.
	if err := Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := List(); err == nil {
		t.Fatal("List before Init returned nil error")
	}
	if _, _, err := GetContent("a.md"); err == nil {
		t.Fatal("GetContent before Init returned nil error")
	}
	if err := Upsert(models.ArticleRecord{}); err == nil {
		t.Fatal("Upsert before Init returned nil error")
	}
	if err := Delete("a.md"); err == nil {
		t.Fatal("Delete before Init returned nil error")
	}
	if err := ReplaceFromDisk(nil); err == nil {
		t.Fatal("ReplaceFromDisk before Init returned nil error")
	}
}

func TestInitCreatesDatabaseFileAndSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "articles.db")
	t.Setenv("ARTICLES_DB_PATH", dbPath)

	if err := Init(""); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = Close() })

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("database file not created at %s: %v", dbPath, err)
	}

	// Re-initializing with the schema already present must succeed.
	if err := Init(""); err != nil {
		t.Fatalf("second Init: %v", err)
	}
}

func TestReplaceFromDiskAndList(t *testing.T) {
	setupTestDB(t)

	records := []models.ArticleRecord{
		{Post: models.Post{Title: "  旧文  ", Lang: "zh", Path: `content\posts\old.md`, Date: "2024-01-01", Status: "published", StatusColor: "green"}},
		{Post: models.Post{Title: "新文", Lang: "zh", Path: "content/posts/new.md", Date: "2024-06-01", Status: "draft", StatusColor: "gray"}},
		{Post: models.Post{Title: "置顶", Lang: "en", Path: "content/posts/pinned.md", Date: "2023-01-01", Status: "published", StatusColor: "green", Pinned: true}},
		{Post: models.Post{Title: "空路径", Path: "   "}}, // must be skipped
	}
	if err := ReplaceFromDisk(records); err != nil {
		t.Fatalf("ReplaceFromDisk: %v", err)
	}

	posts, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("len(posts) = %d, want 3 (empty path must be skipped)", len(posts))
	}

	// Pinned first, then date_text DESC.
	if !posts[0].Pinned || posts[0].Title != "置顶" {
		t.Fatalf("posts[0] = %+v, want pinned article first", posts[0])
	}
	if posts[1].Title != "新文" || posts[2].Title != "旧文" {
		t.Fatalf("date ordering wrong: %q, %q", posts[1].Title, posts[2].Title)
	}
	// Title must be trimmed and path normalized to forward slashes.
	if posts[2].Path != "content/posts/old.md" {
		t.Fatalf("posts[2].Path = %q, want normalized forward-slash path", posts[2].Path)
	}
}

func TestReplaceFromDiskDefaultsTimestamps(t *testing.T) {
	setupTestDB(t)

	if err := ReplaceFromDisk([]models.ArticleRecord{
		{Post: models.Post{Title: "无时间戳", Path: "content/posts/a.md"}},
	}); err != nil {
		t.Fatalf("ReplaceFromDisk: %v", err)
	}

	conn, err := getDB()
	if err != nil {
		t.Fatalf("getDB: %v", err)
	}
	var createdAt, updatedAt string
	err = conn.QueryRow(`SELECT created_at, updated_at FROM articles WHERE path = ?`, "content/posts/a.md").Scan(&createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("query timestamps: %v", err)
	}
	if createdAt == "" || updatedAt == "" {
		t.Fatalf("created_at/updated_at must be defaulted, got %q / %q", createdAt, updatedAt)
	}
}

func TestReplaceFromDiskReplacesExistingRows(t *testing.T) {
	setupTestDB(t)

	first := []models.ArticleRecord{{Post: models.Post{Title: "第一批", Path: "content/posts/a.md"}}}
	if err := ReplaceFromDisk(first); err != nil {
		t.Fatalf("ReplaceFromDisk first: %v", err)
	}
	second := []models.ArticleRecord{{Post: models.Post{Title: "第二批", Path: "content/posts/b.md"}}}
	if err := ReplaceFromDisk(second); err != nil {
		t.Fatalf("ReplaceFromDisk second: %v", err)
	}

	posts, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 1 || posts[0].Title != "第二批" {
		t.Fatalf("posts = %+v, want only the second batch", posts)
	}
}

func TestGetContent(t *testing.T) {
	setupTestDB(t)

	record := models.ArticleRecord{
		Post:    models.Post{Title: "正文测试", Path: `content\posts\body.md`},
		Content: "# 标题\n\n正文内容",
	}
	if err := ReplaceFromDisk([]models.ArticleRecord{record}); err != nil {
		t.Fatalf("ReplaceFromDisk: %v", err)
	}

	content, found, err := GetContent("content/posts/body.md")
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}
	if !found || content != record.Content {
		t.Fatalf("GetContent = (%q, %v), want (%q, true)", content, found, record.Content)
	}

	// Lookup uses the same normalization as writes.
	if _, found, err = GetContent(`content\posts\body.md`); err != nil || !found {
		t.Fatalf("GetContent with backslashes = found %v, err %v; want found", found, err)
	}

	_, found, err = GetContent("content/posts/missing.md")
	if err != nil {
		t.Fatalf("GetContent missing: %v", err)
	}
	if found {
		t.Fatal("GetContent for missing path returned found = true")
	}
}

func TestUpsertInsertUpdateAndValidation(t *testing.T) {
	setupTestDB(t)

	base := models.ArticleRecord{
		Post:    models.Post{Title: "初版", Lang: "zh", Path: "content/posts/u.md", Date: "2024-01-01", Status: "draft"},
		Content: "v1",
	}
	if err := Upsert(base); err != nil {
		t.Fatalf("Upsert insert: %v", err)
	}

	updated := base
	updated.Title = "修订版"
	updated.Status = "published"
	updated.Content = "v2"
	if err := Upsert(updated); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}

	posts, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1 (upsert must not duplicate)", len(posts))
	}
	if posts[0].Title != "修订版" || posts[0].Status != "published" {
		t.Fatalf("posts[0] = %+v, want updated row", posts[0])
	}
	content, found, err := GetContent("content/posts/u.md")
	if err != nil || !found {
		t.Fatalf("GetContent = found %v, err %v", found, err)
	}
	if content != "v2" {
		t.Fatalf("content = %q, want %q", content, "v2")
	}

	if err := Upsert(models.ArticleRecord{Post: models.Post{Title: "无路径"}}); err == nil {
		t.Fatal("Upsert with empty path returned nil error")
	}
}

func TestUpsertDefaultsTimestampsAndPreservesCreatedAt(t *testing.T) {
	setupTestDB(t)

	record := models.ArticleRecord{
		Post:      models.Post{Title: "时间戳", Path: "content/posts/t.md"},
		CreatedAt: "2020-01-02T03:04:05Z",
	}
	if err := Upsert(record); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	conn, err := getDB()
	if err != nil {
		t.Fatalf("getDB: %v", err)
	}
	var createdAt, updatedAt string
	err = conn.QueryRow(`SELECT created_at, updated_at FROM articles WHERE path = ?`, "content/posts/t.md").Scan(&createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("query timestamps: %v", err)
	}
	if createdAt != "2020-01-02T03:04:05Z" {
		t.Fatalf("created_at = %q, want explicit value preserved", createdAt)
	}
	if updatedAt == "" {
		t.Fatal("updated_at must be defaulted when empty")
	}
}

func TestDelete(t *testing.T) {
	setupTestDB(t)

	records := []models.ArticleRecord{
		{Post: models.Post{Title: "留", Path: "content/posts/keep.md"}},
		{Post: models.Post{Title: "删", Path: `content\posts\drop.md`}},
	}
	if err := ReplaceFromDisk(records); err != nil {
		t.Fatalf("ReplaceFromDisk: %v", err)
	}

	if err := Delete("content/posts/drop.md"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, found, err := GetContent("content/posts/drop.md"); err != nil || found {
		t.Fatalf("deleted article still readable: found %v, err %v", found, err)
	}

	// Deleting a missing path must not error.
	if err := Delete("content/posts/never-existed.md"); err != nil {
		t.Fatalf("Delete missing path: %v", err)
	}

	posts, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 1 || posts[0].Title != "留" {
		t.Fatalf("posts = %+v, want only the kept article", posts)
	}
}

func TestListEmptyDatabase(t *testing.T) {
	setupTestDB(t)

	posts, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("len(posts) = %d, want 0", len(posts))
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	setupTestDB(t)
	// setupTestDB's cleanup will Close once more; a second Close must be a no-op.
	if err := Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("third Close: %v", err)
	}
}

func TestUpsertTrimsFields(t *testing.T) {
	setupTestDB(t)

	record := models.ArticleRecord{
		Post: models.Post{
			Title:  "  带空格  ",
			Lang:   " zh ",
			Path:   "content/posts/trim.md",
			Status: " draft ",
		},
	}
	if err := Upsert(record); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	posts, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}
	got := posts[0]
	if got.Title != "带空格" || got.Lang != "zh" || got.Status != "draft" {
		t.Fatalf("fields not trimmed: %+v", got)
	}
	if strings.ContainsAny(got.Path, `\`) {
		t.Fatalf("path not normalized: %q", got.Path)
	}
}
