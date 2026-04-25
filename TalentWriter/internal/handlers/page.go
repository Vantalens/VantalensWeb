package handlers

import "strings"

func DashboardHTML(version string, controlURL string) string {
	page := `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Vantalens Writer 后台</title>
  <style>
    :root {
      color-scheme: dark;
      --bg0: #050816;
      --bg1: #0b1020;
      --panel: rgba(15, 23, 42, 0.82);
      --text: #e5eefc;
      --muted: #8da2c0;
      --border: rgba(148, 163, 184, 0.18);
      --shadow: 0 24px 70px rgba(2, 6, 23, 0.5);
      --danger: #fb7185;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--text);
      font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      background:
        radial-gradient(circle at 15% 20%, rgba(96, 165, 250, 0.22), transparent 28%),
        radial-gradient(circle at 80% 10%, rgba(94, 234, 212, 0.18), transparent 25%),
        linear-gradient(135deg, var(--bg0) 0%, var(--bg1) 55%, #101935 100%);
    }
    .shell { max-width: 1480px; margin: 0 auto; padding: 24px; }
    .topbar {
      display: flex; justify-content: space-between; align-items: center; gap: 16px;
      padding: 18px 20px; border: 1px solid var(--border); border-radius: 22px;
      background: rgba(15, 23, 42, 0.78); box-shadow: var(--shadow); backdrop-filter: blur(18px);
      margin-bottom: 18px;
    }
    .brand h1 { margin: 0; font-size: 22px; }
    .brand p { margin: 6px 0 0; color: var(--muted); font-size: 13px; }
    .statusline { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; justify-content: flex-end; }
    .pill { padding: 8px 12px; border-radius: 999px; border: 1px solid var(--border); background: rgba(96, 165, 250, 0.10); font-size: 13px; }
    .grid { display: grid; grid-template-columns: 320px minmax(0, 1fr); gap: 18px; }
    .panel { border: 1px solid var(--border); border-radius: 22px; background: var(--panel); box-shadow: var(--shadow); backdrop-filter: blur(18px); }
    .sidebar { padding: 18px; display: flex; flex-direction: column; gap: 14px; min-height: calc(100vh - 140px); }
    .main { padding: 18px; min-height: calc(100vh - 140px); }
    .card { border: 1px solid var(--border); border-radius: 18px; background: rgba(15, 23, 42, 0.66); padding: 14px; }
    .card h2 { margin: 0 0 10px; font-size: 16px; }
    .muted { color: var(--muted); }
    .row { display: flex; gap: 10px; }
    .stack { display: grid; gap: 10px; }
    .btn {
      appearance: none; border: 1px solid var(--border); border-radius: 14px; padding: 11px 14px;
      background: rgba(96, 165, 250, 0.08); color: var(--text); cursor: pointer; text-decoration: none;
      display: inline-flex; justify-content: center; align-items: center; gap: 8px;
    }
    .btn.primary { background: linear-gradient(135deg, rgba(96, 165, 250, 0.28), rgba(94, 234, 212, 0.20)); }
    .btn.danger { background: rgba(251, 113, 133, 0.10); }
    input, textarea {
      width: 100%; border: 1px solid var(--border); border-radius: 14px; background: rgba(2, 6, 23, 0.45);
      color: var(--text); padding: 12px 14px; outline: none; font: inherit;
    }
    textarea { min-height: 58vh; resize: vertical; line-height: 1.75; }
    .editor-head { display: grid; gap: 12px; margin-bottom: 14px; }
    .editor-head .meta { display: grid; grid-template-columns: 1.3fr 1fr 1fr; gap: 12px; }
    .list { display: grid; gap: 10px; max-height: 34vh; overflow: auto; padding-right: 4px; }
    .post {
      border: 1px solid var(--border); border-radius: 16px; background: rgba(2, 6, 23, 0.28);
      padding: 12px; cursor: pointer; transition: transform .15s ease, border-color .15s ease;
    }
    .post:hover { transform: translateY(-1px); border-color: rgba(96, 165, 250, 0.48); }
    .post.active { border-color: rgba(94, 234, 212, 0.8); box-shadow: 0 0 0 1px rgba(94, 234, 212, 0.14) inset; }
    .post-title { font-weight: 700; margin: 0 0 6px; }
    .post-meta { color: var(--muted); font-size: 12px; display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
    .editor-status { color: var(--muted); font-size: 13px; }
    .badge { display: inline-flex; align-items: center; padding: 3px 8px; border-radius: 999px; font-size: 12px; }
    .badge.ok { background: rgba(52, 211, 153, 0.14); color: #86efac; }
    .badge.warn { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
    .modal {
      position: fixed; inset: 0; display: none; align-items: center; justify-content: center; padding: 20px;
      background: rgba(2, 6, 23, 0.72); z-index: 20;
    }
    .modal-card { width: min(560px, 100%); border: 1px solid var(--border); border-radius: 22px; background: #0f172a; padding: 18px; box-shadow: var(--shadow); }
    .footer { margin-top: 14px; color: var(--muted); font-size: 12px; }
    .login-box {
      border: 1px solid var(--border);
      border-radius: 18px;
      background: rgba(2, 6, 23, 0.25);
      padding: 14px;
      display: grid;
      gap: 10px;
    }
    .login-box .row { flex-wrap: wrap; }
    @media (max-width: 1080px) { .grid { grid-template-columns: 1fr; } .sidebar, .main { min-height: auto; } textarea { min-height: 50vh; } }
    @media (max-width: 760px) { .topbar { flex-direction: column; align-items: flex-start; } .editor-head .meta { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <h1>Vantalens Writer 后台</h1>
        <p>文章编辑、保存、新建与删除都在这里完成。</p>
      </div>
      <div class="statusline">
        <span class="pill" id="service-pill">正在检查服务</span>
        <span class="pill" id="auth-pill">未登录</span>
        <span class="pill">v{{VERSION}}</span>
        <a class="btn" href="{{CONTROL_URL}}">总控平台</a>
      </div>
    </header>

    <section class="grid">
      <aside class="panel sidebar">
        <div class="card login-box">
          <h2>登录写作后台</h2>
          <div class="muted" style="font-size:12px; line-height:1.6;">不同端口会使用各自的登录状态。这里登录后，才能看到文章和标题。</div>
          <input id="login-user" value="admin" placeholder="用户名">
          <input id="login-pass" type="password" placeholder="密码">
          <div class="row">
            <button class="btn primary" onclick="loginAndStore()">登录</button>
            <span id="login-state" class="badge warn">未登录</span>
          </div>
        </div>

        <div class="card">
          <h2>工作区模式</h2>
          <div class="stack">
            <div class="muted" style="font-size:12px; line-height:1.6;">请先登录后再进行文章与评论管理操作。</div>
            <button class="btn" onclick="loadPosts()">刷新文章</button>
          </div>
        </div>

        <div class="card">
          <div class="row" style="justify-content:space-between; align-items:center; margin-bottom:10px;">
            <h2 style="margin:0">文章列表</h2>
            <button class="btn" onclick="loadPosts()">刷新</button>
          </div>
          <div class="list" id="post-list">
            <div class="muted">点击刷新加载文章。</div>
          </div>
        </div>

        <div class="card">
          <h2>新建文章</h2>
          <div class="stack">
            <input id="new-title" placeholder="文章标题">
            <input id="new-categories" placeholder="分类，逗号分隔">
            <button class="btn primary" onclick="createPost()">创建文章</button>
          </div>
        </div>

        <div class="card">
          <h2>文章操作</h2>
          <div class="stack">
            <button class="btn" onclick="runPlatformCommand('frontend','check')">检查前端环境</button>
            <button class="btn" onclick="runPlatformCommand('frontend','build')">执行前端构建</button>
          </div>
        </div>

        <div class="card">
          <h2>评论处理</h2>
          <div class="row" style="justify-content:space-between; align-items:center; margin-bottom:8px;">
            <span class="muted" id="comment-count">请选择文章</span>
            <button class="btn" onclick="loadCommentsForCurrent()">刷新评论</button>
          </div>
          <div id="comment-panel" class="list" style="max-height: 220px;"></div>
        </div>

        <div class="footer">默认保存位置为 Hugo 的 content 目录内。</div>
      </aside>

      <main class="panel main">
        <div class="editor-head">
          <div class="row" style="justify-content:space-between; align-items:center; gap:12px; flex-wrap:wrap;">
            <div>
              <div class="muted" style="font-size:12px;">当前文件</div>
              <div id="current-path" style="font-size:18px; font-weight:700; margin-top:4px;">未选择文章</div>
            </div>
            <div class="editor-status" id="editor-status">等待选择文章</div>
          </div>
          <div class="meta">
            <input id="meta-title" placeholder="标题">
            <input id="meta-date" placeholder="日期，如 2026-04-14T10:30:00+08:00">
            <input id="meta-categories" placeholder="分类，逗号分隔">
          </div>
          <div class="row" style="gap:12px; flex-wrap:wrap; align-items:center;">
            <label class="badge"><input id="meta-draft" type="checkbox" style="width:auto; margin-right:8px;">草稿</label>
            <label class="badge"><input id="meta-pinned" type="checkbox" style="width:auto; margin-right:8px;">置顶</label>
            <button class="btn primary" onclick="savePost()">保存文章</button>
            <button class="btn danger" onclick="deletePost()">删除文章</button>
          </div>
        </div>

        <textarea id="editor" placeholder="先从左侧选择一篇文章，或者新建一篇文章开始写作。"></textarea>
      </main>
    </section>
  </div>

  <div class="modal" id="login-modal">
    <div class="modal-card">
      <h3>需要登录</h3>
      <p class="muted" id="login-message">请先登录后再编辑文章。</p>
      <div class="row" style="justify-content:flex-end; margin-top:14px;">
        <button class="btn primary" onclick="closeLoginModal()">知道了</button>
      </div>
    </div>
  </div>

  <script>
    const state = { posts: [], currentPath: '' };

    function setAuth() {
      const token = localStorage.getItem('ws_token') || localStorage.getItem('auth_token');
      document.getElementById('auth-pill').textContent = token ? '已登录' : '未登录';
      const loginState = document.getElementById('login-state');
      if (loginState) {
        loginState.textContent = token ? '已登录' : '未登录';
        loginState.className = token ? 'badge ok' : 'badge warn';
      }
    }

    function clearAuthToken() {
      localStorage.removeItem('ws_token');
      localStorage.removeItem('auth_token');
      setAuth();
    }

    function isTokenInvalid(response, data) {
      const message = String((data && data.message) || '').toLowerCase();
      return response.status === 401 || message.includes('invalid token') || message.includes('unauthorized');
    }

    async function loginAndStore() {
      const username = document.getElementById('login-user').value.trim() || 'admin';
      const password = document.getElementById('login-pass').value;
      if (!password) {
        showLoginModal('请先输入管理员密码');
        return;
      }
      try {
        const response = await fetch('/api/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: username, password: password })
        });
        const data = await response.json().catch(() => ({}));
        const token = data?.data?.access_token || data?.data?.token || data?.data?.refresh_token;
        if (!response.ok || !data.success || !token) {
          clearAuthToken();
          showLoginModal(data.message || '登录失败');
          return;
        }
        localStorage.setItem('ws_token', token);
        localStorage.setItem('auth_token', token);
        setAuth();
        await loadPosts();
        await loadCommentsForCurrent();
      } catch (error) {
        showLoginModal('登录请求异常: ' + error.message);
      }
    }

    function authHeaders() {
      const token = localStorage.getItem('ws_token') || localStorage.getItem('auth_token');
      return token ? { Authorization: 'Bearer ' + token } : {};
    }

    async function authFetch(url, options = {}) {
      const headers = Object.assign({}, options.headers || {}, authHeaders());
      return fetch(url, Object.assign({}, options, { headers }));
    }

    function hasToken() {
      const token = localStorage.getItem('ws_token') || localStorage.getItem('auth_token');
      return !!token;
    }

    function showLoginModal(message) {
      document.getElementById('login-message').textContent = message || '请先登录后再编辑文章。';
      document.getElementById('login-modal').style.display = 'flex';
    }

    function closeLoginModal() {
      document.getElementById('login-modal').style.display = 'none';
    }

    async function checkHealth() {
      try {
        const response = await fetch('/health', { cache: 'no-store' });
        const data = await response.json();
        const ok = response.ok && data.status === 'ok';
        document.getElementById('service-pill').textContent = ok ? '服务在线' : '服务异常';
        document.getElementById('service-pill').style.background = ok ? 'rgba(52, 211, 153, 0.14)' : 'rgba(251, 191, 36, 0.14)';
      } catch (error) {
        document.getElementById('service-pill').textContent = '服务离线';
        document.getElementById('service-pill').style.background = 'rgba(251, 113, 133, 0.14)';
      }
    }

    async function loadPosts() {
      if (!hasToken()) {
        showLoginModal('请先登录后再加载文章列表');
        return;
      }
      const response = await authFetch('/api/posts');
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
        }
        showLoginModal(data.message || '无法加载文章列表');
        return;
      }
      state.posts = Array.isArray(data.data) ? data.data : [];
      renderPosts();
    }

    function renderPosts() {
      const list = document.getElementById('post-list');
      if (!state.posts.length) {
        list.innerHTML = '<div class="muted">暂无文章。</div>';
        return;
      }
      list.innerHTML = state.posts.map(function (post) {
        const active = post.path === state.currentPath ? 'active' : '';
        const badgeClass = post.status === 'DRAFT' ? 'warn' : 'ok';
        const encodedPath = encodeURIComponent(post.path || '');
        return '<div class="post ' + active + '" role="button" tabindex="0" data-path="' + encodedPath + '">' +
          '<div class="post-title">' + escapeHtml(post.title || fallbackTitleFromPath(post.path) || 'Untitled') + '</div>' +
          '<div class="post-meta"><span>' + escapeHtml(post.path) + '</span><span class="badge ' + badgeClass + '">' + escapeHtml(post.status || 'PUBLISHED') + '</span></div>' +
          '</div>';
      }).join('');
    }

    function bindPostListEvents() {
      const list = document.getElementById('post-list');
      if (!list || list.dataset.bound === '1') return;
      list.addEventListener('click', function (event) {
        const item = event.target.closest('.post[data-path]');
        if (!item || !list.contains(item)) return;
        const encodedPath = item.getAttribute('data-path');
        if (!encodedPath) return;
        openPost(decodeURIComponent(encodedPath));
      });
      list.addEventListener('keydown', function (event) {
        if (event.key !== 'Enter' && event.key !== ' ') return;
        const item = event.target.closest('.post[data-path]');
        if (!item || !list.contains(item)) return;
        event.preventDefault();
        const encodedPath = item.getAttribute('data-path');
        if (!encodedPath) return;
        openPost(decodeURIComponent(encodedPath));
      });
      list.dataset.bound = '1';
    }

    async function openPost(path) {
      state.currentPath = path;
      renderPosts();
      document.getElementById('editor-status').textContent = '加载中...';
      const response = await authFetch('/api/get_content?path=' + encodeURIComponent(path));
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
        }
        document.getElementById('editor-status').textContent = data.message || '读取失败';
        return;
      }
      const content = data?.data?.content || '';
      document.getElementById('current-path').textContent = path;
      document.getElementById('editor').value = content;
      applyFrontmatterToForm(content);
      document.getElementById('editor-status').textContent = '已载入';
      await loadCommentsForCurrent();
    }

    function applyFrontmatterToForm(content) {
      const meta = parseFrontmatter(content);
      document.getElementById('meta-title').value = meta.title || fallbackTitleFromPath(state.currentPath) || '';
      document.getElementById('meta-date').value = meta.date || '';
      document.getElementById('meta-categories').value = (meta.categories || []).join(', ');
      document.getElementById('meta-draft').checked = !!meta.draft;
      document.getElementById('meta-pinned').checked = !!meta.pinned;
    }

    function fallbackTitleFromPath(path) {
      if (!path) return '';
      const file = String(path).split(/[\\/]/).pop() || '';
      const parent = String(path).split(/[\\/]/).slice(-2, -1)[0] || '';
      if (parent && parent !== 'index.md') {
        return parent.replace(/[-_]+/g, ' ').replace(/\b\w/g, function (ch) { return ch.toUpperCase(); });
      }
      return file.replace(/\.md$/i, '').replace(/[-_]+/g, ' ');
    }

    function parseFrontmatter(content) {
      const fallback = { title: '', date: '', categories: [], draft: false, pinned: false };
      if (!content || !content.startsWith('---')) return fallback;
      const match = content.match(/^---\n([\s\S]*?)\n---\n?/);
      if (!match) return fallback;
      const lines = match[1].split('\n');
      const meta = Object.assign({}, fallback);
      let currentKey = '';
      lines.forEach(function (line) {
        if (/^\s*-/.test(line)) {
          const value = line.replace(/^\s*-\s*/, '').trim();
          if (currentKey === 'categories' && value) meta.categories.push(value);
          return;
        }
        const pair = line.match(/^([A-Za-z0-9_\-]+):\s*(.*)$/);
        if (!pair) return;
        currentKey = pair[1];
        const raw = pair[2].trim();
        if (currentKey === 'title') meta.title = raw.replace(/^"|"$/g, '');
        if (currentKey === 'date') meta.date = raw.replace(/^"|"$/g, '');
        if (currentKey === 'draft') meta.draft = raw.toLowerCase() === 'true';
        if (currentKey === 'pinned') meta.pinned = raw.toLowerCase() === 'true';
        if (currentKey === 'categories' && raw.startsWith('[')) {
          meta.categories = raw.replace(/[\[\]]/g, '').split(',').map(function (item) { return item.trim(); }).filter(Boolean);
        }
      });
      return meta;
    }

    function splitBody(content) {
      const match = content.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/);
      return match ? match[1] : content;
    }

    function rebuildContent() {
      const body = splitBody(document.getElementById('editor').value || '');
      const title = document.getElementById('meta-title').value.trim();
      const date = document.getElementById('meta-date').value.trim();
      const categories = document.getElementById('meta-categories').value.split(',').map(function (item) { return item.trim(); }).filter(Boolean);
      const draft = document.getElementById('meta-draft').checked;
      const pinned = document.getElementById('meta-pinned').checked;
      const lines = ['---'];
      lines.push('title: ' + JSON.stringify(title || 'Untitled'));
      if (date) lines.push('date: ' + date);
      lines.push('draft: ' + String(draft));
      lines.push('pinned: ' + String(pinned));
      if (categories.length) {
        lines.push('categories:');
        categories.forEach(function (category) { lines.push('  - ' + category); });
      }
      lines.push('---');
      return lines.join('\n') + '\n' + body.replace(/^\n+/, '');
    }

    async function savePost() {
      if (!state.currentPath) {
        showLoginModal('请先选择一篇文章。');
        return;
      }
      const content = rebuildContent();
      const response = await authFetch('/api/save_content', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: state.currentPath, content: content })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.success) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
        }
        document.getElementById('editor-status').textContent = data.message || '保存失败';
        return;
      }
      document.getElementById('editor').value = content;
      document.getElementById('editor-status').textContent = '已保存';
      await loadPosts();
    }

    async function deletePost() {
      if (!state.currentPath) return;
      if (!confirm('确定删除这篇文章？此操作不可恢复。')) return;
      const response = await authFetch('/api/delete_post', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: state.currentPath })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.success) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
        }
        document.getElementById('editor-status').textContent = data.message || '删除失败';
        return;
      }
      state.currentPath = '';
      document.getElementById('current-path').textContent = '未选择文章';
      document.getElementById('editor').value = '';
      document.getElementById('editor-status').textContent = '文章已删除';
      await loadPosts();
    }

    async function createPost() {
      const title = document.getElementById('new-title').value.trim();
      const categories = document.getElementById('new-categories').value.trim();
      if (!title) {
        showLoginModal('请输入文章标题。');
        return;
      }
      const response = await authFetch('/api/create_post', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: title, categories: categories })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.success) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
        }
        showLoginModal(data.message || '创建失败');
        return;
      }
      document.getElementById('new-title').value = '';
      document.getElementById('new-categories').value = '';
      await loadPosts();
      if (data?.data?.path) {
        await openPost(data.data.path);
      }
    }

    async function runPlatformCommand(scope, action) {
      if (!hasToken()) {
        showLoginModal('请先登录后再执行平台命令');
        return;
      }
      const response = await authFetch('/api/control/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scope: scope, action: action })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.success) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
        }
        showLoginModal(data.message || '命令执行失败');
        return;
      }
      const result = JSON.stringify(data.data || {}, null, 2);
      alert('执行成功:\n' + result.slice(0, 500));
    }

    async function loadCommentsForCurrent() {
      const panel = document.getElementById('comment-panel');
      const count = document.getElementById('comment-count');
      if (!hasToken()) {
        panel.innerHTML = '<div class="muted">请先登录后再刷新评论</div>';
        count.textContent = '需要登录';
        showLoginModal('请先登录后再刷新评论');
        return;
      }
      panel.innerHTML = '<div class="muted">加载中...</div>';
      const response = await authFetch('/api/comments?all=1');
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.success) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
          panel.innerHTML = '<div class="muted">登录态已失效，请重新登录后刷新评论</div>';
          count.textContent = '需要重新登录';
          showLoginModal('登录态已失效，请重新登录后刷新评论');
          return;
        }
        panel.innerHTML = '<div class="muted">评论加载失败</div>';
        count.textContent = '加载失败';
        return;
      }
      const comments = Array.isArray(data.data) ? data.data : [];
      const pending = comments.filter(function (item) { return !item.approved; }).length;
      const approvedCount = comments.length - pending;
      count.textContent = '全库评论 ' + comments.length + ' 条，已审核 ' + approvedCount + ' 条，待审核 ' + pending + ' 条';
      if (!comments.length) {
        panel.innerHTML = '<div class="muted">数据库中暂无评论。提交成功但未出现在这里时，请先确认前台是否提示“评论已提交”。</div>';
        return;
      }
      const summary = '<div class="muted" style="margin-bottom:10px;">这里显示整个评论数据库，不按当前文章过滤。前台文章页只展示“已审核”的评论，待审核评论不会公开。</div>';
      panel.innerHTML = comments.map(function (item) {
        const approved = !!item.approved;
        const itemPath = item.post_path || state.currentPath || '';
        const risk = Array.isArray(item.risk_reasons) && item.risk_reasons.length
          ? '<div class="post-meta"><span>风控：' + escapeHtml(item.risk_reasons.join('；')) + '</span></div>'
          : '';
        return '<div class="post" style="' + (approved ? '' : 'border-color:rgba(245,158,11,.45);') + '">' +
          '<div class="post-title" style="font-size:13px;">' + escapeHtml(item.author || '匿名') + ' · ' + (approved ? '已审核，可在前台显示' : '待审核，前台暂不显示') + '</div>' +
          '<div class="post-meta"><span>' + escapeHtml(item.timestamp || '-') + '</span><span>' + escapeHtml(itemPath) + '</span></div>' +
          '<div class="post-meta"><span>' + escapeHtml((item.content || '').slice(0, 120)) + '</span></div>' +
          risk +
          '<div class="row" style="margin-top:8px;">' +
          (approved ? '<button class="btn" disabled>已通过</button>' : '<button class="btn" onclick="approveComment(' + JSON.stringify(item.id || '') + ')">审核通过</button>') +
          '<button class="btn danger" onclick="removeComment(' + JSON.stringify(item.id || '') + ')">删除</button>' +
          '</div>' +
          '</div>';
      }).join('');
      panel.innerHTML = summary + panel.innerHTML;
    }

    async function approveComment(id) {
      if (!id) return;
      const response = await authFetch('/api/comments/approve?id=' + encodeURIComponent(id), {
        method: 'POST'
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.success) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
          showLoginModal('登录态已失效，请重新登录后审核评论');
          return;
        }
        showLoginModal(data.message || '审核失败');
        return;
      }
      await loadCommentsForCurrent();
    }

    async function removeComment(id) {
      if (!id) return;
      if (!confirm('确认删除这条评论吗？')) return;
      const response = await authFetch('/api/comments/delete?id=' + encodeURIComponent(id), {
        method: 'POST'
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.success) {
        if (isTokenInvalid(response, data)) {
          clearAuthToken();
          showLoginModal('登录态已失效，请重新登录后删除评论');
          return;
        }
        showLoginModal(data.message || '删除失败');
        return;
      }
      await loadCommentsForCurrent();
    }

    function escapeHtml(input) {
      return String(input || '')
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
    }

    (async function init() {
      setAuth();
      bindPostListEvents();
      await checkHealth();
      if (hasToken()) {
        await loadPosts();
        await loadCommentsForCurrent();
      }
    })();
  </script>
</body>
</html>`
	page = strings.ReplaceAll(page, "{{VERSION}}", version)
	page = strings.ReplaceAll(page, "{{CONTROL_URL}}", controlURL)
	return page
}
