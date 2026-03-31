package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alexandria/api/rest"
	"alexandria/app"
	"alexandria/app/ports"
	"alexandria/config"
	"alexandria/domain"
	"alexandria/infra/epub"
	"alexandria/infra/metadata"
	"alexandria/infra/parser"
	"alexandria/infra/pdf"
	"alexandria/infra/sqlite"
	"alexandria/infra/storage"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// Database
	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	// Infrastructure
	userRepo := sqlite.NewUserRepo(db)
	bookRepo := sqlite.NewBookRepo(db)
	progressRepo := sqlite.NewProgressRepo(db)
	listRepo := sqlite.NewListRepo(db)

	fileStore := storage.NewLocalFileStore(cfg.DataDir)
	bookParser := parser.New(map[domain.FileType]ports.BookParser{
		domain.FileTypeEPUB: epub.New(),
		domain.FileTypePDF:  pdf.New(),
	})

	// Metadata providers (no API keys required — public endpoints)
	metadataProviders := []ports.MetadataProvider{
		metadata.NewGoogleBooks(),
		metadata.NewOpenLibrary(),
	}

	// Application core
	application := app.New(userRepo, bookRepo, progressRepo, listRepo, fileStore, bookParser, metadataProviders, cfg)
	if err := application.Auth.ValidateStartup(context.Background()); err != nil {
		log.Fatalf("invalid startup state: %v", err)
	}

	// Determine static dir for SPA (client/dist if it exists)
	staticDir := ""
	if _, err := os.Stat("client/dist"); err == nil {
		staticDir = "client/dist"
	}

	// HTTP server
	server := rest.New(application, staticDir, cfg, readinessCheck(db, cfg.DataDir))

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	serverErr := make(chan error, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("Alexandria listening on %s", addr)
		serverErr <- server.Start(addr)
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("server stopped: %v", err)
	case <-quit:
	}
	log.Println("shutting down...")
	if err := server.Shutdown(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("bye")
}

func readinessCheck(db interface{ DB() (*sql.DB, error) }, dataDir string) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("get sql db: %w", err)
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			return fmt.Errorf("ping db: %w", err)
		}

		f, err := os.CreateTemp(dataDir, ".readyz-*")
		if err != nil {
			return fmt.Errorf("write data dir: %w", err)
		}
		name := f.Name()
		if err := f.Close(); err != nil {
			return fmt.Errorf("close readiness file: %w", err)
		}
		if err := os.Remove(name); err != nil {
			return fmt.Errorf("remove readiness file: %w", err)
		}
		return nil
	}
}
