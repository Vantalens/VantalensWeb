package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/comment"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/models"
	"vantalens/talentwriter/internal/oplog"
)

var (
	opLogMu    sync.RWMutex
	opLogStore *oplog.Store
)

// SetOpLogStore overrides the operation log store (used by tests).
func SetOpLogStore(store *oplog.Store) {
	opLogMu.Lock()
	opLogStore = store
	opLogMu.Unlock()
}

func getOpLogStore() *oplog.Store {
	opLogMu.RLock()
	store := opLogStore
	opLogMu.RUnlock()
	if store != nil {
		return store
	}
	opLogMu.Lock()
	defer opLogMu.Unlock()
	if opLogStore != nil {
		return opLogStore
	}
	opened, err := oplog.Open(config.GetOplogPath(articleRootDir()))
	if err != nil {
		log.Printf("[OPLOG] disabled: %v", err)
		return nil
	}
	opLogStore = opened
	return opLogStore
}

// ensureOplogID returns the logical operation id for this request, generating
// and pinning one on the request header when absent. The pinned id is
// forwarded to the remote authority backend so both sides log the same id.
func ensureOplogID(r *http.Request) string {
	id := strings.TrimSpace(r.Header.Get(oplog.HeaderOplogID))
	if id == "" {
		id = oplog.NewID()
		r.Header.Set(oplog.HeaderOplogID, id)
	}
	return id
}

func opActor(r *http.Request) string {
	if claims, _, err := auth.AuthenticateRequest(r); err == nil && strings.TrimSpace(claims.Sub) != "" {
		return strings.TrimSpace(claims.Sub)
	}
	if strings.TrimSpace(r.Header.Get("X-Admin-Token")) != "" {
		return "admin-token"
	}
	return "unknown"
}

// opSyncedByDefault reports whether operations executed here are already
// authoritative remotely (this instance IS the authority backend).
func opSyncedByDefault() bool {
	return config.AuthorityBackendEnabled()
}

// recordOp appends one operation entry. Logging failures only warn and never
// affect the main operation flow.
func recordOp(r *http.Request, opType, target, summary, snapshot string, opErr error, synced bool) {
	store := getOpLogStore()
	if store == nil {
		return
	}
	entry := oplog.Entry{
		ID:       ensureOplogID(r),
		Origin:   oplog.OriginLocal,
		Actor:    opActor(r),
		Type:     opType,
		Target:   target,
		Summary:  summary,
		Snapshot: snapshot,
		Result:   oplog.FailedResult(opErr),
		Synced:   synced,
	}
	if _, err := store.Append(entry); err != nil {
		log.Printf("[OPLOG] append failed for %s %s: %v", opType, target, err)
	}
}

// snapshotCommentByID captures the current comment row as JSON (best effort;
// empty string when the local cache does not have the row).
func snapshotCommentByID(id string) string {
	c, ok, err := comment.GetComment(id)
	if err != nil || !ok {
		return ""
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(raw)
}

// snapshotCommentSettings captures the current comment settings as JSON.
func snapshotCommentSettings() string {
	raw, err := json.Marshal(comment.LoadSettings())
	if err != nil {
		return ""
	}
	return string(raw)
}

type oplogListData struct {
	Entries []oplog.Entry `json:"entries"`
	Total   int           `json:"total"`
	Pending int           `json:"pending"`
}

func HandleOplogList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	store := getOpLogStore()
	if store == nil {
		RespondJSON(w, http.StatusServiceUnavailable, models.APIResponse{Success: false, Message: "oplog is unavailable"})
		return
	}
	opType := strings.TrimSpace(r.URL.Query().Get("type"))
	limit := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 5000 {
		limit = 5000
	}
	entries, err := store.List(opType, limit)
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	total, pending, _ := store.Stats()
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: oplogListData{Entries: entries, Total: total, Pending: pending}})
}

func HandleOplogEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	store := getOpLogStore()
	if store == nil {
		RespondJSON(w, http.StatusServiceUnavailable, models.APIResponse{Success: false, Message: "oplog is unavailable"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	entry, ok, err := store.Get(id)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	if !ok {
		RespondJSON(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "oplog entry not found"})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: entry})
}

func HandleOplogRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	store := getOpLogStore()
	if store == nil {
		RespondJSON(w, http.StatusServiceUnavailable, models.APIResponse{Success: false, Message: "oplog is unavailable"})
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeJSONBody(w, r, &req, 8<<10); err != nil {
		return
	}
	entry, ok, err := store.Get(req.ID)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	if !ok {
		RespondJSON(w, http.StatusNotFound, models.APIResponse{Success: false, Message: "oplog entry not found"})
		return
	}
	if err := rollbackOplogEntry(entry); err != nil {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "回溯失败: " + err.Error()})
		return
	}
	recordOp(r, entry.Type+oplog.RollbackSuffix, entry.Target, "回溯操作「"+entry.Summary+"」", "", nil, opSyncedByDefault())
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "已回溯到该操作之前的状态", Data: entry})
}

func rollbackOplogEntry(e oplog.Entry) error {
	if e.IsRollback() {
		return fmt.Errorf("回溯记录本身不可再次回溯")
	}
	if !e.Success() {
		return fmt.Errorf("失败的操作没有改变状态，无需回溯")
	}
	switch e.Type {
	case oplog.TypePostSave, oplog.TypePostDelete:
		if strings.TrimSpace(e.Snapshot) == "" {
			return fmt.Errorf("该记录缺少文章快照，无法回溯")
		}
		return writeArticle(e.Target, e.Snapshot)
	case oplog.TypePostCreate:
		return removeArticle(e.Target)
	case oplog.TypePostRestore:
		_, err := trashArticle(e.Target)
		return err
	case oplog.TypePostPurge:
		var snap struct {
			OriginalPath string `json:"original_path"`
			Content      string `json:"content"`
		}
		if strings.TrimSpace(e.Snapshot) == "" || json.Unmarshal([]byte(e.Snapshot), &snap) != nil || snap.OriginalPath == "" {
			return fmt.Errorf("永久删除没有可用快照，不可回溯")
		}
		return writeArticle(snap.OriginalPath, snap.Content)
	case oplog.TypeCommentApprove:
		return comment.DeleteComment("", e.Target)
	case oplog.TypeCommentDelete:
		var c models.Comment
		if strings.TrimSpace(e.Snapshot) == "" || json.Unmarshal([]byte(e.Snapshot), &c) != nil || c.ID == "" {
			return fmt.Errorf("该记录缺少评论快照，无法回溯")
		}
		return comment.RestoreComment(c)
	case oplog.TypeSettingsSave:
		var settings models.CommentSettings
		if strings.TrimSpace(e.Snapshot) == "" || json.Unmarshal([]byte(e.Snapshot), &settings) != nil {
			return fmt.Errorf("该记录缺少设置快照，无法回溯")
		}
		return comment.SaveSettings(settings)
	case oplog.TypePublish:
		return fmt.Errorf("发布操作不可回溯")
	default:
		return fmt.Errorf("不支持回溯的操作类型: %s", e.Type)
	}
}

type oplogCompareData struct {
	LocalOnly   []oplog.Entry `json:"local_only"`
	RemoteOnly  []oplog.Entry `json:"remote_only"`
	LocalTotal  int           `json:"local_total"`
	RemoteTotal int           `json:"remote_total"`
}

func HandleOplogCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	data, err := compareOplogWithRemote(r)
	if err != nil {
		RespondJSON(w, http.StatusBadGateway, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: data})
}

func compareOplogWithRemote(r *http.Request) (oplogCompareData, error) {
	store := getOpLogStore()
	if store == nil {
		return oplogCompareData{}, fmt.Errorf("oplog is unavailable")
	}
	if !remoteAdminConfigured() {
		return oplogCompareData{}, fmt.Errorf("远端未配置：需要 REMOTE_ADMIN_BASE，且当前端不是权威后端")
	}
	localEntries, err := store.List("", 5000)
	if err != nil {
		return oplogCompareData{}, err
	}
	result, err := proxyRemoteAdmin(r, http.MethodGet, "/api/oplog/list?limit=5000", nil)
	if err != nil {
		return oplogCompareData{}, fmt.Errorf("拉取远端操作日志失败: %w", err)
	}
	raw, err := json.Marshal(result.Response.Data)
	if err != nil {
		return oplogCompareData{}, err
	}
	var remoteList oplogListData
	if err := json.Unmarshal(raw, &remoteList); err != nil {
		return oplogCompareData{}, fmt.Errorf("远端操作日志解析失败: %w", err)
	}

	localIDs := map[string]bool{}
	for _, e := range localEntries {
		localIDs[e.ID] = true
	}
	remoteIDs := map[string]bool{}
	for _, e := range remoteList.Entries {
		remoteIDs[e.ID] = true
	}
	data := oplogCompareData{
		LocalOnly:   []oplog.Entry{},
		RemoteOnly:  []oplog.Entry{},
		LocalTotal:  len(localEntries),
		RemoteTotal: len(remoteList.Entries),
	}
	for _, e := range localEntries {
		if !remoteIDs[e.ID] {
			data.LocalOnly = append(data.LocalOnly, e)
		}
	}
	for _, e := range remoteList.Entries {
		if !localIDs[e.ID] {
			data.RemoteOnly = append(data.RemoteOnly, e)
		}
	}
	return data, nil
}

type oplogSyncItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Target  string `json:"target"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func HandleOplogSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireAdminAccess(w, r) {
		return
	}
	store := getOpLogStore()
	if store == nil {
		RespondJSON(w, http.StatusServiceUnavailable, models.APIResponse{Success: false, Message: "oplog is unavailable"})
		return
	}
	var req struct {
		Direction string `json:"direction"`
	}
	if err := decodeJSONBody(w, r, &req, 8<<10); err != nil {
		return
	}
	direction := strings.TrimSpace(req.Direction)
	if direction != "push" && direction != "pull" {
		RespondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Message: "direction 必须是 push 或 pull"})
		return
	}
	cmp, err := compareOplogWithRemote(r)
	if err != nil {
		RespondJSON(w, http.StatusBadGateway, models.APIResponse{Success: false, Message: err.Error()})
		return
	}

	todo := cmp.LocalOnly
	if direction == "pull" {
		todo = cmp.RemoteOnly
	}
	// replay in chronological order (compare returns newest-first)
	for i, j := 0, len(todo)-1; i < j; i, j = i+1, j-1 {
		todo[i], todo[j] = todo[j], todo[i]
	}

	items := []oplogSyncItem{}
	succeeded, failed := 0, 0
	for _, e := range todo {
		var opErr error
		if direction == "push" {
			opErr = replayOplogToRemote(r, e)
			if opErr == nil {
				if _, err := store.MarkSynced(e.ID); err != nil {
					log.Printf("[OPLOG] mark synced failed for %s: %v", e.ID, err)
				}
			}
		} else {
			opErr = applyRemoteOplogLocally(r, e)
			if opErr == nil {
				entry := oplog.Entry{
					ID:       e.ID,
					Ts:       e.Ts,
					Origin:   oplog.OriginRemote,
					Actor:    e.Actor,
					Type:     e.Type,
					Target:   e.Target,
					Summary:  "自远端同步: " + e.Summary,
					Snapshot: e.Snapshot,
					Result:   oplog.ResultSuccess,
					Synced:   true,
				}
				if _, err := store.Append(entry); err != nil {
					opErr = fmt.Errorf("本地落账失败: %w", err)
				}
			}
		}
		item := oplogSyncItem{ID: e.ID, Type: e.Type, Target: e.Target, Success: opErr == nil}
		if opErr != nil {
			item.Error = opErr.Error()
			failed++
		} else {
			succeeded++
		}
		items = append(items, item)
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]interface{}{
		"direction": direction,
		"results":   items,
		"succeeded": succeeded,
		"failed":    failed,
	}})
}

// replayOplogToRemote pushes one local-only operation to the remote authority
// backend by calling the matching remote API through proxyRemoteAdmin.
func replayOplogToRemote(r *http.Request, e oplog.Entry) error {
	r.Header.Set(oplog.HeaderOplogID, e.ID)
	switch e.Type {
	case oplog.TypePostSave:
		content, err := readArticle(e.Target)
		if err != nil {
			return fmt.Errorf("读取本地文章失败: %w", err)
		}
		body, _ := json.Marshal(map[string]string{"path": e.Target, "content": content})
		_, err = proxyRemoteAdmin(r, http.MethodPost, "/api/save_content", body)
		return err
	case oplog.TypePostCreate:
		content, err := readArticle(e.Target)
		if err != nil {
			return fmt.Errorf("读取本地文章失败: %w", err)
		}
		doc, err := parseArticleDocument(content)
		if err != nil {
			return fmt.Errorf("解析本地文章失败: %w", err)
		}
		body, _ := json.Marshal(map[string]interface{}{
			"title":      doc.Metadata.Title,
			"categories": strings.Join(doc.Metadata.Categories, ","),
			"body":       doc.Body,
			"draft":      doc.Metadata.Draft,
		})
		_, err = proxyRemoteAdmin(r, http.MethodPost, "/api/create_post", body)
		return err
	case oplog.TypePostDelete:
		body, _ := json.Marshal(map[string]string{"path": e.Target})
		_, err := proxyRemoteAdmin(r, http.MethodPost, "/api/delete_post", body)
		return err
	case oplog.TypeCommentApprove:
		_, err := proxyRemoteAdmin(r, http.MethodPost, "/api/admin/comments/"+url.PathEscape(e.Target)+"/approve", nil)
		return err
	case oplog.TypeCommentDelete:
		_, err := proxyRemoteAdmin(r, http.MethodPost, "/api/admin/comments/"+url.PathEscape(e.Target)+"/delete", nil)
		return err
	case oplog.TypeSettingsSave:
		body, _ := json.Marshal(comment.LoadSettings())
		_, err := proxyRemoteAdmin(r, http.MethodPost, "/api/settings/save", body)
		return err
	case oplog.TypePublish:
		_, err := proxyRemoteAdmin(r, http.MethodPost, "/api/publish", nil)
		return err
	default:
		return fmt.Errorf("该类型暂不支持推送到远端: %s", e.Type)
	}
}

// applyRemoteOplogLocally pulls one remote-only operation into the local
// Hugo files / databases.
func applyRemoteOplogLocally(r *http.Request, e oplog.Entry) error {
	switch e.Type {
	case oplog.TypePostSave, oplog.TypePostCreate:
		result, err := proxyRemoteAdmin(r, http.MethodGet, "/api/get_content?path="+url.QueryEscape(e.Target), nil)
		if err != nil {
			return fmt.Errorf("拉取远端文章内容失败: %w", err)
		}
		raw, _ := json.Marshal(result.Response.Data)
		var doc ArticleDocument
		if err := json.Unmarshal(raw, &doc); err != nil || strings.TrimSpace(doc.Content) == "" {
			return fmt.Errorf("远端文章内容解析失败")
		}
		return writeArticle(e.Target, doc.Content)
	case oplog.TypePostDelete:
		_, err := trashArticle(e.Target)
		return err
	case oplog.TypeCommentApprove:
		return comment.ApproveComment("", e.Target)
	case oplog.TypeCommentDelete:
		return comment.DeleteComment("", e.Target)
	case oplog.TypeSettingsSave:
		result, err := proxyRemoteAdmin(r, http.MethodGet, "/api/settings", nil)
		if err != nil {
			return fmt.Errorf("拉取远端设置失败: %w", err)
		}
		raw, _ := json.Marshal(result.Response.Data)
		var settings models.CommentSettings
		if err := json.Unmarshal(raw, &settings); err != nil {
			return fmt.Errorf("远端设置解析失败: %w", err)
		}
		return comment.SaveSettings(settings)
	default:
		return fmt.Errorf("该类型暂不支持拉取到本地: %s", e.Type)
	}
}
