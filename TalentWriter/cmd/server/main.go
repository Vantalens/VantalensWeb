package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"vantalens/talentwriter/internal/analytics"
	"vantalens/talentwriter/internal/article"
	"vantalens/talentwriter/internal/auth"
	"vantalens/talentwriter/internal/comment"
	"vantalens/talentwriter/internal/config"
	"vantalens/talentwriter/internal/email"
	"vantalens/talentwriter/internal/handlers"
	"vantalens/talentwriter/internal/server"
)

const Version = "2.0.0"

func main() {
	printBanner()
	config.LoadEnvFiles(".env", "../.env")

	hugoPath := config.ResolveHugoPath(config.GetEnv("HUGO_PATH", "."))
	adminToken := config.GetEnvAny([]string{"ADMIN_TOKEN", "ADMIN_PASSWORD"}, "")

	log.Printf("[CONFIG] HUGO_PATH: %s", hugoPath)
	log.Printf("[CONFIG] ADMIN_TOKEN: %s", maskToken(adminToken))

	cfg := &config.Config{
		HugoPath:     hugoPath,
		LauncherMode: "all",
		AdminToken:   adminToken,
		ControlPort:  parsePort(config.GetEnv("CONTROL_PORT", strconv.Itoa(config.Port)), config.Port),
		WriterPort:   parsePort(config.GetEnv("WRITER_PORT", "9091"), 9091),
	}
	config.SetConfig(cfg)

	auth.InitJWTSecret()
	log.Println("[AUTH] JWT secret initialized")

	if err := analytics.Init(hugoPath); err != nil {
		log.Fatalf("[ANALYTICS] Init failed: %v", err)
	}
	log.Println("[ANALYTICS] SQLite initialized")

	if err := comment.Init(hugoPath); err != nil {
		log.Fatalf("[COMMENTS] Init failed: %v", err)
	}
	log.Println("[COMMENTS] SQLite initialized")

	if err := article.Init(hugoPath); err != nil {
		log.Fatalf("[ARTICLES] Init failed: %v", err)
	}
	log.Println("[ARTICLES] SQLite initialized")
	if posts, err := handlers.SyncArticlesToDatabase(); err != nil {
		log.Printf("[ARTICLES] Initial sync skipped: %v", err)
	} else {
		log.Printf("[ARTICLES] Initial sync completed: %d posts", len(posts))
	}

	email.StartWorkers()
	log.Println("[EMAIL] Workers started")

	mux := server.BuildMux(server.ModeAll, Version)
	log.Println("[HTTP] Routes registered (mode=all)")

	host := config.GetEnv("HTTP_HOST", "127.0.0.1")
	port := parsePort(config.GetEnv("HTTP_PORT", strconv.Itoa(config.Port)), config.Port)
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[SERVER] Starting on %s", addr)

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.WithSecurityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[SERVER] Error: %v", err)
		}
	}()

	if config.GetEnv("TALENTWRITER_AUTO_OPEN_BROWSER", "true") != "false" {
		go openBrowserWhenReady(fmt.Sprintf("http://127.0.0.1:%d/platform/backend", port))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[SERVER] Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[SERVER] Graceful shutdown failed: %v", err)
	}
}

func parsePort(raw string, def int) int {
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 {
		return def
	}
	return port
}

func printBanner() {
	fmt.Println("================================================")
	fmt.Println("          Vantalens Writer v" + Version)
	fmt.Println("      Blog Comment Management System")
	fmt.Println("================================================")
}

func maskToken(token string) string {
	if len(token) == 0 {
		return "(not set)"
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

func openBrowserWhenReady(url string) {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	openBrowser(url)
}

func openBrowser(url string) {
	if _, err := os.Stat(filepath.Join(os.Getenv("WINDIR"), "System32", "cmd.exe")); err == nil {
		_ = execCommand("cmd", "/c", "start", "", url)
		return
	}
	_ = execCommand("rundll32", "url.dll,FileProtocolHandler", url)
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}

const dashboardHTML = `<!doctype html>
<html lang="zh-CN">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Vantalens Writer 后台</title>
	<style>
		:root {
			color-scheme: dark;
			--bg: #0b1020;
			--panel: rgba(15, 23, 42, 0.86);
			--panel-strong: rgba(30, 41, 59, 0.92);
			--text: #e5eefc;
			--muted: #8da2c0;
			--accent: #5eead4;
			--accent-2: #60a5fa;
			--border: rgba(148, 163, 184, 0.18);
			--shadow: 0 18px 60px rgba(2, 6, 23, 0.45);
		}
		* { box-sizing: border-box; }
		body {
			margin: 0;
			min-height: 100vh;
			font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
			color: var(--text);
			background:
				radial-gradient(circle at top left, rgba(96, 165, 250, 0.24), transparent 30%),
				radial-gradient(circle at 85% 15%, rgba(94, 234, 212, 0.16), transparent 28%),
				linear-gradient(135deg, #050816 0%, #0b1020 46%, #101935 100%);
		}
		.wrap { max-width: 1180px; margin: 0 auto; padding: 40px 20px 56px; }
		.hero {
			display: grid;
			gap: 18px;
			grid-template-columns: minmax(0, 1.3fr) minmax(320px, 0.7fr);
			align-items: start;
			margin-bottom: 22px;
		}
		.panel {
			background: var(--panel);
			border: 1px solid var(--border);
			border-radius: 24px;
			box-shadow: var(--shadow);
			backdrop-filter: blur(18px);
		}
		.hero-main { padding: 32px; }
		.eyebrow {
			display: inline-flex;
			align-items: center;
			gap: 10px;
			padding: 8px 12px;
			border-radius: 999px;
			background: rgba(96, 165, 250, 0.14);
			color: #d7e9ff;
			font-size: 13px;
			letter-spacing: 0.08em;
			text-transform: uppercase;
		}
		h1 { margin: 16px 0 12px; font-size: clamp(34px, 5vw, 58px); line-height: 1.02; }
		.lead { margin: 0; color: var(--muted); font-size: 16px; line-height: 1.7; max-width: 760px; }
		.stats {
			display: grid;
			grid-template-columns: repeat(3, minmax(0, 1fr));
			gap: 14px;
			margin-top: 24px;
		}
		.stat {
			padding: 18px;
			border-radius: 18px;
			background: var(--panel-strong);
			border: 1px solid var(--border);
		}
		.stat .label { display: block; color: var(--muted); font-size: 12px; margin-bottom: 6px; }
		.stat .value { font-size: 18px; font-weight: 700; }
		.side { padding: 24px; }
		.status {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 12px;
			padding: 16px 18px;
			border-radius: 18px;
			background: rgba(15, 23, 42, 0.72);
			border: 1px solid var(--border);
			margin-bottom: 14px;
		}
		.dot { width: 12px; height: 12px; border-radius: 50%; background: #f59e0b; box-shadow: 0 0 18px rgba(245, 158, 11, 0.75); }
		.status.ok .dot { background: #34d399; box-shadow: 0 0 18px rgba(52, 211, 153, 0.75); }
		.status-title { margin: 0; font-weight: 700; }
		.status-sub { margin: 4px 0 0; color: var(--muted); font-size: 13px; }
		.actions { display: grid; gap: 10px; margin-top: 14px; }
		.btn {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			padding: 12px 14px;
			border-radius: 14px;
			border: 1px solid var(--border);
			color: var(--text);
			text-decoration: none;
			background: rgba(96, 165, 250, 0.08);
		}
		.grid { display: grid; gap: 18px; grid-template-columns: repeat(2, minmax(0, 1fr)); }
		.card { padding: 24px; }
		.card h2 { margin: 0 0 10px; font-size: 18px; }
		.card p { margin: 0 0 14px; color: var(--muted); }
		.endpoint { display: flex; justify-content: space-between; gap: 12px; padding: 12px 14px; border-radius: 14px; background: rgba(15, 23, 42, 0.65); border: 1px solid var(--border); margin-top: 10px; }
		code { color: #b9f7eb; font-size: 13px; }
		.footer { margin-top: 18px; color: var(--muted); font-size: 13px; }
		@media (max-width: 900px) {
			.hero, .grid, .stats { grid-template-columns: 1fr; }
			.hero-main, .side, .card { padding: 20px; }
		}
	</style>
</head>
<body>
	<div class="wrap">
		<section class="hero">
			<div class="panel hero-main">
				<div class="eyebrow">Vantalens Writer v2.0.0</div>
				<h1>后台管理面板</h1>
				<p class="lead">这是评论管理与站点配置的入口页面。你可以在这里检查服务状态、确认 API 可用性，并进入登录与设置相关接口。</p>
				<div class="stats">
					<div class="stat"><span class="label">服务状态</span><span class="value" id="service-status">正在检查</span></div>
					<div class="stat"><span class="label">健康检查</span><span class="value" id="health-version">-</span></div>
					<div class="stat"><span class="label">当前版本</span><span class="value">2.0.0</span></div>
				</div>
			</div>
			<aside class="panel side">
				<div class="status" id="status-box">
					<div>
						<p class="status-title">后端连接</p>
						<p class="status-sub" id="status-detail">等待健康检查响应</p>
					</div>
					<div class="dot"></div>
				</div>
				<div class="actions">
					<a class="btn" href="/health" target="_blank" rel="noreferrer">查看健康检查</a>
					<a class="btn" href="/api?format=json" target="_blank" rel="noreferrer">查看 API 说明</a>
				</div>
				<div class="footer">如果你是从浏览器打开这里，这一页现在应该是后台页面，而不是纯 JSON。</div>
			</aside>
		</section>
		<section class="grid">
			<div class="panel card">
				<h2>核心接口</h2>
				<p>这些接口由后端提供，用于登录、评论管理和站点设置。</p>
				<div class="endpoint"><code>/api/login</code><span>登录</span></div>
				<div class="endpoint"><code>/api/comments</code><span>读取评论</span></div>
				<div class="endpoint"><code>/api/settings</code><span>读取设置</span></div>
			</div>
			<div class="panel card">
				<h2>管理操作</h2>
				<p>这些接口需要管理员身份，前端可以接上你的登录态后调用。</p>
				<div class="endpoint"><code>/api/comments/approve</code><span>审核</span></div>
				<div class="endpoint"><code>/api/comments/delete</code><span>删除</span></div>
				<div class="endpoint"><code>/api/settings/save</code><span>保存配置</span></div>
			</div>
		</section>
	</div>
	<script>
		(async function () {
			const statusBox = document.getElementById('status-box');
			const statusDetail = document.getElementById('status-detail');
			const serviceStatus = document.getElementById('service-status');
			const healthVersion = document.getElementById('health-version');
			try {
				const response = await fetch('/health', { cache: 'no-store' });
				const data = await response.json();
				const ok = response.ok && data.status === 'ok';
				statusBox.classList.toggle('ok', ok);
				serviceStatus.textContent = ok ? '在线' : '异常';
				statusDetail.textContent = ok ? '服务已响应，API 可继续使用' : '健康检查返回异常';
				healthVersion.textContent = data.version || '-';
			} catch (error) {
				statusBox.classList.remove('ok');
				serviceStatus.textContent = '离线';
				statusDetail.textContent = '无法连接到 /health';
				healthVersion.textContent = '-';
			}
		})();
	</script>
</body>
</html>`
