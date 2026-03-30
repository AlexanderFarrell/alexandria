package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"alexandria/api/rest"
	"alexandria/app"
	"alexandria/config"
	"alexandria/infra/epub"
	"alexandria/infra/sqlite"
	"alexandria/infra/storage"
)

func main() {
	cfg := config.Load()

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
	epubParser := epub.New()

	// Application core
	application := app.New(userRepo, bookRepo, progressRepo, listRepo, fileStore, epubParser, cfg)

	// Determine static dir for SPA (client/dist if it exists)
	staticDir := ""
	if _, err := os.Stat("client/dist"); err == nil {
		staticDir = "client/dist"
	}

	// HTTP server
	server := rest.New(application, staticDir, cfg.UploadMaxBytes)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("Alexandria listening on %s", addr)
		if err := server.Start(addr); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")
	if err := server.Shutdown(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("bye")
}
