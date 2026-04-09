package app

import (
	"alexandria/internal/app/ports"
	"alexandria/internal/app/repos"
	"alexandria/internal/app/services"
	"alexandria/internal/config"
)

// App is the application core — it holds all services and is passed to the API layer.
type App struct {
	Users    *services.UserService
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
		Users:    services.NewUserService(userRepo),
		Auth:     services.NewAuthService(userRepo, cfg),
		Books:    services.NewBookService(bookRepo, store, parser, cfg),
		Reader:   services.NewReaderService(progressRepo, bookRepo, cfg),
		Lists:    services.NewListService(listRepo),
		Metadata: services.NewMetadataService(metadataProviders),
	}
}
