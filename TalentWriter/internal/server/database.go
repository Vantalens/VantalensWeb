package server

import (
	"context"
	"errors"
	"log"

	"vantalens/talentwriter/internal/analytics"
	"vantalens/talentwriter/internal/article"
	"vantalens/talentwriter/internal/comment"
	"vantalens/talentwriter/internal/dbsync"
	"vantalens/talentwriter/internal/handlers"
)

func InitializeDatabases(ctx context.Context, hugoPath string) (*dbsync.Service, error) {
	cfg := dbsync.DefaultConfig(hugoPath)
	var service *dbsync.Service
	if cfg.Enabled {
		service = dbsync.NewServiceWithHooks(cfg, nil, dbsync.Hooks{
			BeforeReplace: closeDatabases,
			AfterReplace: func() error {
				return initializeDatabaseConnections(hugoPath, false)
			},
		})
		handlers.SetSyncService(service)
		if err := initializeDatabaseConnections(hugoPath, true); err != nil {
			return nil, err
		}
		service.Start(ctx)
		go func() {
			status := service.RunOnce(ctx)
			logSyncStatus(status)
		}()
		return service, nil
	}

	handlers.SetSyncService(dbsync.NewService(cfg, nil))
	if err := initializeDatabaseConnections(hugoPath, true); err != nil {
		return nil, err
	}
	log.Println("[DBSYNC] disabled; using local databases")
	return nil, nil
}

func initializeDatabaseConnections(hugoPath string, syncArticlesFromDisk bool) error {
	if err := analytics.Init(hugoPath); err != nil {
		return err
	}
	if err := comment.Init(hugoPath); err != nil {
		return err
	}
	if err := article.Init(hugoPath); err != nil {
		return err
	}
	if syncArticlesFromDisk {
		if posts, err := handlers.SyncArticlesToDatabase(); err != nil {
			log.Printf("[ARTICLES] Initial sync skipped: %v", err)
		} else {
			log.Printf("[ARTICLES] Initial sync completed: %d posts", len(posts))
		}
	}
	return nil
}

func closeDatabases() error {
	var result error
	if err := analytics.Close(); err != nil {
		result = errors.Join(result, err)
	}
	if err := comment.Close(); err != nil {
		result = errors.Join(result, err)
	}
	if err := article.Close(); err != nil {
		result = errors.Join(result, err)
	}
	return result
}

func logSyncStatus(status dbsync.Status) {
	if !status.Enabled {
		log.Println("[DBSYNC] disabled")
		return
	}
	for _, db := range status.Databases {
		if db.LastError != "" {
			log.Printf("[DBSYNC] %s sync failed: %s", db.Name, db.LastError)
			continue
		}
		if db.Success {
			log.Printf("[DBSYNC] %s synced: %d bytes", db.Name, db.SizeBytes)
		}
	}
}
