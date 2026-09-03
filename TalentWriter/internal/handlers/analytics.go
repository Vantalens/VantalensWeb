package handlers

import (
	_ "embed"
	"net/http"
	"strconv"
	"strings"

	"vantalens/talentwriter/internal/analytics"
	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/models"
)

// 世界地图底图（Natural Earth 1:50m，CC0 1.0 公有领域）内嵌进二进制，
// 保证本地直连后端（127.0.0.1:9090）与线上反代都能访问 /vendor/ 资源。
//
//go:embed assets/world-map-equirectangular.svg
var worldMapSVG []byte

//go:embed assets/world-map-equirectangular.LICENSE.txt
var worldMapLicense []byte

// HandleVendorWorldMap 提供等距圆柱投影世界地图 SVG 底图（强缓存，内容长期不变）。
func HandleVendorWorldMap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(worldMapSVG)
}

// HandleVendorWorldMapLicense 提供底图来源与许可说明（CC0 1.0 署名文件）。
func HandleVendorWorldMapLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(worldMapLicense)
}

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

	if remoteAdminConfigured() {
		remotePath := "/api/admin/analytics/stats?limit=" + strconv.Itoa(limit)
		result, err := proxyRemoteAdmin(r, http.MethodGet, remotePath, nil)
		if err == nil {
			result.Response.Message = "已从服务器权威后端实时读取访问统计"
			RespondJSON(w, http.StatusOK, result.Response)
			return
		}
		if !localCacheEnabled() {
			RespondJSON(w, http.StatusBadGateway, models.APIResponse{
				Success: false,
				Message: "服务器权威后端不可达，且本地缓存兜底已关闭",
				Data:    remoteErrorData(result, err, "disabled"),
			})
			return
		}
		stats, localErr := analytics.GetSiteStatistics(limit)
		if localErr != nil {
			RespondJSON(w, http.StatusBadGateway, models.APIResponse{
				Success: false,
				Message: "服务器权威后端不可达，本地缓存也读取失败: " + localErr.Error(),
				Data:    remoteErrorData(result, err, "failed"),
			})
			return
		}
		RespondJSON(w, http.StatusOK, models.APIResponse{
			Success: true,
			Message: "服务器权威后端不可达，当前显示本地缓存数据: " + err.Error(),
			Data:    stats,
		})
		return
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
      color-scheme: light;
      --bg-a: #fffaf2;
      --bg-b: #fff1f7;
      --bg-c: #eefcf7;
      --panel: rgba(255,255,255,0.74);
      --card: rgba(250,249,245,0.78);
      --line: rgba(217,190,160,0.42);
      --text: #392833;
      --muted: rgba(82,58,71,0.68);
      --accent: #d46a92;
      --gold: #c79545;
      --teal: #4f9e8f;
      --danger: #c24c5f;
      --shadow: 0 18px 45px rgba(161,120,89,0.16);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      color: var(--text);
      background: linear-gradient(135deg, var(--bg-a) 0%, var(--bg-b) 52%, var(--bg-c) 100%);
    }
    .shell { max-width: 1440px; margin: 0 auto; padding: 24px; }
    .topbar, .panel {
      border: 1px solid var(--line);
      border-radius: 12px;
      background: var(--panel);
      box-shadow: var(--shadow);
      backdrop-filter: blur(16px);
    }
    .topbar {
      display: flex; justify-content: space-between; gap: 16px; align-items: center;
      padding: 18px 20px; margin-bottom: 18px;
    }
    .topbar h1 { margin: 0; font-size: 24px; }
    .topbar p { margin: 6px 0 0; color: var(--muted); font-size: 13px; }
    .actions { display: flex; gap: 10px; flex-wrap: wrap; }
    .actions a.active { background: linear-gradient(135deg, rgba(212,106,146,0.24), rgba(79,158,143,0.16)); border-color: rgba(212,106,146,.36); }
    .btn, input {
      border: 1px solid var(--line);
      border-radius: 10px;
      background: rgba(255,255,255,0.72);
      color: var(--text);
      padding: 11px 14px;
      font: inherit;
    }
    .btn { cursor: pointer; }
    .btn.primary { background: linear-gradient(135deg, rgba(212,106,146,0.24), rgba(79,158,143,0.16)); }
    .btn:disabled { opacity: .58; cursor: not-allowed; }
    .grid { display: grid; gap: 18px; }
    .stats { grid-template-columns: repeat(4, minmax(0, 1fr)); margin-bottom: 18px; }
    .card { padding: 16px; border-radius: 10px; background: var(--card); border: 1px solid var(--line); }
    .label { color: var(--muted); font-size: 12px; margin-bottom: 8px; }
    .value { font-size: 28px; font-weight: 700; }
    .panel { padding: 18px; margin-bottom: 18px; }
    .panel h2 { margin: 0 0 14px; font-size: 18px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 8px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-weight: 600; font-size: 12px; }
    td { font-size: 13px; }
    .mono { font-family: "Cascadia Mono", "Consolas", monospace; font-size: 12px; overflow-wrap:anywhere; word-break:break-word; }
    .table-scroll { width:100%; max-width:100%; overflow-x:auto; }
    .table-scroll table { min-width:640px; }
    .map-base { opacity: .9; }
    .map-ocean { fill: #e2f1ed; }
    .map-land path, .map-land polygon { fill: #eddfc4; stroke: #d3bd95; stroke-width: 6; stroke-linejoin: round; }
    .graticule { stroke: rgba(199,149,69,.26); stroke-width: 1; stroke-dasharray: 4 5; }
    .graticule-major { stroke: rgba(79,158,143,.52); stroke-width: 1.5; stroke-dasharray: none; }
    .marker { cursor: pointer; }
    .marker .marker-halo { fill: rgba(212,106,146,.10); }
    .marker .marker-main { fill: rgba(212,106,146,.42); stroke: #d46a92; stroke-width: 1.6; transition: fill .15s ease; }
    .marker:hover .marker-main, .marker.selected .marker-main { fill: rgba(212,106,146,.64); }
    .marker.selected .marker-main { stroke: var(--gold); stroke-width: 2.8; }
    .marker:focus-visible .marker-main { stroke: #236d62; stroke-width: 3; }
    .marker .marker-dot { fill: #d46a92; }
    .marker-label { font-size: 11px; fill: #392833; paint-order: stroke; stroke: rgba(255,255,255,.82); stroke-width: 3px; stroke-linejoin: round; }
    .map-legend text { font-size: 11px; fill: rgba(82,58,71,.78); }
    .map-legend circle { fill: rgba(212,106,146,.25); stroke: #d46a92; stroke-width: 1.2; }
    .map-legend .legend-title { font-weight: 600; }
    .map-tooltip { position: absolute; pointer-events: none; background: rgba(255,255,255,.95); border: 1px solid var(--line); border-radius: 8px; padding: 8px 10px; font-size: 12px; line-height: 1.5; box-shadow: var(--shadow); z-index: 5; max-width: 230px; }
    .map-credit { position: absolute; left: 10px; bottom: 8px; font-size: 11px; color: var(--muted); background: rgba(255,250,242,.85); border-radius: 6px; padding: 3px 8px; }
    .map-credit a { color: var(--teal); }
    .statusbar { display:flex; gap:10px; flex-wrap:wrap; align-items:center; margin-bottom:18px; }
    .quick-login { margin-left:auto; display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
    .quick-login input { width:auto; min-width:132px; padding:8px 10px; border-radius:999px; }
    .quick-login .btn { padding:8px 11px; border-radius:999px; }
    .hidden { display:none!important; }
    .muted { color: var(--muted); }
    .split { display: grid; gap: 18px; grid-template-columns: 1.2fr 1fr; }
    .map-wrap { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(280px, .9fr); gap: 16px; align-items: stretch; }
    .map-card { position: relative; min-height: 420px; border: 1px solid var(--line); border-radius: 10px; background: rgba(255,255,255,.62); overflow: hidden; }
    .map-card svg { width: 100%; height: 100%; min-height: 420px; display: block; }
    .map-detail { border:1px solid var(--line); border-radius:10px; background:rgba(255,255,255,.58); padding:12px; margin-bottom:12px; }
    details.fold { border: 1px solid var(--line); border-radius: 10px; background: rgba(255,255,255,.54); margin-bottom: 10px; overflow: hidden; }
    details.fold summary { cursor: pointer; padding: 12px 14px; display: flex; justify-content: space-between; gap: 10px; align-items: center; }
    .fold-body { padding: 0 14px 14px; display: grid; gap: 8px; }
    .mini-row { display: grid; grid-template-columns: 1.1fr .7fr .8fr; gap: 10px; align-items: start; border-top: 1px solid var(--line); padding-top: 8px; font-size: 13px; }
    .tag { display: inline-flex; align-items: center; border: 1px solid var(--line); border-radius: 999px; padding: 3px 8px; background: rgba(255,255,255,.62); color: var(--muted); font-size: 12px; }
    .status-ok { color: var(--teal); }
    .status-error { color: var(--danger); }
    @media (max-width: 1100px) { .stats, .split { grid-template-columns: 1fr 1fr; } }
    @media (max-width: 980px) { .map-wrap { grid-template-columns: 1fr; } }
    @media (max-width: 760px) { .topbar, .stats, .split { grid-template-columns: 1fr; display: grid; } .quick-login, .quick-login input, .quick-login .btn { width:100%; margin-left:0; } .mini-row { grid-template-columns: 1fr; } }
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
        <a class="btn" href="/platform">入口</a>
        <a class="btn" href="/platform/posts">文章</a>
        <a class="btn" href="/platform/comments">评论</a>
        <a class="btn active" href="/platform/analytics">监控</a>
        <button id="refresh-btn" class="btn primary" onclick="loadStats(this)">刷新</button>
      </div>
    </header>

    <section class="statusbar">
      <span id="auth-status" class="tag">未登录</span>
      <span id="data-source" class="tag">数据源未检查</span>
      <span id="time-status" class="tag">北京时间 --</span>
      <div id="quick-login" class="quick-login hidden">
        <input id="login-user" value="vantalens" placeholder="用户名">
        <input id="login-pass" type="password" placeholder="后台密码">
        <button class="btn primary" onclick="login(this)">登录</button>
      </div>
      <button id="logout-btn" class="btn hidden" onclick="logout()">退出</button>
    </section>

    <section class="grid stats">
      <div class="card"><div class="label">总访问量</div><div id="total-views" class="value">-</div></div>
      <div class="card"><div class="label">总分页数</div><div id="total-pages" class="value">-</div></div>
      <div class="card"><div class="label">独立 IP</div><div id="unique-ips" class="value">-</div></div>
      <div class="card"><div class="label">独立会话</div><div id="unique-sessions" class="value">-</div></div>
    </section>

    <section class="panel">
      <h2>地区分布</h2>
      <div class="map-wrap">
        <div id="map-box" class="map-card"></div>
        <div>
          <div id="map-detail" class="map-detail muted">点击地图上的暖色气泡查看详情。</div>
          <div id="regions-box" class="muted">暂无数据</div>
        </div>
      </div>
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
      const el = document.getElementById('auth-status');
      el.textContent = text;
      el.className = 'tag ' + (text.includes('失败') || text.includes('失效') || text.includes('未登录') ? 'status-error' : 'status-ok');
      const ok = !!(localStorage.getItem('ws_token') || localStorage.getItem('auth_token'));
      document.getElementById('quick-login')?.classList.toggle('hidden', ok);
      document.getElementById('logout-btn')?.classList.toggle('hidden', !ok);
    }

    function formatBeijingTime(value) {
      if (!value) return '-';
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) return String(value);
      return new Intl.DateTimeFormat('zh-CN', {
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      }).format(date).replaceAll('/', '-');
    }

    function refreshBeijingClock() {
      const el = document.getElementById('time-status');
      if (el) el.textContent = '北京时间 ' + formatBeijingTime(new Date().toISOString());
    }

    async function tryPlatformSession() {
      try {
        const res = await fetch('/platform/session', { cache: 'no-store', credentials: 'same-origin' });
        const data = await res.json().catch(() => ({}));
        const token = data?.data?.access_token || data?.data?.token;
        if (res.ok && data.success && token) {
          localStorage.setItem('ws_token', token);
          localStorage.setItem('auth_token', token);
          setStatus('已登录');
          return true;
        }
      } catch (e) {}
      return false;
    }

    function escapeHtml(input) {
      return String(input || '')
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
    }

    function setButton(button, busy, label) {
      if (!button) return;
      if (!button.dataset.originalText) button.dataset.originalText = button.textContent;
      button.disabled = !!busy;
      button.textContent = busy ? label : button.dataset.originalText;
    }

    async function login(button) {
      setButton(button, true, '登录中...');
      const username = document.getElementById('login-user').value.trim() || 'vantalens';
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
        setButton(button, false);
        return;
      }
      localStorage.setItem('ws_token', token);
      localStorage.setItem('auth_token', token);
      setStatus('已登录');
      setButton(button, false);
      await loadStats();
    }

    function logout() {
      localStorage.removeItem('ws_token');
      localStorage.removeItem('auth_token');
      setStatus('已退出');
    }

    function renderTable(headers, rows) {
      if (!rows.length) return '<div class="muted">暂无数据</div>';
      return '<div class="table-scroll"><table><thead><tr>' +
        headers.map(h => '<th>' + escapeHtml(h) + '</th>').join('') +
        '</tr></thead><tbody>' +
        rows.map(row => '<tr>' + row.map(col => '<td>' + col + '</td>').join('') + '</tr>').join('') +
        '</tbody></table></div>';
    }

    function regionLabel(item) {
      return item.label || [item.country, item.region, item.city].filter(Boolean).join(' / ') || '未知地区';
    }

    function normalizeLocationName(value) {
      const key = String(value || '').trim().toLowerCase();
      const aliases = {
        '中华人民共和国': 'china', '中国': 'china',
        '美国': 'united states', '美利坚合众国': 'united states', 'us': 'united states', 'usa': 'united states',
        '英国': 'united kingdom', '俄罗斯': 'russia', '俄国': 'russia',
        '韩国': 'south korea', '日本': 'japan', '新加坡': 'singapore',
        '德国': 'germany', '法国': 'france', '澳大利亚': 'australia',
        '加拿大': 'canada', '巴西': 'brazil', '印度': 'india',
        '荷兰': 'netherlands', '意大利': 'italy', '西班牙': 'spain',
        '香港': 'hong kong', '中国香港': 'hong kong', '澳门': 'macau', '台湾': 'taiwan',
        '马来西亚': 'malaysia', '泰国': 'thailand', '越南': 'vietnam',
        '印度尼西亚': 'indonesia', '印尼': 'indonesia', '菲律宾': 'philippines',
        '墨西哥': 'mexico', '阿根廷': 'argentina', '智利': 'chile', '哥伦比亚': 'colombia', '秘鲁': 'peru',
        '埃及': 'egypt', '南非': 'south africa', '尼日利亚': 'nigeria', '肯尼亚': 'kenya',
        '土耳其': 'turkey', '阿联酋': 'united arab emirates', '阿拉伯联合酋长国': 'united arab emirates',
        '沙特阿拉伯': 'saudi arabia', '沙特': 'saudi arabia', '以色列': 'israel',
        '巴基斯坦': 'pakistan', '孟加拉国': 'bangladesh', '哈萨克斯坦': 'kazakhstan', '乌克兰': 'ukraine',
        '波兰': 'poland', '瑞典': 'sweden', '挪威': 'norway', '芬兰': 'finland', '丹麦': 'denmark',
        '瑞士': 'switzerland', '奥地利': 'austria', '比利时': 'belgium', '葡萄牙': 'portugal',
        '爱尔兰': 'ireland', '新西兰': 'new zealand', '希腊': 'greece', '捷克': 'czechia',
        '匈牙利': 'hungary', '罗马尼亚': 'romania',
        '北京': 'beijing', '北京市': 'beijing', '上海': 'shanghai', '上海市': 'shanghai',
        '天津': 'tianjin', '天津市': 'tianjin', '重庆': 'chongqing', '重庆市': 'chongqing',
        '广东': 'guangdong', '广东省': 'guangdong', '浙江': 'zhejiang', '浙江省': 'zhejiang',
        '江苏': 'jiangsu', '江苏省': 'jiangsu', '山东': 'shandong', '山东省': 'shandong',
        '四川': 'sichuan', '四川省': 'sichuan', '湖北': 'hubei', '湖北省': 'hubei',
        '湖南': 'hunan', '湖南省': 'hunan', '河南': 'henan', '河南省': 'henan',
        '河北': 'hebei', '河北省': 'hebei', '福建': 'fujian', '福建省': 'fujian',
        '安徽': 'anhui', '安徽省': 'anhui', '江西': 'jiangxi', '江西省': 'jiangxi',
        '辽宁': 'liaoning', '辽宁省': 'liaoning', '吉林': 'jilin', '吉林省': 'jilin',
        '黑龙江': 'heilongjiang', '黑龙江省': 'heilongjiang', '山西': 'shanxi', '山西省': 'shanxi',
        '陕西': 'shaanxi', '陕西省': 'shaanxi', '甘肃': 'gansu', '甘肃省': 'gansu',
        '青海': 'qinghai', '青海省': 'qinghai', '宁夏': 'ningxia', '宁夏回族自治区': 'ningxia',
        '新疆': 'xinjiang', '新疆维吾尔自治区': 'xinjiang', '西藏': 'tibet', '西藏自治区': 'tibet',
        '云南': 'yunnan', '云南省': 'yunnan', '贵州': 'guizhou', '贵州省': 'guizhou',
        '广西': 'guangxi', '广西壮族自治区': 'guangxi', '海南': 'hainan', '海南省': 'hainan',
        '内蒙古': 'inner mongolia', '内蒙古自治区': 'inner mongolia',
        '成都': 'chengdu', '深圳': 'shenzhen', '广州': 'guangzhou', '杭州': 'hangzhou',
        '南京': 'nanjing', '苏州': 'suzhou', '武汉': 'wuhan', '西安': "xi'an",
        '青岛': 'qingdao', '长沙': 'changsha', '郑州': 'zhengzhou', '厦门': 'xiamen',
        '福州': 'fuzhou', '昆明': 'kunming', '合肥': 'hefei', '济南': 'jinan',
        '大连': 'dalian', '沈阳': 'shenyang', '哈尔滨': 'harbin', '台北': 'taipei'
      };
      return aliases[key] || key;
    }

    function resolveMapPoint(item) {
      const latitude = Number(item.latitude);
      const longitude = Number(item.longitude);
      if (Number.isFinite(latitude) && Number.isFinite(longitude) && latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180 && (latitude !== 0 || longitude !== 0)) {
        return { lat: latitude, lon: longitude };
      }
      const names = [item.country, item.region, item.city].map(normalizeLocationName);
      const cityKey = names.filter(Boolean).join('|');
      const regionKey = names.slice(0, 2).filter(Boolean).join('|');
      const countryKey = names[0] || '';
      const cityPoints = {
        'united states|california|san francisco': [37.7749, -122.4194],
        'united states|california|los angeles': [34.0522, -118.2437],
        'united states|new york|new york': [40.7128, -74.006],
        'china|beijing|beijing': [39.9042, 116.4074],
        'china|shanghai|shanghai': [31.2304, 121.4737],
        'china|tianjin|tianjin': [39.3434, 117.3616],
        'china|chongqing|chongqing': [29.563, 106.5516],
        'china|sichuan|chengdu': [30.5728, 104.0668],
        'china|guangdong|shenzhen': [22.5431, 114.0579],
        'china|guangdong|guangzhou': [23.1291, 113.2644],
        'china|zhejiang|hangzhou': [30.2741, 120.1551],
        'china|jiangsu|nanjing': [32.0603, 118.7969],
        'china|jiangsu|suzhou': [31.2989, 120.5853],
        'china|hubei|wuhan': [30.5928, 114.3055],
        "china|shaanxi|xi'an": [34.3416, 108.9398],
        'china|shandong|qingdao': [36.0671, 120.3826],
        'china|shandong|jinan': [36.6512, 117.1201],
        'china|hunan|changsha': [28.2282, 112.9388],
        'china|henan|zhengzhou': [34.7466, 113.6254],
        'china|fujian|xiamen': [24.4798, 118.0894],
        'china|fujian|fuzhou': [26.0745, 119.2965],
        'china|yunnan|kunming': [24.8801, 102.8329],
        'china|anhui|hefei': [31.8206, 117.2272],
        'china|liaoning|dalian': [38.914, 121.6147],
        'china|liaoning|shenyang': [41.8057, 123.4315],
        'china|heilongjiang|harbin': [45.8038, 126.5349],
        'china|hong kong|hong kong': [22.3193, 114.1694],
        'china|macau|macau': [22.1987, 113.5439],
        'china|taiwan|taipei': [25.033, 121.5654],
        'japan|tokyo|tokyo': [35.6762, 139.6503],
        'japan|osaka|osaka': [34.6937, 135.5023],
        'south korea|seoul|seoul': [37.5665, 126.978],
        'singapore|singapore|singapore': [1.3521, 103.8198],
        'united kingdom|england|london': [51.5072, -0.1276],
        'france|ile-de-france|paris': [48.8566, 2.3522],
        'germany|berlin|berlin': [52.52, 13.405],
        'australia|new south wales|sydney': [-33.8688, 151.2093],
        'canada|ontario|toronto': [43.6532, -79.3832],
        'brazil|sao paulo|sao paulo': [-23.5558, -46.6396],
        'india|maharashtra|mumbai': [19.076, 72.8777]
      };
      const regionPoints = {
        'united states|california': [36.7783, -119.4179],
        'united states|new york': [43.0, -75.0],
        'china|beijing': [39.9042, 116.4074],
        'china|shanghai': [31.2304, 121.4737],
        'china|tianjin': [39.3434, 117.3616],
        'china|chongqing': [29.563, 106.5516],
        'china|guangdong': [23.379, 113.7633],
        'china|zhejiang': [29.1832, 120.0934],
        'china|jiangsu': [32.0603, 118.7969],
        'china|shandong': [36.6512, 117.1201],
        'china|sichuan': [30.5728, 104.0668],
        'china|hubei': [30.5928, 114.3055],
        'china|hunan': [28.2282, 112.9388],
        'china|henan': [34.7466, 113.6254],
        'china|hebei': [38.0428, 114.5149],
        'china|fujian': [26.0745, 119.2965],
        'china|anhui': [31.8206, 117.2272],
        'china|jiangxi': [28.682, 115.8582],
        'china|liaoning': [41.8057, 123.4315],
        'china|jilin': [43.8171, 125.3235],
        'china|heilongjiang': [45.8038, 126.5349],
        'china|shanxi': [37.8706, 112.5489],
        'china|shaanxi': [34.3416, 108.9398],
        'china|gansu': [36.0611, 103.8343],
        'china|qinghai': [36.6171, 101.7782],
        'china|ningxia': [38.4872, 106.2309],
        'china|xinjiang': [43.8256, 87.6168],
        'china|tibet': [29.652, 91.1721],
        'china|yunnan': [24.8801, 102.8329],
        'china|guizhou': [26.647, 106.6302],
        'china|guangxi': [22.817, 108.3669],
        'china|hainan': [20.044, 110.1989],
        'china|inner mongolia': [40.8426, 111.7492],
        'china|hong kong': [22.3193, 114.1694],
        'china|macau': [22.1987, 113.5439],
        'china|taiwan': [25.033, 121.5654],
        'japan|tokyo': [35.6762, 139.6503],
        'united kingdom|england': [52.3555, -1.1743],
        'canada|ontario': [50.0, -85.0],
        'australia|new south wales': [-31.2532, 146.9211]
      };
      const countryPoints = {
        'united states': [39.8283, -98.5795],
        'usa': [39.8283, -98.5795],
        'china': [35.8617, 104.1954],
        'hong kong': [22.3193, 114.1694],
        'macau': [22.1987, 113.5439],
        'taiwan': [25.033, 121.5654],
        'japan': [36.2048, 138.2529],
        'singapore': [1.3521, 103.8198],
        'united kingdom': [55.3781, -3.436],
        'france': [46.2276, 2.2137],
        'germany': [51.1657, 10.4515],
        'australia': [-25.2744, 133.7751],
        'canada': [56.1304, -106.3468],
        'brazil': [-14.235, -51.9253],
        'india': [20.5937, 78.9629],
        'russia': [61.524, 105.3188],
        'south korea': [35.9078, 127.7669],
        'netherlands': [52.1326, 5.2913],
        'italy': [41.8719, 12.5674],
        'spain': [40.4637, -3.7492],
        'malaysia': [4.2105, 101.9758],
        'thailand': [15.87, 100.9925],
        'vietnam': [14.0583, 108.2772],
        'indonesia': [-0.7893, 113.9213],
        'philippines': [12.8797, 121.774],
        'mexico': [23.6345, -102.5528],
        'argentina': [-38.4161, -63.6167],
        'chile': [-35.6751, -71.543],
        'colombia': [4.5709, -74.2973],
        'peru': [-9.19, -75.0152],
        'egypt': [26.8206, 30.8025],
        'south africa': [-30.5595, 22.9375],
        'nigeria': [9.082, 8.6753],
        'kenya': [-0.0236, 37.9062],
        'turkey': [38.9637, 35.2433],
        'united arab emirates': [23.4241, 53.8478],
        'uae': [23.4241, 53.8478],
        'saudi arabia': [23.8859, 45.0792],
        'israel': [31.0461, 34.8516],
        'pakistan': [30.3753, 69.3451],
        'bangladesh': [23.685, 90.3563],
        'kazakhstan': [48.0196, 66.9237],
        'ukraine': [48.3794, 31.1656],
        'poland': [51.9194, 19.1451],
        'sweden': [60.1282, 18.6435],
        'norway': [60.472, 8.4689],
        'finland': [61.9241, 25.7482],
        'denmark': [56.2639, 9.5018],
        'switzerland': [46.8182, 8.2275],
        'austria': [47.5162, 14.5501],
        'belgium': [50.5039, 4.4699],
        'portugal': [39.3999, -8.2245],
        'ireland': [53.4129, -8.2439],
        'new zealand': [-40.9006, 174.886],
        'greece': [39.0742, 21.8243],
        'czechia': [49.8175, 15.473],
        'czech republic': [49.8175, 15.473],
        'hungary': [47.1625, 19.5033],
        'romania': [45.9432, 24.9668]
      };
      const point = cityPoints[cityKey] || regionPoints[regionKey] || countryPoints[countryKey];
      return point ? { lat: point[0], lon: point[1] } : null;
    }

    // 等距圆柱投影：底图 viewBox 0 0 5760 2880 与本视口 960×480 同为 2:1，线性映射即可对齐。
    const MAP_W = 960, MAP_H = 480;
    function projectPoint(lat, lon) {
      return {
        x: ((lon + 180) / 360) * MAP_W,
        y: ((90 - lat) / 180) * MAP_H
      };
    }

    function bubbleRadius(views, max) {
      return 4 + Math.round(Math.sqrt((views || 0) / Math.max(1, max)) * 14);
    }

    let analyticsState = {};
    let mapRegions = [];

    // 内联底图：拉取 /vendor/ SVG 后按比例缩放进地图，可受 CSS 控制陆地上色；
    // 失败时退化为 <image> 按投影比例引用（viewBox 与视口同为 2:1，不拉伸）。
    async function renderWorldBase() {
      const layer = document.getElementById('land-layer');
      if (!layer) return;
      const url = '/vendor/world-map-equirectangular.svg';
      try {
        const res = await fetch(url, { cache: 'force-cache' });
        if (!res.ok) throw new Error('HTTP ' + res.status);
        const text = await res.text();
        const doc = new DOMParser().parseFromString(text, 'image/svg+xml');
        const source = doc.querySelector('svg');
        if (!source || !source.children.length) throw new Error('empty svg');
        const g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
        g.setAttribute('class', 'map-land');
        g.setAttribute('transform', 'scale(' + (MAP_W / 5760) + ')');
        Array.from(source.children).forEach(node => {
          const clone = document.importNode(node, true);
          if (clone.nodeType === 1) {
            clone.removeAttribute('fill');
            clone.removeAttribute('stroke');
          }
          g.appendChild(clone);
        });
        layer.appendChild(g);
      } catch (e) {
        const img = document.createElementNS('http://www.w3.org/2000/svg', 'image');
        img.setAttribute('class', 'map-base');
        img.setAttribute('href', url);
        img.setAttribute('x', '0');
        img.setAttribute('y', '0');
        img.setAttribute('width', String(MAP_W));
        img.setAttribute('height', String(MAP_H));
        layer.appendChild(img);
      }
    }

    function renderGraticule() {
      let out = '';
      for (let lon = -150; lon <= 150; lon += 30) {
        const x = projectPoint(0, lon).x.toFixed(1);
        out += '<line class="graticule' + (lon === 0 ? ' graticule-major' : '') + '" x1="' + x + '" y1="0" x2="' + x + '" y2="' + MAP_H + '"></line>';
      }
      for (let lat = -60; lat <= 60; lat += 30) {
        const y = projectPoint(lat, 0).y.toFixed(1);
        out += '<line class="graticule' + (lat === 0 ? ' graticule-major' : '') + '" x1="0" y1="' + y + '" x2="' + MAP_W + '" y2="' + y + '"></line>';
      }
      return out;
    }

    function renderMapLegend(max) {
      const samples = [max, Math.max(1, Math.round(max / 4)), 1];
      let out = '<g class="map-legend"><text class="legend-title" x="' + (MAP_W - 118) + '" y="26">气泡大小 = 访问量</text>';
      samples.forEach((v, i) => {
        const cy = 52 + i * 34;
        const r = bubbleRadius(v, max);
        out += '<circle cx="' + (MAP_W - 44) + '" cy="' + cy + '" r="' + r + '"></circle>' +
          '<text x="' + (MAP_W - 118) + '" y="' + (cy + 4) + '">' + v + ' 次访问</text>';
      });
      return out + '</g>';
    }

    function showMapTooltip(event, index) {
      const item = mapRegions[index];
      const tip = document.getElementById('map-tooltip');
      const box = document.getElementById('map-box');
      if (!item || !tip || !box) return;
      tip.innerHTML = '<strong>' + escapeHtml(regionLabel(item)) + '</strong><br>访问量 ' + (item.views || 0) + ' 次 · 独立 IP ' + (item.unique_ips || 0) + ' 个';
      tip.classList.remove('hidden');
      const rect = box.getBoundingClientRect();
      const x = Math.min(event.clientX - rect.left + 14, Math.max(8, rect.width - 240));
      const y = Math.min(event.clientY - rect.top + 14, Math.max(8, rect.height - 70));
      tip.style.left = x + 'px';
      tip.style.top = y + 'px';
    }

    function hideMapTooltip() {
      const tip = document.getElementById('map-tooltip');
      if (tip) tip.classList.add('hidden');
    }

    function selectRegion(index) {
      const item = mapRegions[index];
      if (!item) return;
      const point = resolveMapPoint(item);
      const hasCoords = Number.isFinite(Number(item.latitude)) && Number.isFinite(Number(item.longitude));
      const detail = document.getElementById('map-detail');
      if (detail) {
        detail.classList.remove('muted');
        detail.innerHTML =
          '<div><strong>' + escapeHtml(regionLabel(item)) + '</strong></div>' +
          '<div class="muted" style="margin-top:6px">访问量 ' + (item.views || 0) + ' 次 · 独立 IP ' + (item.unique_ips || 0) + ' 个</div>' +
          '<div class="muted">代表 IP：<span class="mono">' + escapeHtml(item.representative_ip || '-') + '</span></div>' +
          '<div class="muted">最近访问：' + escapeHtml(formatBeijingTime(item.last_seen)) + '</div>' +
          (point ? '<div class="muted">地图定位：' + point.lat.toFixed(2) + ', ' + point.lon.toFixed(2) + '（' + (hasCoords ? '数据自带坐标' : '内置坐标表近似位置') + '）</div>' : '');
      }
      document.querySelectorAll('#map-box .marker').forEach(el => {
        el.classList.toggle('selected', el.dataset.index === String(index));
      });
    }

    function renderMap(regions) {
      mapRegions = regions.filter(item => resolveMapPoint(item));
      mapRegions.sort((a, b) => (b.views || 0) - (a.views || 0));
      const unknown = regions.length - mapRegions.length;
      const max = Math.max(1, ...mapRegions.map(item => item.views || 0));
      const markers = mapRegions.map((item, index) => {
        const point = resolveMapPoint(item);
        const pos = projectPoint(point.lat, point.lon);
        const r = bubbleRadius(item.views, max);
        const label = escapeHtml(regionLabel(item)).slice(0, 22);
        return '<g class="marker" tabindex="0" role="button" data-index="' + index + '" onclick="selectRegion(' + index + ')" onmousemove="showMapTooltip(event,' + index + ')" onmouseleave="hideMapTooltip()" onkeydown="if(event.keyCode===13||event.keyCode===32){event.preventDefault();selectRegion(' + index + ')}" transform="translate(' + pos.x.toFixed(1) + ' ' + pos.y.toFixed(1) + ')">' +
          '<circle class="marker-halo" r="' + (r + 9) + '"></circle>' +
          '<circle class="marker-main" r="' + r + '"></circle>' +
          '<circle class="marker-dot" r="2.4"></circle>' +
          (index < 12 ? '<text class="marker-label" x="' + (r + 7) + '" y="4">' + label + ' · ' + (item.views || 0) + '</text>' : '') +
          '<title>' + escapeHtml(regionLabel(item)) + '｜访问 ' + (item.views || 0) + '｜独立 IP ' + (item.unique_ips || 0) + '</title>' +
          '</g>';
      }).join('');
      document.getElementById('map-box').innerHTML =
        '<svg viewBox="0 0 ' + MAP_W + ' ' + MAP_H + '" role="img" aria-label="世界地图访问来源分布">' +
        '<rect class="map-ocean" width="' + MAP_W + '" height="' + MAP_H + '"></rect>' +
        '<g id="land-layer"></g>' +
        renderGraticule() +
        markers +
        renderMapLegend(max) +
        '</svg>' +
        '<div id="map-tooltip" class="map-tooltip hidden"></div>' +
        '<div class="map-credit">底图：<a href="/vendor/world-map-equirectangular.LICENSE.txt" target="_blank" rel="noopener">Natural Earth 1:50m · CC0 1.0 公有领域</a>，等距圆柱投影；暖色气泡为可定位访问来源，未定位 ' + unknown + ' 组（见下方折叠列表）。</div>';
      renderWorldBase();
      const detail = document.getElementById('map-detail');
      if (detail) detail.innerHTML = mapRegions.length ? '<div class="muted">点击地图上的暖色气泡查看地区详情，悬停可快速预览。</div>' : '<div class="muted">暂无可定位地区，数据已保留在下方折叠列表。</div>';
    }

    function renderRegionFolds(stats) {
      const regions = stats.regions || [];
      if (!regions.length) return '<div class="muted">暂无地区数据</div>';
      const visitors = stats.visitors || [];
      return regions.map((region, index) => {
        const label = regionLabel(region);
        const group = visitors.filter(item => regionLabel(item) === label || (item.country || '') === (region.country || '') && (item.region || '') === (region.region || '') && (item.city || '') === (region.city || ''));
        const rows = group.slice(0, 10).map(item =>
          '<div class="mini-row"><span class="mono">' + escapeHtml(item.ip || '-') + '</span><span>' + (item.visit_count || 0) + ' 次</span><span>' + escapeHtml(formatBeijingTime(item.last_seen)) + '</span></div>'
        ).join('') || '<div class="muted">暂无该地区 IP 明细</div>';
        return '<details class="fold" ' + (index < 3 ? 'open' : '') + '><summary><strong>' + escapeHtml(label) + '</strong><span><span class="tag">' + (region.views || 0) + ' 次访问</span> <span class="tag">' + (region.unique_ips || 0) + ' 个 IP</span></span></summary><div class="fold-body">' +
          '<div class="muted">代表 IP：<span class="mono">' + escapeHtml(region.representative_ip || '-') + '</span>，最近访问：' + escapeHtml(formatBeijingTime(region.last_seen)) + '</div>' +
          rows + '</div></details>';
      }).join('');
    }

    function groupBy(items, keyFn) {
      return items.reduce((acc, item) => {
        const key = keyFn(item) || '未知地区';
        if (!acc[key]) acc[key] = [];
        acc[key].push(item);
        return acc;
      }, {});
    }

    function renderVisitorFolds(visitors) {
      if (!visitors.length) return '<div class="muted">暂无数据</div>';
      const groups = groupBy(visitors, item => [item.country, item.region, item.city].filter(Boolean).join(' / ') || '未知地区');
      return Object.keys(groups).map((label, index) => {
        const rows = groups[label].slice(0, 10).map(item =>
          '<div class="mini-row"><span class="mono">' + escapeHtml(item.ip || '-') + '</span><span>' + (item.visit_count || 0) + ' 次 / ' + escapeHtml(item.device || '-') + '</span><span>' + escapeHtml(formatBeijingTime(item.last_seen)) + '</span></div>'
        ).join('');
        const more = groups[label].length > 10 ? '<div class="muted">已折叠 ' + (groups[label].length - 10) + ' 条更多 IP，可在 API 中按需查看完整数据。</div>' : '';
        return '<details class="fold" ' + (index < 3 ? 'open' : '') + '><summary><strong>' + escapeHtml(label) + '</strong><span class="tag">' + groups[label].length + ' 个 IP</span></summary><div class="fold-body">' + rows + more + '</div></details>';
      }).join('');
    }

    function renderRecentFolds(visits) {
      if (!visits.length) return '<div class="muted">暂无数据</div>';
      const groups = groupBy(visits, item => [item.country, item.region, item.city].filter(Boolean).join(' / ') || '未知地区');
      return Object.keys(groups).map((label, index) => {
        const rows = groups[label].slice(0, 10).map(item =>
          '<div class="mini-row"><span>' + escapeHtml(formatBeijingTime(item.created_at)) + '<br><span class="mono">' + escapeHtml(item.ip || '-') + '</span></span><span><span class="mono">' + escapeHtml(item.path || '-') + '</span><br>' + escapeHtml(item.title || '-') + '</span><span>' + escapeHtml([item.device || '-', item.browser || '-', item.os || '-'].join(' / ')) + '</span></div>'
        ).join('');
        const more = groups[label].length > 10 ? '<div class="muted">已折叠 ' + (groups[label].length - 10) + ' 条访问记录。</div>' : '';
        return '<details class="fold" ' + (index < 2 ? 'open' : '') + '><summary><strong>' + escapeHtml(label) + '</strong><span class="tag">' + groups[label].length + ' 条记录</span></summary><div class="fold-body">' + rows + more + '</div></details>';
      }).join('');
    }

    async function loadStats(button) {
      const headers = authHeaders();
      if (!headers.Authorization) {
        setStatus('未登录');
        return;
      }
      setButton(button, true, '刷新中...');
      setStatus('加载中...');
      const res = await fetch('/api/analytics/stats?limit=300', { headers });
      const result = await res.json().catch(() => ({}));
      if (!res.ok || !result.success) {
        setButton(button, false);
        setStatus(result.message || '加载失败');
        return;
      }
      setStatus('已登录');
      const source = document.getElementById('data-source');
      if (source) source.textContent = result.message || '已读取访问统计';
      setButton(button, false);
      const stats = result.data || {};
      analyticsState = stats;
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
          formatBeijingTime(item.last_seen)
        ])
      );

      renderMap(stats.regions || []);
      document.getElementById('regions-box').innerHTML = renderRegionFolds(stats);
      document.getElementById('visitors-box').innerHTML = renderVisitorFolds(stats.visitors || []);
      document.getElementById('recent-box').innerHTML = renderRecentFolds(stats.recent_visits || []);
    }

    async function bootstrap() {
      refreshBeijingClock();
      setInterval(refreshBeijingClock, 1000);
      if (!(localStorage.getItem('ws_token') || localStorage.getItem('auth_token'))) await tryPlatformSession();
      setStatus((localStorage.getItem('ws_token') || localStorage.getItem('auth_token')) ? '已登录' : '未登录');
      if (localStorage.getItem('ws_token') || localStorage.getItem('auth_token')) {
        await loadStats();
      }
    }
    bootstrap();
  </script>
</body>
</html>`
	return strings.ReplaceAll(page, "\t", "  ")
}
