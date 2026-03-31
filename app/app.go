package app

import (
	"alexandria/app/ports"
	"alexandria/app/repos"
	"alexandria/app/services"
	"alexandria/config"
)

// App is the application core — it holds all services and is passed to the API layer.
type App struct {
	Auth     *services.AuthService
	Books    *services.BookService
	Reader   *services.ReaderService
	Lists    *services.ListService
	Metadata *services.MetadataService
}

// New wires together all services with their dependencies.
func New(
	userRepo repos.UserRepo,
	bookRepo repos.BookRepo,
	progressRepo repos.ProgressRepo,
	listRepo repos.ListRepo,
	store ports.FileStore,
	parser ports.BookParser,
	metadataProviders []ports.MetadataProvider,
	cfg *config.Config,
) *App {
	return &App{
		Auth:     services.NewAuthService(userRepo, cfg),
		Books:    services.NewBookService(bookRepo, store, parser, cfg),
		Reader:   services.NewReaderService(progressRepo),
		Lists:    services.NewListService(listRepo),
		Metadata: services.NewMetadataService(metadataProviders),
	}
}
