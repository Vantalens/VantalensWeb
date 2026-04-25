package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"vantalens/talentwriter/internal/article"
	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
)

func HandleGetPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: getPosts()})
}

func HandleGetContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	content, err := readArticle(path)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]string{"content": content}})
}

func HandleSaveContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeJSONBody(w, r, &req, 2<<20); err != nil {
		return
	}
	if err := writeArticle(req.Path, req.Content); err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Saved"})
}

func HandleDeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeJSONBody(w, r, &req, 16<<10); err != nil {
		return
	}
	if err := removeArticle(req.Path); err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Deleted"})
}

func HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}
	var req struct {
		Title      string `json:"title"`
		Categories string `json:"categories"`
		Body       string `json:"body"`
		Draft      bool   `json:"draft"`
	}
	if err := decodeJSONBody(w, r, &req, 64<<10); err != nil {
		return
	}
	path, err := createArticle(req.Title, req.Categories, req.Body, req.Draft)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]string{"path": path}})
}

func getPosts() []models.Post {
	posts, err := article.List()
	if err == nil && len(posts) > 0 {
		return posts
	}
	posts, err = SyncArticlesToDatabase()
	if err != nil {
		return []models.Post{}
	}
	return posts
}

func SyncArticlesToDatabase() ([]models.Post, error) {
	root := articleRootDir()
	if root == "" {
		return []models.Post{}, fmt.Errorf("hugo path is not configured")
	}

	contentRoot := filepath.Join(root, "content")
	info, err := os.Stat(contentRoot)
	if err != nil || !info.IsDir() {
		return []models.Post{}, fmt.Errorf("content directory not found")
	}

	var records []models.ArticleRecord
	_ = filepath.Walk(contentRoot, func(path string, entry os.FileInfo, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || strings.HasPrefix(entry.Name(), "_") {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if !isArticlePath(relPath) {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		frontmatter := parseFrontmatter(string(content))
		lang := "en"
		normalized := strings.ToLower(strings.ReplaceAll(relPath, string(os.PathSeparator), "/"))
		if strings.Contains(normalized, "/zh-cn/") || strings.Contains(normalized, "/zh/") {
			lang = "zh-cn"
		}

		status := "PUBLISHED"
		color := "#22c55e"
		if frontmatter.Draft {
			status = "DRAFT"
			color = "#f59e0b"
		}

		date := entry.ModTime().Format("2006-01-02")
		if frontmatter.Date != "" {
			date = frontmatter.Date
		}

		records = append(records, models.ArticleRecord{
			Post: models.Post{
				Title:       fallbackArticleTitle(frontmatter.Title, relPath),
				Lang:        lang,
				Path:        strings.ReplaceAll(relPath, string(os.PathSeparator), "/"),
				Date:        date,
				Status:      status,
				StatusColor: color,
				Pinned:      frontmatter.Pinned,
			},
			Content:   string(content),
			CreatedAt: entry.ModTime().UTC().Format(time.RFC3339),
			UpdatedAt: entry.ModTime().UTC().Format(time.RFC3339),
		})
		return nil
	})

	if err := article.ReplaceFromDisk(records); err == nil {
		posts, listErr := article.List()
		if listErr == nil {
			return posts, nil
		}
	}

	posts := make([]models.Post, 0, len(records))
	for _, record := range records {
		posts = append(posts, record.Post)
	}
	sort.Slice(posts, func(i, j int) bool {
		if posts[i].Pinned != posts[j].Pinned {
			return posts[i].Pinned
		}
		return posts[i].Date > posts[j].Date
	})

	return posts, nil
}

func readArticle(relPath string) (string, error) {
	if content, ok, err := article.GetContent(relPath); err == nil && ok {
		return content, nil
	}
	path, err := resolveArticlePath(relPath)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found")
		}
		return "", err
	}
	return string(content), nil
}

func writeArticle(relPath, content string) error {
	path, err := resolveArticlePath(relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return err
	}
	return article.Upsert(articleRecordFromContent(relPath, content, time.Now()))
}

func removeArticle(relPath string) error {
	path, err := resolveArticlePath(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	_ = article.Delete(relPath)
	cleanupEmptyDirs(filepath.Dir(path), articleRootDir())
	return nil
}

func createArticle(title, categories, body string, draft bool) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}

	slug := slugify(title)
	relPath := filepath.Join("content", "posts", slug, "index.md")
	path, err := resolveArticlePath(relPath)
	if err != nil {
		return "", err
	}
	for i := 2; fileExists(path); i++ {
		relPath = filepath.Join("content", "posts", slug+"-"+strconv.Itoa(i), "index.md")
		path, err = resolveArticlePath(relPath)
		if err != nil {
			return "", err
		}
	}

	if strings.TrimSpace(body) == "" {
		body = "# " + title + "\n\n在这里开始写作。\n"
	}

	catList := normalizeCategories(categories)
	date := time.Now().Format(time.RFC3339)
	lines := []string{"---", fmt.Sprintf("title: %q", title), "date: " + date, "draft: " + strconv.FormatBool(draft), "pinned: false"}
	if len(catList) > 0 {
		lines = append(lines, "categories:")
		for _, category := range catList {
			lines = append(lines, "  - "+category)
		}
	}
	lines = append(lines, "---", "", body)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	if err := article.Upsert(articleRecordFromContent(relPath, content, time.Now())); err != nil {
		return "", err
	}
	return relPath, nil
}

func articleRecordFromContent(relPath, content string, updatedAt time.Time) models.ArticleRecord {
	frontmatter := parseFrontmatter(content)
	lang := "en"
	normalized := strings.ToLower(strings.ReplaceAll(relPath, "\\", "/"))
	if strings.Contains(normalized, "/zh-cn/") || strings.Contains(normalized, "/zh/") {
		lang = "zh-cn"
	}
	status := "PUBLISHED"
	color := "#22c55e"
	if frontmatter.Draft {
		status = "DRAFT"
		color = "#f59e0b"
	}
	return models.ArticleRecord{
		Post: models.Post{
			Title:       fallbackArticleTitle(frontmatter.Title, relPath),
			Lang:        lang,
			Path:        strings.ReplaceAll(relPath, "\\", "/"),
			Date:        frontmatter.Date,
			Status:      status,
			StatusColor: color,
			Pinned:      frontmatter.Pinned,
		},
		Content:   content,
		UpdatedAt: updatedAt.UTC().Format(time.RFC3339),
	}
}

func parseFrontmatter(content string) models.Frontmatter {
	meta := models.Frontmatter{Title: "Untitled", Date: time.Now().Format("2006-01-02")}
	if !strings.HasPrefix(content, "---") {
		return meta
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return meta
	}
	currentKey := ""
	for _, line := range strings.Split(parts[1], "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "-") {
			if currentKey == "categories" {
				category := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
				category = strings.Trim(category, `"`)
				if category != "" {
					meta.Categories = append(meta.Categories, category)
				}
			}
			continue
		}
		pair := strings.SplitN(trimmed, ":", 2)
		if len(pair) != 2 {
			continue
		}
		currentKey = strings.TrimSpace(pair[0])
		value := strings.TrimSpace(pair[1])
		value = strings.Trim(value, `"`)
		switch currentKey {
		case "title":
			meta.Title = value
		case "date":
			if len(value) >= 10 {
				meta.Date = value[:10]
			} else if value != "" {
				meta.Date = value
			}
		case "draft":
			meta.Draft = strings.EqualFold(value, "true")
		case "pinned":
			meta.Pinned = strings.EqualFold(value, "true")
		case "categories":
			if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				raw := strings.Trim(value, "[]")
				for _, category := range strings.Split(raw, ",") {
					category = strings.TrimSpace(strings.Trim(category, `"`))
					if category != "" {
						meta.Categories = append(meta.Categories, category)
					}
				}
			}
		}
	}
	return meta
}

func articleRootDir() string {
	cfg := config.GetConfig()
	candidates := []string{}
	if cfg != nil && strings.TrimSpace(cfg.HugoPath) != "" {
		candidates = append(candidates, cfg.HugoPath)
	}
	candidates = append(candidates, "..", ".")
	for _, candidate := range candidates {
		root, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, statErr := os.Stat(filepath.Join(root, "content")); statErr == nil && info.IsDir() {
			return root
		}
	}
	if cfg != nil && strings.TrimSpace(cfg.HugoPath) != "" {
		root, err := filepath.Abs(cfg.HugoPath)
		if err == nil {
			return root
		}
	}
	root, _ := filepath.Abs("..")
	return root
}

func resolveArticlePath(relPath string) (string, error) {
	root := articleRootDir()
	if root == "" {
		return "", fmt.Errorf("hugo path is not configured")
	}
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	normalized := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(relPath, "\\", "/")))
	if filepath.IsAbs(normalized) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	if filepath.Ext(normalized) != ".md" {
		return "", fmt.Errorf("only .md files are allowed")
	}
	fullPath := filepath.Join(root, normalized)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path is outside hugo root")
	}
	return absPath, nil
}

func isArticlePath(relPath string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(relPath, "\\", "/"))
	return strings.HasSuffix(normalized, ".md") && (strings.Contains(normalized, "/post/") || strings.Contains(normalized, "/posts/"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fallbackArticleTitle(title, relPath string) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}
	if relPath == "" {
		return "Untitled"
	}
	base := filepath.Base(filepath.Dir(relPath))
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	}
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	if base == "" {
		return "Untitled"
	}
	return strings.TrimSpace(base)
}

func cleanupEmptyDirs(dir, stopDir string) {
	for dir != "" && stopDir != "" {
		if samePath(dir, stopDir) {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		_ = os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func slugify(title string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		slug = "post-" + time.Now().Format("20060102150405")
	}
	return slug
}

func normalizeCategories(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
