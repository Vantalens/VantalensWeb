package handlers

import "strings"

func ControlHTML(writerURL string) string {
	page := `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Vantalens 总控平台</title>
  <style>
    :root {
      --bg-a: #eef7f1;
      --bg-b: #d8ebe0;
      --panel: rgba(255,255,255,0.78);
      --ink: #1f352b;
      --sub: #4d6a5e;
      --accent: #1f7a53;
      --line: rgba(31, 53, 43, 0.16);
      --warn: #9a6a10;
      --ok: #0f8c4a;
      --err: #a33131;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      color: var(--ink);
      background:
        radial-gradient(1200px 600px at -10% 10%, rgba(31,122,83,0.18), transparent 70%),
        radial-gradient(1200px 600px at 120% -10%, rgba(15,140,74,0.14), transparent 65%),
        linear-gradient(135deg, var(--bg-a), var(--bg-b));
      padding: 26px;
    }
    .shell { max-width: 1280px; margin: 0 auto; }
    .header {
      display: flex; justify-content: space-between; gap: 14px; align-items: center;
      margin-bottom: 16px;
    }
    .title { margin: 0; font-size: clamp(24px, 3vw, 38px); }
    .sub { margin: 6px 0 0; color: var(--sub); }
    .actions { display: flex; gap: 10px; flex-wrap: wrap; }
    .btn {
      border: 1px solid var(--line); background: rgba(255,255,255,0.8); color: var(--ink);
      border-radius: 12px; padding: 10px 14px; cursor: pointer; text-decoration: none;
      font-weight: 600;
    }
    .btn.primary { background: var(--accent); color: #fff; border-color: transparent; }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; align-items: start; }
    .panel {
      background: var(--panel); border: 1px solid var(--line); border-radius: 16px;
      padding: 16px; backdrop-filter: blur(8px);
    }
    .panel h2 { margin: 0 0 10px; font-size: 20px; }
    .panel h3 { margin: 12px 0 8px; font-size: 16px; }
    .row { display: flex; gap: 10px; flex-wrap: wrap; }
    .kpis { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-bottom: 8px; }
    .kpi { border: 1px solid var(--line); border-radius: 12px; padding: 10px; background: rgba(255,255,255,0.75); }
    .kpi .label { color: var(--sub); font-size: 12px; }
    .kpi .value { font-weight: 700; margin-top: 4px; }
    .log {
 margin-top: auto; min-height: 220px; max-height: 340px; overflow: auto;
      border: 1px solid var(--line); border-radius: 12px; background: rgba(13,32,23,0.9);
      color: #d7f5e5; padding: 12px; font-family: "Cascadia Mono", "Consolas", monospace; font-size: 13px;
      line-height: 1.6;
      white-space: pre-wrap;
      word-break: break-word;
    }
    .state.ok { color: var(--ok); }
    .state.warn { color: var(--warn); }
    .state.err { color: var(--err); }
    @media (max-width: 980px) {
      .grid { grid-template-columns: 1fr; }
      .kpis { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="header">
      <div>
        <h1 class="title">Vantalens 网站总控平台</h1>
        <p class="sub">统一查看前后端状态，并执行前端构建、后端路由巡检等管理操作。</p>
      </div>
      <div class="actions">
        <a class="btn primary" href="{{WRITER_URL}}">进入后端写作平台</a>
        <a class="btn" href="/platform/analytics">访问监控</a>
        <button class="btn" onclick="refreshStatus()">刷新状态</button>
      </div>
    </section>
    <section class="panel" style="margin-bottom:14px;">
      <div class="row" style="justify-content:space-between; align-items:center; gap:12px; flex-wrap:wrap;">
        <div>
          <h2 style="margin:0 0 6px;">管理员登录</h2>
          <p class="sub" style="margin:0;">先登录再执行构建和巡检命令。</p>
        </div>
        <div class="row" style="flex:1; justify-content:flex-end; align-items:center; flex-wrap:wrap;">
          <input id="login-user" value="admin" placeholder="用户名" style="max-width:180px;">
          <input id="login-pass" type="password" placeholder="密码" style="max-width:240px;">
          <button class="btn primary" onclick="loginAndStore()">登录并保存 Token</button>
          <span id="auth-state" class="btn">未登录</span>
        </div>
      </div>
    </section>
    <section class="grid">
      <article class="panel" style="display:flex;flex-direction:column;">
        <h2>前端总控</h2>
        <div class="kpis">
          <div class="kpi"><div class="label">Hugo 检查</div><div id="frontend-check" class="value state warn">未检查</div></div>
          <div class="kpi"><div class="label">最近构建</div><div id="frontend-build" class="value state warn">未执行</div></div>
          <div class="kpi"><div class="label">实时预览</div><div id="frontend-preview" class="value state warn">未启动</div></div>
          <div class="kpi"><div class="label">输出摘要</div><div id="frontend-out" class="value">-</div></div>
        </div>
        <h3>操作</h3>
        <div class="row">
          <button class="btn" onclick="runControl('frontend','check')">检查 Hugo</button>
          <button class="btn" onclick="runControl('frontend','build')">执行前端构建</button>
          <button class="btn primary" onclick="runControl('frontend','preview')">启动前端预览</button>
        </div>
      
 <h3>控制台日志</h3>
 <div id="log" class="log">初始化中...</div></article>

      <article class="panel">
        <h2>后端总控</h2>
        <div class="kpis">
          <div class="kpi"><div class="label">服务状态</div><div id="backend-check" class="value state warn">未检查</div></div>
          <div class="kpi"><div class="label">路由巡检</div><div id="backend-routes" class="value state warn">未执行</div></div>
          <div class="kpi"><div class="label">HUGO_PATH</div><div id="backend-path" class="value">-</div></div>
        </div>
        <h3>操作</h3>
        <div class="row">
          <button class="btn" onclick="runControl('backend','check')">后端健康巡检</button>
          <button class="btn" onclick="runControl('backend','routes')">后端路由巡检</button>
          <button class="btn" id="stop-writer-btn" onclick="runControl('backend','stop_writer')">关闭写作端口</button>
          <button class="btn" id="stop-control-btn" onclick="runControl('backend','stop_control')">关闭总控端口</button>
        </div>
      </article>


    
  </div>

  <script>
    const WRITER_URL = '{{WRITER_URL}}';

    function authHeaders() {
      const token = localStorage.getItem('ws_token') || localStorage.getItem('auth_token');
      return token ? { Authorization: 'Bearer ' + token } : {};
    }

    function clearAuthToken() {
      localStorage.removeItem('ws_token');
      localStorage.removeItem('auth_token');
      setAuthState();
    }

    function isTokenInvalid(res, data) {
      const msg = String((data && data.message) || '').toLowerCase();
      return res.status === 401 || msg.includes('invalid token') || msg.includes('unauthorized');
    }

    function setAuthState() {
      const token = localStorage.getItem('ws_token') || localStorage.getItem('auth_token');
      const state = document.getElementById('auth-state');
      if (state) {
        state.textContent = token ? '已登录' : '未登录';
      }
    }

    async function loginAndStore() {
      const username = document.getElementById('login-user').value.trim() || 'admin';
      const password = document.getElementById('login-pass').value;
      if (!password) {
        appendLog('请先输入管理员密码');
        return;
      }
      try {
        const res = await fetch('/api/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        const data = await res.json().catch(() => ({}));
        const token = data?.data?.access_token || data?.data?.token || data?.data?.refresh_token;
        if (!res.ok || !data.success || !token) {
          clearAuthToken();
          appendLog('登录失败: ' + (data.message || res.status));
          return;
        }
        localStorage.setItem('ws_token', token);
        localStorage.setItem('auth_token', token);
        setAuthState();
        appendLog('登录成功，Token 已保存');
      } catch (err) {
        appendLog('登录请求异常: ' + err.message);
      }
    }

    function appendLog(msg) {
      const el = document.getElementById('log');
      const line = '[' + new Date().toLocaleTimeString() + '] ' + msg;
      if (!el.textContent || el.textContent === '初始化中...') {
        el.textContent = line;
      } else {
        el.textContent += '\n' + line;
      }
      el.scrollTop = el.scrollHeight;
    }

    function stateClass(ok) {
      return ok ? 'state ok' : 'state err';
    }

    function setButtonState(id, disabled, label) {
      const button = document.getElementById(id);
      if (!button) return;
      button.disabled = !!disabled;
      if (label) button.textContent = label;
      button.style.opacity = disabled ? '0.55' : '1';
      button.style.cursor = disabled ? 'not-allowed' : 'pointer';
    }

    async function refreshStatus() {
      appendLog('请求 /api/control/status');
      const headers = authHeaders();
      if (!headers.Authorization) {
        appendLog('状态获取失败: 未检测到登录 token，请先登录后台');
        return;
      }
      try {
        const res = await fetch('/api/control/status', { headers });
        const data = await res.json();
        if (isTokenInvalid(res, data)) {
          clearAuthToken();
          appendLog('登录态失效，请重新登录');
          return;
        }
        if (!res.ok || !data.success) {
          appendLog('状态获取失败: ' + (data.message || res.status));
          return;
        }

        const frontend = data.data?.frontend || {};
        const backend = data.data?.backend || {};

        const frontOk = !!frontend.success;
        document.getElementById('frontend-check').className = 'value ' + stateClass(frontOk);
        document.getElementById('frontend-check').textContent = frontOk ? '可用' : '不可用';
        document.getElementById('frontend-out').textContent = (frontend.output || '-').slice(0, 40);

        document.getElementById('backend-check').className = 'value state ok';
        document.getElementById('backend-check').textContent = backend.service || 'online';
        document.getElementById('backend-path').textContent = backend.hugo_path || '-';
        if (backend.preview_running) {
          document.getElementById('frontend-preview').className = 'value state ok';
          document.getElementById('frontend-preview').textContent = '已启动';
        } else {
          document.getElementById('frontend-preview').className = 'value state warn';
          document.getElementById('frontend-preview').textContent = '未启动';
        }
        if (backend.launcher_mode === 'all') {
          setButtonState('stop-writer-btn', true, '统一模式内嵌');
          setButtonState('stop-control-btn', true, '请关闭程序窗口');
        } else {
          setButtonState('stop-writer-btn', false, '关闭写作端口');
          setButtonState('stop-control-btn', false, '关闭总控端口');
        }
        appendLog('状态更新完成');
      } catch (err) {
        appendLog('状态请求异常: ' + err.message);
      }
    }

    async function runControl(scope, action) {
      appendLog('执行命令: ' + scope + '/' + action);
      const headers = authHeaders();
      if (!headers.Authorization) {
        appendLog('失败: 未检测到登录 token，请先登录后台后重试');
        return;
      }
      try {
        const res = await fetch('/api/control/command', {
          method: 'POST',
          headers: Object.assign({ 'Content-Type': 'application/json' }, headers),
          body: JSON.stringify({ scope, action })
        });
        const rawText = await res.text();
        let data;
        try {
          data = rawText ? JSON.parse(rawText) : {};
        } catch (e) {
          data = { success: false, message: '服务返回了非 JSON 响应: ' + rawText.slice(0, 160) };
        }
        if (isTokenInvalid(res, data)) {
          clearAuthToken();
          appendLog('登录态失效，请重新登录');
        }
        const ok = !!(res.ok && data.success);
        const detail = data.data || data.message || ('HTTP ' + res.status);
        appendLog((ok ? '成功' : '失败') + ': ' + (typeof detail === 'string' ? detail : JSON.stringify(detail)));

        if (scope === 'frontend' && action === 'build') {
          const el = document.getElementById('frontend-build');
          el.className = 'value ' + stateClass(ok);
          el.textContent = ok ? '成功' : '失败';
        }
        if (scope === 'frontend' && action === 'preview') {
          const el = document.getElementById('frontend-preview');
          el.className = 'value ' + stateClass(ok);
          el.textContent = ok ? '已启动' : '失败';
          const previewUrl = data?.data?.result?.preview_url;
          if (ok && previewUrl) {
            appendLog('预览地址: ' + previewUrl);
            window.open(previewUrl, '_blank', 'noopener');
          }
        }
        if (scope === 'backend' && action === 'routes') {
          const el = document.getElementById('backend-routes');
          el.className = 'value ' + stateClass(ok);
          el.textContent = ok ? '完成' : '失败';
        }

        if (scope === 'backend' && (action === 'stop_writer' || action === 'stop_control')) {
          const message = String(data?.data?.result?.message || '');
          if (message.includes('nothing to stop separately') || message.includes('close the program window instead')) {
            document.getElementById('backend-check').className = 'value state warn';
            document.getElementById('backend-check').textContent = '统一模式';
          } else if (ok) {
            document.getElementById('backend-check').className = 'value state warn';
            document.getElementById('backend-check').textContent = '正在关闭';
          }
        }
      } catch (err) {
        appendLog('请求异常: ' + err.message);
      }
    }

    setAuthState();
    refreshStatus();
  </script>
</body>
</html>`
	page = strings.ReplaceAll(page, "{{WRITER_URL}}", writerURL)
	return page
}
