package handlers

import "strings"

func platformHTML(title, subtitle, active, body, script string) string {
	page := `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{TITLE}}</title>
  <style>
    :root {
      color-scheme: light;
      --bg0:#fffaf2; --bg1:#fff1f7; --bg2:#eefcf7;
      --panel:rgba(255,255,255,.78); --card:rgba(250,249,245,.82);
      --text:#392833; --muted:rgba(82,58,71,.68); --line:rgba(217,190,160,.42);
      --accent:#d46a92; --teal:#4f9e8f; --gold:#c79545; --danger:#c24c5f;
      --shadow:0 18px 45px rgba(161,120,89,.16);
      --shadow-card:0 6px 18px rgba(161,120,89,.10);
    }
    *{box-sizing:border-box} body{margin:0;min-height:100vh;font-family:"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:var(--text);background:linear-gradient(135deg,var(--bg0),var(--bg1) 52%,var(--bg2))}
    .shell{max-width:1480px;margin:0 auto;padding:22px}
    .topbar,.panel{border:1px solid var(--line);background:var(--panel);box-shadow:var(--shadow);backdrop-filter:blur(16px)}
    .card{border:1px solid var(--line);background:var(--card);box-shadow:var(--shadow-card)}
    .topbar{display:grid;grid-template-columns:minmax(260px,1fr) auto;gap:16px;align-items:center;border-radius:14px;padding:18px 20px;margin-bottom:16px}
    .brand h1{margin:0;font-size:24px}.brand p{margin:6px 0 0;color:var(--muted);font-size:13px;line-height:1.6}
    .nav{display:flex;gap:8px;flex-wrap:wrap;justify-content:flex-end}
    .nav a,.btn{border:1px solid var(--line);border-radius:10px;background:rgba(255,255,255,.68);color:var(--text);padding:10px 13px;text-decoration:none;font:inherit;cursor:pointer;display:inline-flex;align-items:center;justify-content:center;gap:6px;transition:box-shadow .16s ease,border-color .16s ease,transform .16s ease}
    .nav a:hover,.btn:hover{border-color:rgba(212,106,146,.45)}
    .nav a.active,.btn.primary{background:linear-gradient(135deg,rgba(212,106,146,.24),rgba(79,158,143,.16));border-color:rgba(212,106,146,.36)}
    .nav a.active{font-weight:700;color:#9c3f63;box-shadow:0 0 0 1px rgba(212,106,146,.28) inset,0 8px 18px rgba(212,106,146,.16)}
    .btn.danger{background:rgba(194,76,95,.10)}.btn:disabled{opacity:.58;cursor:not-allowed}
    .statusbar{display:flex;gap:10px;flex-wrap:wrap;align-items:center;margin-bottom:16px}
    .pill-row{display:flex;gap:10px;flex-wrap:wrap;align-items:center;flex:1;min-width:0}
    .pill,.badge{display:inline-flex;align-items:center;border:1px solid var(--line);border-radius:999px;padding:7px 10px;background:rgba(255,255,255,.58);font-size:12px;color:var(--muted)}
    .badge.ok{color:#236d62;background:rgba(79,158,143,.14)}.badge.warn{color:#8a5a10;background:rgba(199,149,69,.16)}.badge.err{color:var(--danger);background:rgba(194,76,95,.12)}
    .quick-login{margin-left:auto;display:flex;gap:8px;align-items:center;flex-wrap:wrap}
    .quick-login input{width:auto;min-width:132px;padding:8px 10px;border-radius:999px}.quick-login .btn{padding:8px 11px;border-radius:999px}
    .hidden{display:none!important}
    .panel{border-radius:14px;padding:16px;margin-bottom:16px}.card{border-radius:12px;padding:14px}
    .grid{display:grid;gap:16px}.grid.two{grid-template-columns:320px minmax(0,1fr)}.grid.three{grid-template-columns:repeat(3,minmax(0,1fr))}.grid.four{grid-template-columns:repeat(4,minmax(0,1fr))}
    .stack{display:grid;gap:10px}.row{display:flex;gap:10px;flex-wrap:wrap;align-items:center}.between{justify-content:space-between}
    .muted{color:var(--muted)}.metric{font-size:28px;font-weight:800;margin-top:6px}.section-title{margin:0 0 10px;font-size:17px}.small{font-size:12px;line-height:1.6}
    input,textarea,select{width:100%;border:1px solid var(--line);border-radius:10px;background:rgba(255,255,255,.74);color:var(--text);padding:11px 12px;font:inherit;outline:none}
    textarea{min-height:58vh;resize:vertical;line-height:1.75}
    .list{display:grid;gap:10px;max-height:58vh;overflow:auto;padding-right:4px}
    .item{border:1px solid var(--line);border-radius:10px;background:rgba(255,255,255,.58);padding:12px}
    .item.active{border-color:rgba(79,158,143,.8);box-shadow:0 0 0 1px rgba(79,158,143,.14) inset}
    .item-title{font-weight:700;margin-bottom:6px}.item-meta{color:var(--muted);font-size:12px;display:flex;gap:8px;justify-content:space-between;flex-wrap:wrap}
    .status{font-size:13px;color:var(--muted)}
    details{border:1px solid var(--line);border-radius:10px;background:rgba(255,255,255,.56);overflow:hidden}
    summary{cursor:pointer;padding:11px 12px;display:flex;justify-content:space-between;gap:10px;align-items:center}
    .detail-body{padding:0 12px 12px;color:var(--muted);font-size:12px;white-space:pre-wrap;word-break:break-word}
    .table-wrap{overflow:auto}table{width:100%;border-collapse:collapse}th,td{padding:10px 8px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top;font-size:13px}th{color:var(--muted);font-size:12px}
    .mono{font-family:"Cascadia Mono","Consolas",monospace;font-size:12px}
    .home-card{display:block;color:inherit;text-decoration:none;position:relative;transition:transform .16s ease,border-color .16s ease,box-shadow .16s ease}
    .home-card:hover{transform:translateY(-3px);border-color:rgba(212,106,146,.5);box-shadow:0 14px 30px rgba(161,120,89,.18)}
    .home-card .icon{width:36px;height:36px;border-radius:10px;display:inline-flex;align-items:center;justify-content:center;background:linear-gradient(135deg,rgba(212,106,146,.18),rgba(79,158,143,.14));margin-bottom:10px}
    .home-card svg{width:20px;height:20px;stroke:#a44a70;fill:none}
    #toast-root{position:fixed;right:18px;bottom:18px;display:grid;gap:8px;z-index:9999;max-width:min(360px,90vw)}
    .toast{border:1px solid var(--line);border-radius:10px;background:rgba(255,255,255,.94);box-shadow:0 10px 26px rgba(161,120,89,.22);padding:10px 14px;font-size:13px;animation:toast-in .18s ease}
    .toast.ok{border-left:4px solid var(--teal)}.toast.err{border-left:4px solid var(--danger)}.toast.info{border-left:4px solid var(--gold)}
    @keyframes toast-in{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:none}}
    .md-toolbar{display:flex;gap:6px;flex-wrap:wrap}
    .md-toolbar .btn{padding:6px 10px;font-size:12px;border-radius:8px}
    .seg{display:inline-flex;border:1px solid var(--line);border-radius:10px;overflow:hidden;background:rgba(255,255,255,.55)}
    .seg button{border:0;background:transparent;padding:8px 13px;cursor:pointer;font:inherit;font-size:13px;color:var(--muted)}
    .seg button.active{background:linear-gradient(135deg,rgba(212,106,146,.22),rgba(79,158,143,.14));color:var(--text);font-weight:700}
    .editor-wrap.view-preview .editor-area{display:none}
    .editor-wrap.view-edit #preview{display:none}
    .editor-wrap.view-split{display:grid;grid-template-columns:1fr 1fr;gap:12px;align-items:start}
    .editor-wrap.view-split #editor,.editor-wrap.view-split #preview{min-height:52vh}
    .editor-area{position:relative}
    #editor{font-family:"Cascadia Mono","Consolas",monospace;font-size:13.5px;tab-size:2}
    #line-hl{position:absolute;left:0;right:0;background:rgba(212,106,146,.10);border-radius:6px;pointer-events:none;display:none;z-index:0}
    .editor-area.hl-on #editor{background:transparent;position:relative;z-index:1}
    #preview{border:1px solid var(--line);border-radius:10px;background:rgba(255,255,255,.74);padding:12px 16px;min-height:58vh;overflow:auto;font-size:14px;line-height:1.75}
    .md-body h1,.md-body h2,.md-body h3,.md-body h4,.md-body h5,.md-body h6{margin:14px 0 8px;line-height:1.4}
    .md-body p{margin:8px 0}
    .md-body code{font-family:"Cascadia Mono","Consolas",monospace;background:rgba(79,158,143,.12);border-radius:6px;padding:1px 5px;font-size:.92em}
    .md-body pre{background:#2f2731;color:#f5e9ee;border-radius:10px;padding:12px 14px;overflow:auto;font-size:12.5px;line-height:1.6}
    .md-body pre code{background:none;padding:0;color:inherit}
    .md-body blockquote{border-left:3px solid var(--accent);margin:8px 0;padding:4px 12px;color:var(--muted);background:rgba(212,106,146,.06);border-radius:0 8px 8px 0}
    .md-body img{max-width:100%;border-radius:10px;border:1px solid var(--line)}
    .md-body img.img-broken{display:inline-flex;align-items:center;justify-content:center;min-width:150px;min-height:64px;border:1px dashed var(--danger);background:rgba(194,76,95,.06);color:var(--danger);font-size:12px;padding:10px}
    .md-body hr{border:0;border-top:1px solid var(--line);margin:14px 0}
    .md-body ul,.md-body ol{margin:8px 0;padding-left:22px}
    .md-body a{color:var(--accent)}
    @media(max-width:1080px){.topbar{grid-template-columns:1fr}.nav{justify-content:flex-start}.grid.two,.grid.three,.grid.four{grid-template-columns:1fr 1fr}textarea{min-height:46vh}.quick-login{margin-left:0;width:100%}.editor-wrap.view-split{grid-template-columns:1fr}}
    @media(max-width:760px){.shell{padding:14px}.grid.two,.grid.three,.grid.four{grid-template-columns:1fr}.row{align-items:stretch}.btn,.nav a{width:100%}.metric{font-size:24px}.quick-login,.quick-login input,.quick-login .btn{width:100%}.pill-row{width:100%;flex-wrap:nowrap;overflow-x:auto;padding-bottom:6px}.pill-row .pill,.pill-row .badge{flex:0 0 auto;white-space:nowrap}.md-toolbar .btn{width:auto}.seg button{flex:1}.seg{width:100%}}
  </style>
</head>
<body>
  <div class="shell">
    <header class="topbar">
      <div class="brand"><h1>{{TITLE}}</h1><p>{{SUBTITLE}}</p></div>
      <nav class="nav">
        <a class="{{HOME_ACTIVE}}" href="/platform">入口</a>
        <a class="{{POSTS_ACTIVE}}" href="/platform/posts">文章</a>
        <a class="{{COMMENTS_ACTIVE}}" href="/platform/comments">评论</a>
        <a class="{{ANALYTICS_ACTIVE}}" href="/platform/analytics">监控</a>
      </nav>
    </header>
    <section class="statusbar">
      <div class="pill-row">
        <span id="auth-pill" class="pill">未登录</span>
        <span id="remote-pill" class="pill">远端状态未检查</span>
        <span id="health-pill" class="pill">服务状态未检查</span>
        <span id="time-pill" class="pill">北京时间 --</span>
      </div>
      <div id="quick-login" class="quick-login hidden">
        <input id="login-user" value="vantalens" aria-label="用户名">
        <input id="login-pass" type="password" placeholder="后台密码" aria-label="后台密码">
        <button class="btn primary" onclick="manualLogin(this)">登录</button>
        <span id="login-status" class="status">等待登录</span>
      </div>
      <button id="logout-btn" class="btn hidden" onclick="clearToken();checkShell()">退出</button>
    </section>
    {{BODY}}
  </div>
  <div id="toast-root" aria-live="polite"></div>
  <script>
    const $ = (id) => document.getElementById(id);
    function token(){return localStorage.getItem('ws_token')||localStorage.getItem('auth_token')||''}
    function authHeaders(){const t=token(); return t?{Authorization:'Bearer '+t}:{}}
    function hasToken(){return !!token()}
    function setToken(t){localStorage.setItem('ws_token',t);localStorage.setItem('auth_token',t);setAuthState();window.dispatchEvent(new Event('auth-ready'))}
    function clearToken(){localStorage.removeItem('ws_token');localStorage.removeItem('auth_token');setAuthState()}
    function formatBeijingTime(value){if(!value)return'-';const d=new Date(value);if(Number.isNaN(d.getTime()))return String(value);return new Intl.DateTimeFormat('zh-CN',{timeZone:'Asia/Shanghai',year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',second:'2-digit',hour12:false}).format(d).replaceAll('/','-')}
    function refreshBeijingClock(){const el=$('time-pill');if(el)el.textContent='北京时间 '+formatBeijingTime(new Date().toISOString())}
    function setAuthState(){const ok=hasToken();const el=$('auth-pill'); if(el){el.textContent=ok?'已登录':'未登录'; el.className=ok?'badge ok':'badge warn'}const login=$('quick-login');if(login)login.classList.toggle('hidden',ok);const logout=$('logout-btn');if(logout)logout.classList.toggle('hidden',!ok)}
    function escapeHtml(v){return String(v??'').replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;').replaceAll("'","&#39;")}
    function setButton(btn,busy,label){if(!btn)return;if(!btn.dataset.text)btn.dataset.text=btn.textContent;btn.disabled=!!busy;btn.textContent=busy?label:btn.dataset.text}
    function setText(id,text,kind){const el=$(id); if(!el)return; el.textContent=text; el.style.color=kind==='error'?'var(--danger)':kind==='success'?'var(--teal)':'var(--muted)'}
    function toast(msg,kind){const root=$('toast-root');if(!root)return;const el=document.createElement('div');el.className='toast '+(kind||'info');el.textContent=msg;root.appendChild(el);setTimeout(()=>{el.style.opacity='0';el.style.transition='opacity .3s';setTimeout(()=>el.remove(),320)},3600)}
    async function authFetch(url, options={}){const headers=Object.assign({},options.headers||{},authHeaders());return fetch(url,Object.assign({},options,{headers}))}
    function invalidAuth(res,data){const msg=String(data?.message||'').toLowerCase();return res.status===401||msg.includes('invalid token')||msg.includes('unauthorized')}
    async function loginFromFields(){const u=($('login-user')?.value||'vantalens').trim()||'vantalens';const p=$('login-pass')?.value||'';const r=await fetch('/api/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:u,password:p})});const d=await r.json().catch(()=>({}));const t=d?.data?.access_token||d?.data?.token;if(!r.ok||!d.success||!t)throw new Error(d.message||'登录失败');setToken(t);return true}
    async function manualLogin(btn){try{setButton(btn,true,'登录中...');await loginFromFields();setText('login-status','已登录','success');toast('登录成功','ok');await checkShell()}catch(e){setText('login-status',e.message,'error');toast('登录失败：'+e.message,'err')}finally{setButton(btn,false)}}
    async function tryPlatformSession(){try{const r=await fetch('/platform/session',{cache:'no-store',credentials:'same-origin'});const d=await r.json().catch(()=>({}));const t=d?.data?.access_token||d?.data?.token;if(r.ok&&d.success&&t){setToken(t);return true}}catch(e){}return false}
    async function checkShell(){refreshBeijingClock();setAuthState();try{const h=await fetch('/api/health',{cache:'no-store'});const d=await h.json();const ok=h.ok&&d.status==='ok';const el=$('health-pill');if(el){el.textContent=ok?'服务在线':'服务异常';el.className=ok?'badge ok':'badge err'}}catch(e){const el=$('health-pill');if(el){el.textContent='服务离线';el.className='badge err'}} if(hasToken()){try{const r=await authFetch('/api/sync/status');const d=await r.json().catch(()=>({}));const remote=d?.data?.remote||{};const el=$('remote-pill');if(el){el.textContent=remote.enabled?(remote.reachable?'远端实时连接':'远端不可达'):'本机权威模式';el.className=remote.reachable||!remote.enabled?'badge ok':'badge err'}}catch(e){const el=$('remote-pill');if(el){el.textContent='远端状态检查失败';el.className='badge err'}}}}
    async function bootstrapShell(){refreshBeijingClock();setInterval(refreshBeijingClock,1000);if(!hasToken())await tryPlatformSession();await checkShell()}
    bootstrapShell();
    {{SCRIPT}}
  </script>
</body>
</html>`
	repl := map[string]string{
		"{{TITLE}}":            title,
		"{{SUBTITLE}}":         subtitle,
		"{{BODY}}":             body,
		"{{SCRIPT}}":           script,
		"{{HOME_ACTIVE}}":      activeClass(active, "home"),
		"{{POSTS_ACTIVE}}":     activeClass(active, "posts"),
		"{{COMMENTS_ACTIVE}}":  activeClass(active, "comments"),
		"{{ANALYTICS_ACTIVE}}": activeClass(active, "analytics"),
	}
	for k, v := range repl {
		page = strings.ReplaceAll(page, k, v)
	}
	return page
}

func activeClass(active, name string) string {
	if active == name {
		return "active"
	}
	return ""
}

func PlatformHomeHTML() string {
	body := `
    <section class="grid three">
      <a class="card home-card" href="/platform/posts">
        <div class="icon"><svg viewBox="0 0 24 24" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg></div>
        <div class="section-title">文章管理</div>
        <div class="muted small">创建、编辑、保存和删除 Hugo 文章，支持 Markdown 预览。</div>
        <div id="post-count" class="metric">-</div>
        <div class="muted small">进入文章工作台</div>
      </a>
      <a class="card home-card" href="/platform/comments">
        <div class="icon"><svg viewBox="0 0 24 24" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg></div>
        <div class="section-title">评论审核</div>
        <div class="muted small">审核或删除服务器权威评论库中的评论，支持批量操作。</div>
        <div id="comment-count" class="metric">-</div>
        <div class="muted small">进入评论工作台</div>
      </a>
      <a class="card home-card" href="/platform/analytics">
        <div class="icon"><svg viewBox="0 0 24 24" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 3v18h18"/><path d="M7 15l4-6 4 3 5-8"/></svg></div>
        <div class="section-title">访问监控</div>
        <div class="muted small">查看标准世界地图、地区、IP 和最近访问。</div>
        <div id="visit-count" class="metric">-</div>
        <div class="muted small">进入监控面板</div>
      </a>
    </section>
    <section class="panel">
      <div class="row between"><div><h2 class="section-title">后台总览</h2><div class="muted small">入口页只保留核心业务状态。具体操作进入对应页面完成。</div></div><button class="btn" onclick="refreshHome(this)">刷新总览</button></div>
      <div class="row" style="margin-top:12px"><span id="publish-pill" class="pill">发布状态未检查</span><span id="sync-pill" class="pill">同步状态未检查</span></div>
      <div id="home-status" class="status" style="margin-top:12px">等待刷新</div>
    </section>`
	script := `
    function renderHomePublish(s){const el=$('publish-pill');if(!el)return;s=s||{};const st=s.state||'';el.textContent=st==='running'?'发布中…':st==='succeeded'?'发布产物已生成':st==='failed'?'发布失败':'尚未发布';el.className=st==='succeeded'?'badge ok':st==='failed'?'badge err':st==='running'?'badge warn':'pill'}
    function renderHomeSync(remote){const el=$('sync-pill');if(!el)return;remote=remote||{};el.textContent=remote.enabled?(remote.reachable?'远端实时连接':'远端不可达'):'本机权威模式';el.className=remote.reachable||!remote.enabled?'badge ok':'badge err'}
    async function refreshHome(btn){setButton(btn,true,'刷新中...');try{if(!hasToken()){setText('home-status','请先完成顶部登录','error');toast('请先登录后再刷新总览','err');return}
      const get=(u)=>authFetch(u).then(r=>r.json().catch(()=>({}))).catch(()=>({}));
      const [posts,comments,stats,publish,sync]=await Promise.all([get('/api/posts'),get('/api/comments?all=1'),get('/api/analytics/stats?limit=1'),get('/api/publish/status'),get('/api/sync/status')]);
      $('post-count').textContent=Array.isArray(posts.data)?posts.data.length:'-';
      const cs=Array.isArray(comments.data)?comments.data:[];$('comment-count').textContent=cs.filter(c=>!c.approved).length+' 待审';
      $('visit-count').textContent=stats?.data?.total_views??'-';
      renderHomePublish(publish?.data);renderHomeSync(sync?.data?.remote);
      setText('home-status','总览已刷新：文章 '+$('post-count').textContent+'，待审评论 '+$('comment-count').textContent+'，访问 '+$('visit-count').textContent,'success');checkShell()}catch(e){setText('home-status','刷新失败：'+e.message,'error');toast('总览刷新失败：'+e.message,'err')}finally{setButton(btn,false)}}
    document.querySelectorAll('.home-card').forEach(card=>card.addEventListener('click',e=>{if(hasToken())return;e.preventDefault();toast('请先登录后再进入工作台','err');const p=$('login-pass');if(p){p.scrollIntoView({behavior:'smooth',block:'center'});p.focus()}}));
    window.addEventListener('auth-ready',()=>refreshHome()); setTimeout(()=>{if(hasToken()) refreshHome()},350);`
	return platformHTML("Vantalens 后台入口", "文章、评论、访问监控集中入口。入口页保持干净，具体操作进入对应页面完成。", "home", body, script)
}

func PostsPageHTML(version string) string {
	_ = version
	body := `
    <section class="grid two">
      <aside class="panel stack">
        <div class="card"><div class="row between"><h2 class="section-title">文章列表</h2><button class="btn" onclick="loadPosts(this)">刷新</button></div><input id="post-search" placeholder="搜索标题或路径" style="margin-bottom:10px"><div id="post-list" class="list" style="max-height:30vh"><div class="muted">登录后刷新文章。</div></div></div>
        <div class="card"><h2 class="section-title">新建文章</h2><div class="stack"><input id="new-title" placeholder="文章标题"><input id="new-categories" placeholder="分类，逗号分隔"><button class="btn primary" onclick="createPost(this)">创建草稿</button></div></div>
        <div class="card"><div class="row between"><h2 class="section-title">回收站</h2><button class="btn" onclick="loadTrash(this)">刷新</button></div><div id="trash-list" class="stack"><div class="muted small">尚未加载。</div></div></div>
        <div class="card"><h2 class="section-title">检查与发布</h2><div class="row"><button class="btn" onclick="runBuild('frontend','check',this)">检查 Hugo</button><button class="btn primary" onclick="startPublish(this)">生成发布产物</button></div><div id="publish-status" class="status" style="margin-top:8px">尚未发布</div></div>
      </aside>
      <main class="panel">
        <div class="row between"><div><div class="muted small">当前文件</div><div id="current-path" style="font-weight:800;font-size:18px">未选择文章</div></div><div id="editor-status" class="status">等待操作</div></div>
        <div class="grid three" style="margin:12px 0"><input id="meta-title" placeholder="标题"><input id="meta-date" placeholder="日期"><input id="meta-categories" placeholder="分类，逗号分隔"></div>
        <div class="row" style="margin-bottom:12px"><label class="badge"><input id="meta-draft" type="checkbox" style="width:auto;margin-right:6px">草稿</label><label class="badge"><input id="meta-pinned" type="checkbox" style="width:auto;margin-right:6px">置顶</label><button id="save-btn" class="btn primary" onclick="savePost(this)">保存文章</button><button class="btn danger" onclick="deletePost(this)">移入回收站</button><span id="editor-meta" class="status"></span></div>
        <div class="row between" style="margin-bottom:8px">
          <div class="md-toolbar" id="md-toolbar">
            <button class="btn" data-md="h1" title="一级标题">H1</button>
            <button class="btn" data-md="h2" title="二级标题">H2</button>
            <button class="btn" data-md="bold" title="加粗">加粗</button>
            <button class="btn" data-md="italic" title="斜体">斜体</button>
            <button class="btn" data-md="code" title="行内代码">代码</button>
            <button class="btn" data-md="codeblock" title="代码块">代码块</button>
            <button class="btn" data-md="link" title="链接">链接</button>
            <button class="btn" data-md="image" title="图片">图片</button>
            <button class="btn" data-md="quote" title="引用">引用</button>
            <button class="btn" data-md="ul" title="无序列表">列表</button>
            <button class="btn" data-md="hr" title="分割线">分割线</button>
          </div>
          <div class="row" style="gap:8px;flex-wrap:nowrap">
            <div class="seg" id="view-seg"><button data-view="edit" class="active">编辑</button><button data-view="preview">预览</button><button data-view="split">分屏</button></div>
            <label class="badge" title="高亮光标所在行"><input id="toggle-linehl" type="checkbox" style="width:auto;margin-right:6px">行高亮</label>
          </div>
        </div>
        <div class="editor-wrap view-edit" id="editor-wrap">
          <div class="editor-area" id="editor-area"><div id="line-hl"></div><textarea id="editor" placeholder="选择或新建文章后开始编辑。支持 Markdown，Ctrl+S 保存，Tab 缩进两格。"></textarea></div>
          <div id="preview" class="md-body"></div>
        </div>
      </main>
    </section>`
	script := postsPageScript()
	return platformHTML("文章管理", "专注文章列表、新建、编辑、保存、删除和构建；评论审核已拆分到独立页面。", "posts", body, script)
}

func CommentsPageHTML() string {
	body := `
    <section class="panel">
      <div class="row between"><div><h2 class="section-title">评论审核</h2><div class="muted small">默认待审核优先。审核和删除会走服务器权威后端事务，不显示假成功。</div></div><button class="btn primary" onclick="loadComments(this)">刷新评论</button></div>
      <div class="grid three" style="margin-top:12px"><input id="filter-text" placeholder="搜索作者、内容、文章路径"><select id="filter-status" onchange="renderComments()"><option value="pending">待审核优先</option><option value="all">全部</option><option value="approved">已审核</option></select><span id="comment-summary" class="pill">等待刷新</span></div>
      <div class="row" style="margin-top:12px"><label class="badge"><input id="check-all" type="checkbox" style="width:auto;margin-right:6px">全选</label><button class="btn primary" id="batch-approve">批量通过</button><button class="btn danger" id="batch-delete">批量删除</button><span id="batch-info" class="status"></span></div>
    </section>
    <section class="panel">
      <div id="comment-status" class="status">等待操作</div>
      <div id="comment-list" class="list" style="margin-top:12px;max-height:none"><div class="muted">点击刷新加载评论。</div></div>
    </section>`
	script := `
    let comments=[];
    async function loadComments(btn){if(!hasToken()){setText('comment-status','请先在入口页或任意页面登录','error');return}setButton(btn,true,'刷新中...');setText('comment-status','正在连接服务器权威后端...');try{const r=await authFetch('/api/comments?all=1');const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'加载失败');comments=Array.isArray(d.data)?d.data:[];renderComments();setText('comment-status',d.message||'评论已刷新','success');checkShell()}catch(e){setText('comment-status','刷新失败：'+e.message,'error');toast('评论刷新失败：'+e.message,'err')}finally{setButton(btn,false)}}
    function renderComments(){const q=($('filter-text').value||'').toLowerCase();const mode=$('filter-status').value;let list=comments.slice();if(mode==='pending')list.sort((a,b)=>(a.approved?1:0)-(b.approved?1:0));if(mode==='approved')list=list.filter(x=>x.approved);if(q)list=list.filter(x=>[x.author,x.content,x.post_path,x.email,x.ip_address].join(' ').toLowerCase().includes(q));const pending=comments.filter(x=>!x.approved).length;$('comment-summary').textContent='总计 '+comments.length+'，待审 '+pending;if(!list.length){$('comment-list').innerHTML='<div class="muted">没有匹配评论。</div>';updateBatchInfo();return}$('comment-list').innerHTML=list.map(item=>{const id=String(item.id||'');const risk=Array.isArray(item.risk_reasons)&&item.risk_reasons.length?'<details><summary>风控详情</summary><div class="detail-body">'+escapeHtml(item.risk_reasons.join('\n'))+'</div></details>':'';return '<article class="item"><div class="row" style="align-items:flex-start;flex-wrap:nowrap"><input type="checkbox" class="comment-check" data-id="'+escapeHtml(id)+'" style="width:auto;margin-top:4px" aria-label="选择评论"><div style="flex:1;min-width:0"><div class="row between"><div><div class="item-title">'+escapeHtml(item.author||'匿名')+'</div><div class="item-meta"><span>'+escapeHtml(formatBeijingTime(item.timestamp))+'</span><span class="mono">'+escapeHtml(item.post_path||'-')+'</span></div></div><span class="'+(item.approved?'badge ok':'badge warn')+'">'+(item.approved?'已审核':'待审核')+'</span></div><p>'+escapeHtml(item.content||'')+'</p><div class="item-meta"><span class="mono">'+escapeHtml(item.ip_address||'-')+'</span><span>'+escapeHtml(item.email||'-')+'</span></div>'+risk+'<details style="margin-top:10px"><summary>操作链路详情</summary><div class="detail-body" id="detail-'+escapeHtml(id)+'">等待操作。</div></details><div class="row" style="margin-top:10px">'+(item.approved?'<button class="btn" disabled>已通过</button>':'<button class="btn primary comment-action" data-action="approve" data-id="'+escapeHtml(id)+'">审核通过</button>')+'<button class="btn danger comment-action" data-action="delete" data-id="'+escapeHtml(id)+'">删除</button></div></div></div></article>'}).join('');updateBatchInfo()}
    async function mutateComment(id,action,btn){setButton(btn,true,action==='approve'?'审核中...':'删除中...');const detail=$('detail-'+id);if(detail)detail.textContent='连接服务器 -> 执行远端事务 -> 刷新远端结果';try{const path=action==='approve'?'/api/comments/approve?id=':'/api/comments/delete?id=';const r=await authFetch(path+encodeURIComponent(id),{method:'POST'});const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'操作失败');if(detail)detail.textContent=JSON.stringify(d.data||d,null,2);setText('comment-status',action==='approve'?'审核成功':'删除成功','success');toast(action==='approve'?'评论已通过':'评论已删除','ok');await loadComments()}catch(e){if(detail)detail.textContent='失败：'+e.message;setText('comment-status',e.message,'error');toast(e.message,'err')}finally{setButton(btn,false)}}
    function approveComment(id,btn){mutateComment(id,'approve',btn)} function deleteComment(id,btn){if(confirm('确定删除这条评论？'))mutateComment(id,'delete',btn)}
    function selectedIds(){return Array.from(document.querySelectorAll('.comment-check:checked')).map(x=>x.dataset.id)}
    function updateBatchInfo(){const el=$('batch-info');if(el)el.textContent=selectedIds().length?'已选 '+selectedIds().length+' 条':'';const all=$('check-all');if(all)all.checked=false}
    async function batchMutate(action,btn){const ids=selectedIds();if(!ids.length){toast('请先勾选评论','info');return}if(action==='delete'&&!confirm('确定删除选中的 '+ids.length+' 条评论？'))return;setButton(btn,true,'处理中...');setText('comment-status','批量处理中，共 '+ids.length+' 条...');let ok=0,fail=0;for(const id of ids){try{const path=action==='approve'?'/api/comments/approve?id=':'/api/comments/delete?id=';const r=await authFetch(path+encodeURIComponent(id),{method:'POST'});const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'操作失败');ok++}catch(e){fail++}}setButton(btn,false);const msg=(action==='approve'?'批量通过':'批量删除')+'完成：成功 '+ok+' 条'+(fail?'，失败 '+fail+' 条':'');setText('comment-status',msg,fail?'error':'success');toast(msg,fail?'err':'ok');await loadComments()}
    $('comment-list').addEventListener('click',e=>{const btn=e.target.closest('.comment-action');if(!btn)return;const id=btn.dataset.id||'';const action=btn.dataset.action;if(action==='approve')approveComment(id,btn);if(action==='delete')deleteComment(id,btn)});
    $('comment-list').addEventListener('change',e=>{if(e.target.classList.contains('comment-check'))updateBatchInfo()});
    $('check-all').addEventListener('change',e=>{document.querySelectorAll('.comment-check').forEach(c=>c.checked=e.target.checked);const el=$('batch-info');if(el)el.textContent=selectedIds().length?'已选 '+selectedIds().length+' 条':''});
    $('batch-approve').addEventListener('click',e=>batchMutate('approve',e.target));
    $('batch-delete').addEventListener('click',e=>batchMutate('delete',e.target));
    $('filter-text').addEventListener('input',renderComments); window.addEventListener('auth-ready',()=>loadComments()); if(hasToken()) loadComments();`
	return platformHTML("评论审核", "独立处理评论刷新、审核、删除和错误详情，避免混在文章编辑页面里。", "comments", body, script)
}

func postsPageScript() string {
	return `
    const state={posts:[],currentPath:'',currentRevision:'',trash:[],publishTimer:0,dirty:false};
    const BT=String.fromCharCode(96);const FENCE=BT+BT+BT;
    function setPostStatus(msg,kind){setText('editor-status',msg,kind)}
    function fallbackTitle(path){if(!path)return'';const parts=String(path).split(/[\\/]/);const parent=parts.length>1?parts[parts.length-2]:'';const file=parts[parts.length-1]||'';return (parent&&parent!=='content'?parent:file.replace(/\.md$/i,'')).replace(/[-_]+/g,' ')}
    async function loadPosts(btn){if(!hasToken()){setPostStatus('请先登录后再加载文章','error');return}setButton(btn,true,'刷新中...');try{const r=await authFetch('/api/posts');const d=await r.json().catch(()=>({}));if(!r.ok||!d.success){if(invalidAuth(r,d))clearToken();throw new Error(d.message||'无法加载文章')}state.posts=Array.isArray(d.data)?d.data:[];renderPosts();setPostStatus('文章列表已刷新，共 '+state.posts.length+' 篇','success');checkShell()}catch(e){setPostStatus(e.message,'error');toast('文章列表加载失败：'+e.message,'err')}finally{setButton(btn,false)}}
    function renderPosts(){const box=$('post-list');const q=(($('post-search')||{}).value||'').trim().toLowerCase();let list=state.posts;if(q)list=list.filter(p=>((p.title||'')+' '+(p.path||'')).toLowerCase().includes(q));if(!list.length){box.innerHTML='<div class="muted">'+(state.posts.length?'没有匹配的文章。':'暂无文章。')+'</div>';return}box.innerHTML=list.map(p=>{const t=p.updated_at||p.created_at||p.date||'';return '<div class="item '+(p.path===state.currentPath?'active':'')+'" role="button" tabindex="0" data-path="'+encodeURIComponent(p.path||'')+'"><div class="item-title">'+escapeHtml(p.title||fallbackTitle(p.path)||'Untitled')+'</div><div class="item-meta"><span class="mono">'+escapeHtml(p.path||'-')+'</span><span class="'+(p.status==='DRAFT'?'badge warn':'badge ok')+'">'+escapeHtml(p.status||'PUBLISHED')+'</span></div><div class="item-meta" style="margin-top:6px"><span>更新 '+escapeHtml(t?formatBeijingTime(t):'-')+'</span></div></div>'}).join('')}
    function bindPosts(){const box=$('post-list');box.addEventListener('click',e=>{const item=e.target.closest('.item[data-path]');if(item)openPost(decodeURIComponent(item.dataset.path||''))});box.addEventListener('keydown',e=>{if(e.key!=='Enter'&&e.key!==' ')return;const item=e.target.closest('.item[data-path]');if(item){e.preventDefault();openPost(decodeURIComponent(item.dataset.path||''))}})}
    async function openPost(path){if(!path)return;if(state.dirty&&state.currentPath&&state.currentPath!==path&&!confirm('当前文章有未保存的修改，切换后将丢失，确定继续？'))return;state.currentPath=path;state.dirty=false;renderPosts();setPostStatus('正在读取文章...');try{const r=await authFetch('/api/get_content?path='+encodeURIComponent(path));const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'读取失败');const doc=d.data||{};const content=doc.content||'';state.currentRevision=doc.revision||'';$('current-path').textContent=path;$('editor').value=typeof doc.body==='string'?doc.body:bodyOnly(content);applyMetaObject(doc.metadata||parseMeta(content));state.dirty=false;updateEditorMeta();refreshPreviewIfNeeded();setPostStatus(state.currentRevision?'文章已载入，冲突保护已启用':'文章已载入，但服务器未返回 revision','success')}catch(e){setPostStatus(e.message,'error');toast('读取文章失败：'+e.message,'err')}}
    function parseMeta(content){const meta={title:'',date:'',categories:[],draft:false,pinned:false};const m=String(content||'').match(/^---\n([\s\S]*?)\n---\n?/);if(!m)return meta;let current='';m[1].split('\n').forEach(line=>{if(/^\s*-/.test(line)){if(current==='categories'){const v=line.replace(/^\s*-\s*/,'').trim();if(v)meta.categories.push(v)}return}const pair=line.match(/^([A-Za-z0-9_-]+):\s*(.*)$/);if(!pair)return;current=pair[1];const raw=pair[2].trim();if(current==='title')meta.title=raw.replace(/^"|"$/g,'');if(current==='date')meta.date=raw.replace(/^"|"$/g,'');if(current==='draft')meta.draft=raw.toLowerCase()==='true';if(current==='pinned')meta.pinned=raw.toLowerCase()==='true';if(current==='categories'&&raw.startsWith('['))meta.categories=raw.replace(/[\[\]]/g,'').split(',').map(x=>x.trim()).filter(Boolean)});return meta}
    function applyMetaObject(m){m=m||{};$('meta-title').value=m.title||fallbackTitle(state.currentPath);$('meta-date').value=m.date||'';$('meta-categories').value=Array.isArray(m.categories)?m.categories.join(', '):'';$('meta-draft').checked=!!m.draft;$('meta-pinned').checked=!!m.pinned}
    function bodyOnly(content){const m=String(content||'').match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/);return m?m[1]:content}
    function collectMetadata(){return{title:$('meta-title').value.trim()||'Untitled',date:$('meta-date').value.trim(),categories:$('meta-categories').value.split(',').map(x=>x.trim()).filter(Boolean),draft:$('meta-draft').checked,pinned:$('meta-pinned').checked}}
    async function savePost(btn){if(!state.currentPath){setPostStatus('请先选择文章','error');toast('请先选择文章','info');return}setButton(btn,true,'保存中...');setPostStatus('正在保存文章...');try{const payload={path:state.currentPath,body:$('editor').value,metadata:collectMetadata(),revision:state.currentRevision};const r=await authFetch('/api/save_content',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});const d=await r.json().catch(()=>({}));if(r.status===409)throw new Error('文章已被其他操作修改，请重新载入后再保存');if(!r.ok||!d.success)throw new Error(d.message||'保存失败');state.currentRevision=d?.data?.revision||state.currentRevision;if(typeof d?.data?.body==='string')$('editor').value=d.data.body;applyMetaObject(d?.data?.metadata||payload.metadata);state.dirty=false;updateEditorMeta();await loadPosts();setPostStatus('保存成功，未知 front matter 字段已保留','success');toast('文章已保存','ok')}catch(e){setPostStatus(e.message,'error');toast('保存失败：'+e.message,'err')}finally{setButton(btn,false)}}
    async function deletePost(btn){if(!state.currentPath)return;if(!confirm('确定将这篇文章移入回收站？'))return;setButton(btn,true,'处理中...');try{const r=await authFetch('/api/delete_post',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({path:state.currentPath})});const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'移入回收站失败');state.currentPath='';state.currentRevision='';state.dirty=false;$('current-path').textContent='未选择文章';$('editor').value='';updateEditorMeta();refreshPreviewIfNeeded();await Promise.all([loadPosts(),loadTrash()]);setPostStatus('文章已移入回收站','success');toast('文章已移入回收站','ok')}catch(e){setPostStatus(e.message,'error');toast(e.message,'err')}finally{setButton(btn,false)}}
    async function createPost(btn){const title=$('new-title').value.trim();if(!title){setPostStatus('请输入文章标题','error');return}setButton(btn,true,'创建中...');try{const r=await authFetch('/api/create_post',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({title,categories:$('new-categories').value.trim(),draft:true})});const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'创建失败');$('new-title').value='';$('new-categories').value='';await loadPosts();if(d?.data?.path)await openPost(d.data.path);setPostStatus('草稿已创建','success');toast('草稿已创建','ok')}catch(e){setPostStatus(e.message,'error');toast('创建失败：'+e.message,'err')}finally{setButton(btn,false)}}
    async function loadTrash(btn){if(!hasToken())return;setButton(btn,true,'刷新中...');try{const r=await authFetch('/api/trash/posts');const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'无法加载回收站');state.trash=Array.isArray(d.data)?d.data:[];renderTrash()}catch(e){$('trash-list').innerHTML='<div class="status" style="color:var(--danger)">'+escapeHtml(e.message)+'</div>'}finally{setButton(btn,false)}}
    function renderTrash(){const box=$('trash-list');if(!state.trash.length){box.innerHTML='<div class="muted small">回收站为空。</div>';return}box.innerHTML=state.trash.map(x=>'<div class="item"><div class="item-title">'+escapeHtml(x.title||fallbackTitle(x.original_path))+'</div><div class="item-meta"><span>'+escapeHtml(formatBeijingTime(x.deleted_at))+'</span><span class="mono">'+escapeHtml(x.original_path||'')+'</span></div><div class="row" style="margin-top:8px"><button class="btn trash-action" data-action="restore" data-id="'+escapeHtml(x.id)+'">恢复</button><button class="btn danger trash-action" data-action="purge" data-id="'+escapeHtml(x.id)+'">永久删除</button></div></div>').join('')}
    async function trashAction(action,id,btn){if(action==='purge'&&!confirm('永久删除后无法恢复，确定继续？'))return;setButton(btn,true,'处理中...');try{const endpoint=action==='restore'?'/api/restore_post':'/api/purge_post';const r=await authFetch(endpoint,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({id})});const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'操作失败');await Promise.all([loadTrash(),loadPosts()]);setPostStatus(action==='restore'?'文章已恢复':'回收站项目已永久删除','success');toast(action==='restore'?'文章已恢复':'已永久删除','ok')}catch(e){setPostStatus(e.message,'error');toast(e.message,'err')}finally{setButton(btn,false)}}
    async function startPublish(btn){setButton(btn,true,'启动中...');try{const r=await authFetch('/api/publish',{method:'POST'});const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'无法启动发布');renderPublishStatus(d.data);schedulePublishPoll();toast('发布任务已启动','info')}catch(e){setText('publish-status',e.message,'error');toast(e.message,'err')}finally{setButton(btn,false)}}
    async function loadPublishStatus(){if(!hasToken())return;try{const r=await authFetch('/api/publish/status');const d=await r.json().catch(()=>({}));if(r.ok&&d.success){renderPublishStatus(d.data);if(d?.data?.state==='running')schedulePublishPoll()}}catch(e){}}
    function renderPublishStatus(s){s=s||{};const text=s.state==='running'?'正在生成发布产物…':s.state==='succeeded'?'发布产物生成成功':s.state==='failed'?'发布失败：'+(s.error||'未知错误'):'尚未发布';setText('publish-status',text,s.state==='failed'?'error':s.state==='succeeded'?'success':'info')}
    function schedulePublishPoll(){clearTimeout(state.publishTimer);state.publishTimer=setTimeout(async()=>{await loadPublishStatus()},1500)}
    async function runBuild(scope,action,btn){if(!hasToken()){setPostStatus('请先登录','error');return}setButton(btn,true,'执行中...');try{const r=await authFetch('/api/control/command',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({scope,action})});const d=await r.json().catch(()=>({}));if(!r.ok||!d.success)throw new Error(d.message||'命令失败');setPostStatus('Hugo 检查通过','success');toast('Hugo 检查通过','ok')}catch(e){setPostStatus(e.message,'error');toast(e.message,'err')}finally{setButton(btn,false)}}
    function safeUrl(u){const v=String(u||'').trim();if(/^(https?:|mailto:|#|\/|\.\/|\.\.\/)/i.test(v))return v;return'#'}
    function inlineMd(s){let t=escapeHtml(s);const codes=[];const reCode=new RegExp(BT+'([^'+BT+']+)'+BT,'g');t=t.replace(reCode,(m,c)=>{codes.push(c);return '\u0000'+(codes.length-1)+'\u0000'});t=t.replace(/!\[([^\]]*)\]\(([^)\s]+)[^)]*\)/g,(m,a,u)=>'<img alt="'+a+'" src="'+safeUrl(u)+'" loading="lazy" onerror="this.onerror=null;this.removeAttribute(\'src\');this.className=\'img-broken\';this.textContent=\'图片加载失败\'">');t=t.replace(/\[([^\]]+)\]\(([^)\s]+)[^)]*\)/g,(m,txt,u)=>'<a href="'+safeUrl(u)+'" target="_blank" rel="noopener">'+txt+'</a>');t=t.replace(/\*\*([^*]+)\*\*/g,'<strong>$1</strong>').replace(/__([^_]+)__/g,'<strong>$1</strong>');t=t.replace(/\*([^*\n]+)\*/g,'<em>$1</em>').replace(/_([^_\n]+)_/g,'<em>$1</em>');t=t.replace(/\u0000(\d+)\u0000/g,(m,i)=>'<code>'+(codes[+i]||'')+'</code>');return t}
    function renderMarkdown(src){const lines=String(src||'').replace(/\r\n?/g,'\n').split('\n');let html='',inCode=false,codeBuf=[],listTag='',para=[];const flushPara=()=>{if(para.length){html+='<p>'+para.map(inlineMd).join('<br>')+'</p>';para=[]}};const flushList=()=>{if(listTag){html+='</'+listTag+'>';listTag=''}};for(let i=0;i<lines.length;i++){const line=lines[i];if(line.trim().indexOf(FENCE)===0){if(inCode){html+='<pre><code>'+escapeHtml(codeBuf.join('\n'))+'</code></pre>';codeBuf=[];inCode=false}else{flushPara();flushList();inCode=true}continue}if(inCode){codeBuf.push(line);continue}if(!line.trim()){flushPara();flushList();continue}const h=line.match(/^(#{1,6})\s+(.*)$/);if(h){flushPara();flushList();html+='<h'+h[1].length+'>'+inlineMd(h[2])+'</h'+h[1].length+'>';continue}if(/^\s{0,3}(-{3,}|\*{3,}|_{3,})\s*$/.test(line)){flushPara();flushList();html+='<hr>';continue}const bq=line.match(/^\s{0,3}>\s?(.*)$/);if(bq){flushPara();flushList();html+='<blockquote>'+inlineMd(bq[1])+'</blockquote>';continue}const ul=line.match(/^\s*[-*+]\s+(.*)$/);if(ul){flushPara();if(listTag!=='ul'){flushList();html+='<ul>';listTag='ul'}html+='<li>'+inlineMd(ul[1])+'</li>';continue}const ol=line.match(/^\s*\d+[.)]\s+(.*)$/);if(ol){flushPara();if(listTag!=='ol'){flushList();html+='<ol>';listTag='ol'}html+='<li>'+inlineMd(ol[1])+'</li>';continue}flushList();para.push(line)}if(inCode)html+='<pre><code>'+escapeHtml(codeBuf.join('\n'))+'</code></pre>';flushPara();flushList();return html}
    function updateEditorMeta(){const el=$('editor-meta');if(!el)return;const v=$('editor').value;const cjk=(v.match(/[\u4e00-\u9fff\u3040-\u30ff\uac00-\ud7af]/g)||[]).length;const words=(v.replace(/[\u4e00-\u9fff\u3040-\u30ff\uac00-\ud7af]/g,' ').match(/[A-Za-z0-9]+/g)||[]).length;const lines=v?v.split('\n').length:0;el.textContent='字数 '+(cjk+words)+' · 行数 '+lines+(state.dirty?' · 未保存':'')}
    let currentView='edit';let previewTimer=0;
    function refreshPreview(){$('preview').innerHTML=renderMarkdown($('editor').value)}
    function refreshPreviewIfNeeded(){if(currentView!=='edit')refreshPreview()}
    function setView(v){currentView=v;const w=$('editor-wrap');w.classList.remove('view-edit','view-preview','view-split');w.classList.add('view-'+v);document.querySelectorAll('#view-seg [data-view]').forEach(b=>b.classList.toggle('active',b.dataset.view===v));if(v!=='edit')refreshPreview()}
    function updateLineHl(){const area=$('editor-area');const hl=$('line-hl');if(!area||!hl)return;if(!area.classList.contains('hl-on')){hl.style.display='none';return}const ta=$('editor');const pos=ta.selectionStart||0;const line=ta.value.slice(0,pos).split('\n').length-1;const st=getComputedStyle(ta);const lh=parseFloat(st.lineHeight)||23.6;const pt=parseFloat(st.paddingTop)||0;hl.style.display='block';hl.style.top=(pt+line*lh-ta.scrollTop)+'px';hl.style.height=lh+'px'}
    function mdWrap(before,after,placeholder){const ta=$('editor');const s=ta.selectionStart||0;const e=ta.selectionEnd||0;const v=ta.value;const sel=v.slice(s,e)||placeholder||'';ta.value=v.slice(0,s)+before+sel+after+v.slice(e);ta.focus();ta.setSelectionRange(s+before.length,s+before.length+sel.length);ta.dispatchEvent(new Event('input'))}
    function mdLine(prefix){const ta=$('editor');const s=ta.selectionStart||0;const v=ta.value;const ls=v.lastIndexOf('\n',s-1)+1;ta.value=v.slice(0,ls)+prefix+v.slice(ls);ta.focus();ta.setSelectionRange(s+prefix.length,s+prefix.length);ta.dispatchEvent(new Event('input'))}
    $('md-toolbar').addEventListener('click',e=>{const btn=e.target.closest('[data-md]');if(!btn)return;const act=btn.dataset.md;if(act==='h1')mdLine('# ');else if(act==='h2')mdLine('## ');else if(act==='bold')mdWrap('**','**','加粗文字');else if(act==='italic')mdWrap('*','*','斜体文字');else if(act==='code')mdWrap(BT,BT,'代码');else if(act==='codeblock')mdWrap('\n'+FENCE+'\n','\n'+FENCE+'\n','在此粘贴代码');else if(act==='link')mdWrap('[','](https://example.com)','链接文字');else if(act==='image')mdWrap('![','](图片地址)','图片描述');else if(act==='quote')mdLine('> ');else if(act==='ul')mdLine('- ');else if(act==='hr')mdWrap('\n\n---\n\n','','')});
    $('view-seg').addEventListener('click',e=>{const b=e.target.closest('[data-view]');if(b)setView(b.dataset.view)});
    $('toggle-linehl').addEventListener('change',e=>{$('editor-area').classList.toggle('hl-on',e.target.checked);updateLineHl()});
    const editorEl=$('editor');
    editorEl.addEventListener('input',()=>{state.dirty=true;updateEditorMeta();updateLineHl();if(currentView!=='edit'){clearTimeout(previewTimer);previewTimer=setTimeout(refreshPreview,180)}});
    editorEl.addEventListener('keydown',e=>{if((e.ctrlKey||e.metaKey)&&(e.key==='s'||e.key==='S')){e.preventDefault();savePost($('save-btn'));return}if(e.key==='Tab'){e.preventDefault();const ta=e.target;const s=ta.selectionStart||0;ta.value=ta.value.slice(0,s)+'  '+ta.value.slice(ta.selectionEnd||s);ta.setSelectionRange(s+2,s+2);ta.dispatchEvent(new Event('input'))}});
    editorEl.addEventListener('keyup',updateLineHl);editorEl.addEventListener('click',updateLineHl);editorEl.addEventListener('scroll',updateLineHl);
    ['meta-title','meta-date','meta-categories'].forEach(id=>{const el=$(id);if(el)el.addEventListener('input',()=>{state.dirty=true;updateEditorMeta()})});
    ['meta-draft','meta-pinned'].forEach(id=>{const el=$(id);if(el)el.addEventListener('change',()=>{state.dirty=true;updateEditorMeta()})});
    $('post-search').addEventListener('input',renderPosts);
    window.addEventListener('beforeunload',e=>{if(state.dirty){e.preventDefault();e.returnValue=''}});
    $('trash-list').addEventListener('click',e=>{const btn=e.target.closest('.trash-action');if(btn)trashAction(btn.dataset.action,btn.dataset.id,btn)});
    bindPosts();setAuthState();updateEditorMeta();window.addEventListener('auth-ready',()=>Promise.all([loadPosts(),loadTrash(),loadPublishStatus()]));if(hasToken())Promise.all([loadPosts(),loadTrash(),loadPublishStatus()]);`
}
