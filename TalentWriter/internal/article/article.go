package article

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

var (
	db   *sql.DB
	dbMu sync.RWMutex
)

func Init(hugoPath string) error {
	path := config.GetArticlesDBPath(hugoPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	if err := ensureSchema(conn); err != nil {
		_ = conn.Close()
		return err
	}
	dbMu.Lock()
	if db != nil {
		_ = db.Close()
	}
	db = conn
	dbMu.Unlock()
	return nil
}

func ensureSchema(conn *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS articles (
	path TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	lang TEXT NOT NULL,
	date_text TEXT NOT NULL,
	status TEXT NOT NULL,
	status_color TEXT NOT NULL,
	pinned INTEGER NOT NULL DEFAULT 0,
	content TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_articles_updated_at ON articles(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_articles_lang ON articles(lang);
CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
`
	_, err := conn.Exec(schema)
	return err
}

func ReplaceFromDisk(records []models.ArticleRecord) error {
	conn, err := getDB()
	if err != nil {
		return err
	}
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM articles`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
INSERT INTO articles(path, title, lang, date_text, status, status_color, pinned, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, record := range records {
		record.Path = normalizePath(record.Path)
		if record.Path == "" {
			continue
		}
		createdAt := record.CreatedAt
		if createdAt == "" {
			createdAt = now
		}
		updatedAt := record.UpdatedAt
		if updatedAt == "" {
			updatedAt = now
		}
		if _, err := stmt.Exec(
			record.Path,
			strings.TrimSpace(record.Title),
			strings.TrimSpace(record.Lang),
			strings.TrimSpace(record.Date),
			strings.TrimSpace(record.Status),
			strings.TrimSpace(record.StatusColor),
			boolToInt(record.Pinned),
			record.Content,
			createdAt,
			updatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func List() ([]models.Post, error) {
	conn, err := getDB()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(`
SELECT title, lang, path, date_text, status, status_color, pinned
FROM articles
ORDER BY pinned DESC, date_text DESC, updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var pinned int
		if err := rows.Scan(&post.Title, &post.Lang, &post.Path, &post.Date, &post.Status, &post.StatusColor, &pinned); err != nil {
			return nil, err
		}
		post.Pinned = pinned == 1
		posts = append(posts, post)
	}
	return posts, nil
}

func GetContent(path string) (string, bool, error) {
	conn, err := getDB()
	if err != nil {
		return "", false, err
	}
	var content string
	err = conn.QueryRow(`SELECT content FROM articles WHERE path = ?`, normalizePath(path)).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return content, true, nil
}

func Upsert(record models.ArticleRecord) error {
	conn, err := getDB()
	if err != nil {
		return err
	}
	record.Path = normalizePath(record.Path)
	if record.Path == "" {
		return errors.New("article path is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if record.CreatedAt == "" {
		record.CreatedAt = now
	}
	if record.UpdatedAt == "" {
		record.UpdatedAt = now
	}
	_, err = conn.Exec(`
INSERT INTO articles(path, title, lang, date_text, status, status_color, pinned, content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
	title=excluded.title,
	lang=excluded.lang,
	date_text=excluded.date_text,
	status=excluded.status,
	status_color=excluded.status_color,
	pinned=excluded.pinned,
	content=excluded.content,
	updated_at=excluded.updated_at`,
		record.Path,
		strings.TrimSpace(record.Title),
		strings.TrimSpace(record.Lang),
		strings.TrimSpace(record.Date),
		strings.TrimSpace(record.Status),
		strings.TrimSpace(record.StatusColor),
		boolToInt(record.Pinned),
		record.Content,
		record.CreatedAt,
		record.UpdatedAt,
	)
	return err
}

func Delete(path string) error {
	conn, err := getDB()
	if err != nil {
		return err
	}
	_, err = conn.Exec(`DELETE FROM articles WHERE path = ?`, normalizePath(path))
	return err
}

func getDB() (*sql.DB, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	if db == nil {
		return nil, errors.New("article database not initialized")
	}
	return db, nil
}

func normalizePath(path string) string {
	normalized := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")))
	if normalized == "." || normalized == "" || filepath.IsAbs(normalized) {
		return ""
	}
	return strings.ReplaceAll(normalized, "\\", "/")
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
