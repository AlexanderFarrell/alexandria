package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"alexandria/internal/api/mcpserver"
	"alexandria/internal/app"
	"alexandria/internal/app/ports"
	"alexandria/internal/config"
	"alexandria/internal/domain"
	"alexandria/internal/infra/epub"
	"alexandria/internal/infra/metadata"
	"alexandria/internal/infra/parser"
	"alexandria/internal/infra/pdf"
	"alexandria/internal/infra/sqlite"
	"alexandria/internal/infra/storage"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	userRepo := sqlite.NewUserRepo(db)
	bookRepo := sqlite.NewBookRepo(db)
	progressRepo := sqlite.NewProgressRepo(db)
	listRepo := sqlite.NewListRepo(db)

	fileStore := storage.NewLocalFileStore(cfg.DataDir)
	bookParser := parser.New(map[domain.FileType]ports.BookParser{
		domain.FileTypeEPUB: epub.New(),
		domain.FileTypePDF:  pdf.New(),
	})

	metadataProviders := []ports.MetadataProvider{
		metadata.NewGoogleBooks(),
		metadata.NewOpenLibrary(),
	}

	application := app.New(userRepo, bookRepo, progressRepo, listRepo, fileStore, bookParser, metadataProviders, cfg)
	if err := application.Auth.ValidateStartup(context.Background()); err != nil {
		log.Fatalf("invalid startup state: %v", err)
	}

	server := mcpserver.New(application, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, &sdkmcp.StdioTransport{}); err != nil && err != context.Canceled {
		log.Fatalf("mcp server stopped: %v", err)
	}
}
