package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"vantalens/talentwriter/internal/analytics"
	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/models"
)

func HandleAnalyticsCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	var req models.AnalyticsCollectRequest
	if err := decodeJSONBody(w, r, &req, 64<<10); err != nil {
		return
	}

	req.Path = strings.TrimSpace(req.Path)
	req.Title = strings.TrimSpace(req.Title)
	req.Referrer = strings.TrimSpace(req.Referrer)
	req.DNSHost = strings.TrimSpace(req.DNSHost)
	req.Language = strings.TrimSpace(req.Language)
	req.Timezone = strings.TrimSpace(req.Timezone)
	req.Screen = strings.TrimSpace(req.Screen)
	if req.Path == "" {
		req.Path = "/"
	}

	record, err := analytics.TrackVisit(r, req)
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
		return
	}

	RespondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":         record.ID,
			"created_at": record.CreatedAt,
		},
	})
}

func HandleAnalyticsStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !auth.RequireAuth(w, r) {
		return
	}

	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	stats, err := analytics.GetSiteStatistics(limit)
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: stats})
}

func HandleAnalyticsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/platform/analytics" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(AnalyticsHTML()))
}

func AnalyticsHTML() string {
	page := `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Vantalens 访问监控</title>
  <style>
    :root {
      --bg: #0b1120;
      --panel: rgba(17, 24, 39, 0.86);
      --card: rgba(255,255,255,0.04);
      --line: rgba(148,163,184,0.16);
      --text: #e5edf8;
      --muted: #95a3bb;
      --accent: #38bdf8;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(56,189,248,0.14), transparent 28%),
        linear-gradient(135deg, #020617 0%, #0b1120 55%, #111827 100%);
    }
    .shell { max-width: 1440px; margin: 0 auto; padding: 24px; }
    .topbar, .panel {
      border: 1px solid var(--line);
      border-radius: 22px;
      background: var(--panel);
      box-shadow: 0 20px 60px rgba(2,6,23,0.4);
      backdrop-filter: blur(16px);
    }
    .topbar {
      display: flex; justify-content: space-between; gap: 16px; align-items: center;
      padding: 18px 20px; margin-bottom: 18px;
    }
    .topbar h1 { margin: 0; font-size: 24px; }
    .topbar p { margin: 6px 0 0; color: var(--muted); font-size: 13px; }
    .actions { display: flex; gap: 10px; flex-wrap: wrap; }
    .btn, input {
      border: 1px solid var(--line);
      border-radius: 14px;
      background: rgba(255,255,255,0.05);
      color: var(--text);
      padding: 11px 14px;
      font: inherit;
    }
    .btn { cursor: pointer; }
    .btn.primary { background: linear-gradient(135deg, rgba(56,189,248,0.25), rgba(14,165,233,0.18)); }
    .grid { display: grid; gap: 18px; }
    .stats { grid-template-columns: repeat(4, minmax(0, 1fr)); margin-bottom: 18px; }
    .card { padding: 16px; border-radius: 18px; background: var(--card); border: 1px solid var(--line); }
    .label { color: var(--muted); font-size: 12px; margin-bottom: 8px; }
    .value { font-size: 28px; font-weight: 700; }
    .panel { padding: 18px; margin-bottom: 18px; }
    .panel h2 { margin: 0 0 14px; font-size: 18px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 8px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-weight: 600; font-size: 12px; }
    td { font-size: 13px; }
    .mono { font-family: "Cascadia Mono", "Consolas", monospace; font-size: 12px; }
    .login { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 14px; }
    .muted { color: var(--muted); }
    .split { display: grid; gap: 18px; grid-template-columns: 1.2fr 1fr; }
    @media (max-width: 1100px) { .stats, .split { grid-template-columns: 1fr 1fr; } }
    @media (max-width: 760px) { .topbar, .login, .stats, .split { grid-template-columns: 1fr; display: grid; } }
  </style>
</head>
<body>
  <div class="shell">
    <header class="topbar">
      <div>
        <h1>访问监控</h1>
        <p>统计网站总访问、分页访问、访客 IP、设备、时间、地区，以及前端上报的 WebRTC 信息。DNS 字段记录访问域名，不是访客系统 DNS 解析器。</p>
      </div>
      <div class="actions">
        <a class="btn" href="/platform/control">总控平台</a>
        <button class="btn primary" onclick="loadStats()">刷新</button>
      </div>
    </header>

    <section class="panel">
      <h2>登录</h2>
      <div class="login">
        <input id="login-user" value="admin" placeholder="用户名">
        <input id="login-pass" type="password" placeholder="密码">
        <button class="btn primary" onclick="login()">登录</button>
        <button class="btn" onclick="logout()">退出</button>
        <span id="auth-status" class="muted">未登录</span>
      </div>
    </section>

    <section class="grid stats">
      <div class="card"><div class="label">总访问量</div><div id="total-views" class="value">-</div></div>
      <div class="card"><div class="label">总分页数</div><div id="total-pages" class="value">-</div></div>
      <div class="card"><div class="label">独立 IP</div><div id="unique-ips" class="value">-</div></div>
      <div class="card"><div class="label">独立会话</div><div id="unique-sessions" class="value">-</div></div>
    </section>

    <section class="split">
      <section class="panel">
        <h2>分页访问统计</h2>
        <div id="pages-box" class="muted">暂无数据</div>
      </section>
      <section class="panel">
        <h2>访客 IP</h2>
        <div id="visitors-box" class="muted">暂无数据</div>
      </section>
    </section>

    <section class="panel">
      <h2>最近访问</h2>
      <div id="recent-box" class="muted">暂无数据</div>
    </section>
  </div>

  <script>
    function authHeaders() {
      const token = localStorage.getItem('ws_token') || localStorage.getItem('auth_token');
      return token ? { Authorization: 'Bearer ' + token } : {};
    }

    function setStatus(text) {
      document.getElementById('auth-status').textContent = text;
    }

    async function login() {
      const username = document.getElementById('login-user').value.trim() || 'admin';
      const password = document.getElementById('login-pass').value;
      const res = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      const data = await res.json().catch(() => ({}));
      const token = data?.data?.access_token || data?.data?.token;
      if (!res.ok || !data.success || !token) {
        setStatus('登录失败');
        return;
      }
      localStorage.setItem('ws_token', token);
      localStorage.setItem('auth_token', token);
      setStatus('已登录');
      await loadStats();
    }

    function logout() {
      localStorage.removeItem('ws_token');
      localStorage.removeItem('auth_token');
      setStatus('已退出');
    }

    function renderTable(headers, rows) {
      if (!rows.length) return '<div class="muted">暂无数据</div>';
      return '<table><thead><tr>' +
        headers.map(h => '<th>' + h + '</th>').join('') +
        '</tr></thead><tbody>' +
        rows.map(row => '<tr>' + row.map(col => '<td>' + col + '</td>').join('') + '</tr>').join('') +
        '</tbody></table>';
    }

    async function loadStats() {
      const headers = authHeaders();
      if (!headers.Authorization) {
        setStatus('未登录');
        return;
      }
      const res = await fetch('/api/analytics/stats?limit=100', { headers });
      const result = await res.json().catch(() => ({}));
      if (!res.ok || !result.success) {
        setStatus(result.message || '加载失败');
        return;
      }
      setStatus('已登录');
      const stats = result.data || {};
      document.getElementById('total-views').textContent = stats.total_views ?? '-';
      document.getElementById('total-pages').textContent = stats.total_pages ?? '-';
      document.getElementById('unique-ips').textContent = stats.unique_ips ?? '-';
      document.getElementById('unique-sessions').textContent = stats.unique_sessions ?? '-';

      document.getElementById('pages-box').innerHTML = renderTable(
        ['页面', '访问', '独立 IP', '最近访问'],
        (stats.pages || []).map(item => [
          '<span class="mono">' + (item.path || '-') + '</span><br>' + (item.title || '-'),
          String(item.views || 0),
          String(item.uv || 0),
          item.last_seen || '-'
        ])
      );

      document.getElementById('visitors-box').innerHTML = renderTable(
        ['IP', '访问', '地区', '设备', '最近访问'],
        (stats.visitors || []).map(item => [
          '<span class="mono">' + (item.ip || '-') + '</span>',
          String(item.visit_count || 0),
          item.region || '-',
          item.device || '-',
          item.last_seen || '-'
        ])
      );

      document.getElementById('recent-box').innerHTML = renderTable(
        ['时间', '页面', 'IP', '设备', '地区', 'DNS', 'WebRTC'],
        (stats.recent_visits || []).map(item => [
          item.created_at || '-',
          '<span class="mono">' + (item.path || '-') + '</span><br>' + (item.title || '-'),
          '<span class="mono">' + (item.ip || '-') + '</span>',
          [item.device || '-', item.browser || '-', item.os || '-'].join(' / '),
          [item.country || '', item.region || '', item.city || ''].filter(Boolean).join(' / ') || '-',
          item.dns_host || '-',
          item.webrtc ? JSON.stringify(item.webrtc) : '-'
        ])
      );
    }

    setStatus((localStorage.getItem('ws_token') || localStorage.getItem('auth_token')) ? '已登录' : '未登录');
    if (localStorage.getItem('ws_token') || localStorage.getItem('auth_token')) {
      loadStats();
    }
  </script>
</body>
</html>`
	return strings.ReplaceAll(page, "\t", "  ")
}
