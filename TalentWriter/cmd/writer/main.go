package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

const Version = "2.1.0"

func main() {
	config.LoadEnvFiles(".env", "../.env")

	hugoPath := config.ResolveHugoPath(config.GetEnv("HUGO_PATH", "."))
	adminToken := config.GetEnvAny([]string{"ADMIN_TOKEN", "ADMIN_PASSWORD"}, "")
	controlPort := parsePort(config.GetEnv("CONTROL_PORT", "9090"), 9090)
	port := parsePort(config.GetEnv("WRITER_PORT", "9091"), 9091)

	cfg := &config.Config{
		HugoPath:     hugoPath,
		LauncherMode: "writer",
		AdminToken:   adminToken,
		ControlPort:  controlPort,
		WriterPort:   port,
	}
	config.SetConfig(cfg)

	auth.InitJWTSecret()
	if err := analytics.Init(hugoPath); err != nil {
		log.Fatalf("[ANALYTICS] init failed: %v", err)
	}
	if err := comment.Init(hugoPath); err != nil {
		log.Fatalf("[COMMENTS] init failed: %v", err)
	}
	if err := article.Init(hugoPath); err != nil {
		log.Fatalf("[ARTICLES] init failed: %v", err)
	}
	if posts, err := handlers.SyncArticlesToDatabase(); err != nil {
		log.Printf("[ARTICLES] initial sync skipped: %v", err)
	} else {
		log.Printf("[ARTICLES] initial sync completed: %d posts", len(posts))
	}
	email.StartWorkers()

	addr := fmt.Sprintf(":%d", port)
	mux := server.BuildMux(server.ModeWriter, Version)
	srv := &http.Server{
		Addr:              addr,
		Handler:           server.WithSecurityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	log.Printf("[WRITER] mode=writer, addr=%s, hugo_path=%s", addr, hugoPath)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var parentExitCh <-chan struct{}
	if config.GetEnv("WS_PARENT_WATCH", "") == "1" {
		parentExitCh = server.StartParentWatcher(os.Getppid(), 2*time.Second)
		log.Println("[WRITER] parent watcher enabled")
	}

	select {
	case <-sigCh:
		log.Println("[WRITER] shutdown signal received")
	case <-parentExitCh:
		log.Println("[WRITER] parent process exited, shutting down")
	case err := <-errCh:
		log.Fatalf("[WRITER] listen failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[WRITER] graceful shutdown error: %v", err)
	}
}

func parsePort(raw string, def int) int {
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 {
		return def
	}
	return port
}
