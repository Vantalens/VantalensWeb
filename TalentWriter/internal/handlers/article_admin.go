package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"vantalens/talentwriter/internal/article"
	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

type ArticleTrashRecord struct {
	ID           string `json:"id"`
	OriginalPath string `json:"original_path"`
	StoredPath   string `json:"-"`
	Title        string `json:"title"`
	DeletedAt    string `json:"deleted_at"`
}

func articleTrashRoot() string {
	if configured := strings.TrimSpace(config.GetEnv("ARTICLE_TRASH_PATH", "")); configured != "" {
		return configured
	}
	return filepath.Join(filepath.Dir(config.GetArticlesDBPath(articleRootDir())), "trash")
}

func trashArticle(relPath string) (ArticleTrashRecord, error) {
	path, err := resolveArticlePath(relPath)
	if err != nil {
		return ArticleTrashRecord{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ArticleTrashRecord{}, err
	}
	doc, err := parseArticleDocument(string(content))
	if err != nil {
		return ArticleTrashRecord{}, err
	}
	id := time.Now().UTC().Format("20060102T150405.000000000Z")
	root := articleTrashRoot()
	record := ArticleTrashRecord{ID: id, OriginalPath: strings.ReplaceAll(relPath, "\\", "/"), StoredPath: filepath.Join(root, id+".md"), Title: doc.Metadata.Title, DeletedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := writeFileAtomic(record.StoredPath, content, 0o600); err != nil {
		return ArticleTrashRecord{}, err
	}
	manifest, _ := json.MarshalIndent(record, "", "  ")
	if err := writeFileAtomic(filepath.Join(root, id+".json"), manifest, 0o600); err != nil {
		_ = os.Remove(record.StoredPath)
		return ArticleTrashRecord{}, err
	}
	if err := os.Remove(path); err != nil {
		_ = os.Remove(record.StoredPath)
		_ = os.Remove(filepath.Join(root, id+".json"))
		return ArticleTrashRecord{}, err
	}
	if err := article.Delete(relPath); err != nil {
		if restoreErr := writeFileAtomic(path, content, 0o600); restoreErr != nil {
			return ArticleTrashRecord{}, fmt.Errorf("delete article database row: %v; restore article file: %w", err, restoreErr)
		}
		_ = os.Remove(record.StoredPath)
		_ = os.Remove(filepath.Join(root, id+".json"))
		return ArticleTrashRecord{}, fmt.Errorf("delete article database row: %w", err)
	}
	cleanupEmptyDirs(filepath.Dir(path), articleRootDir())
	return record, nil
}

func listTrashedArticles() ([]ArticleTrashRecord, error) {
	entries, err := os.ReadDir(articleTrashRoot())
	if os.IsNotExist(err) {
		return []ArticleTrashRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]ArticleTrashRecord, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(articleTrashRoot(), entry.Name()))
		if readErr != nil {
			continue
		}
		var item ArticleTrashRecord
		if json.Unmarshal(raw, &item) == nil && item.ID != "" {
			item.StoredPath = filepath.Join(articleTrashRoot(), item.ID+".md")
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DeletedAt > items[j].DeletedAt })
	return items, nil
}

func readTrashRecord(id string) (ArticleTrashRecord, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, `/\\`) {
		return ArticleTrashRecord{}, fmt.Errorf("invalid trash id")
	}
	raw, err := os.ReadFile(filepath.Join(articleTrashRoot(), id+".json"))
	if os.IsNotExist(err) {
		return ArticleTrashRecord{}, fmt.Errorf("trash item not found")
	}
	if err != nil {
		return ArticleTrashRecord{}, err
	}
	var record ArticleTrashRecord
	if err := json.Unmarshal(raw, &record); err != nil || record.ID != id {
		return ArticleTrashRecord{}, fmt.Errorf("invalid trash record")
	}
	record.StoredPath = filepath.Join(articleTrashRoot(), id+".md")
	return record, nil
}

func restoreTrashedArticle(id string) (ArticleTrashRecord, error) {
	record, err := readTrashRecord(id)
	if err != nil {
		return ArticleTrashRecord{}, err
	}
	target, err := resolveArticlePath(record.OriginalPath)
	if err != nil {
		return ArticleTrashRecord{}, err
	}
	if _, err := os.Stat(target); err == nil {
		return ArticleTrashRecord{}, fmt.Errorf("article already exists at %s", record.OriginalPath)
	}
	content, err := os.ReadFile(record.StoredPath)
	if err != nil {
		return ArticleTrashRecord{}, err
	}
	if err := writeFileAtomic(target, content, 0o600); err != nil {
		return ArticleTrashRecord{}, err
	}
	if err := article.Upsert(articleRecordFromContent(record.OriginalPath, string(content), time.Now())); err != nil {
		_ = os.Remove(target)
		return ArticleTrashRecord{}, err
	}
	_ = os.Remove(record.StoredPath)
	_ = os.Remove(filepath.Join(articleTrashRoot(), record.ID+".json"))
	return record, nil
}

func purgeTrashedArticle(id string) error {
	record, err := readTrashRecord(id)
	if err != nil {
		return err
	}
	if err := os.Remove(record.StoredPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Remove(filepath.Join(articleTrashRoot(), record.ID+".json"))
}

func HandleTrashPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	items, err := listTrashedArticles()
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: items})
}

func HandleRestorePost(w http.ResponseWriter, r *http.Request) {
	handleTrashMutation(w, r, false)
}

func HandlePurgePost(w http.ResponseWriter, r *http.Request) {
	handleTrashMutation(w, r, true)
}

func handleTrashMutation(w http.ResponseWriter, r *http.Request, purge bool) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeJSONBody(w, r, &req, 8<<10); err != nil {
		return
	}
	if purge {
		if err := purgeTrashedArticle(req.ID); err != nil {
			RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Trash item permanently deleted"})
		return
	}
	record, err := restoreTrashedArticle(req.ID)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Article restored", Data: record})
}
