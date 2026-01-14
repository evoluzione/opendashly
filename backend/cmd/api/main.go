package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"opentelemetry-dashboard/backend/internal/api"
	"opentelemetry-dashboard/backend/internal/api/handlers"
	"opentelemetry-dashboard/backend/internal/auth"
	"opentelemetry-dashboard/backend/internal/config"
	"opentelemetry-dashboard/backend/internal/query"
	"opentelemetry-dashboard/backend/internal/storage"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	client, err := storage.NewClient(ctx, cfg.ClickHouseAddr)
	if err != nil {
		log.Fatal(err)
	}
	if err := storage.ApplyMigrations(ctx, client.Conn); err != nil {
		log.Fatal(err)
	}

	queryService := &query.Service{Storage: client}
	relatedService := &query.RelatedService{Storage: client}
	savedRepo := query.NewSavedQueryRepo()
	authRepo := &auth.Repo{Conn: client.Conn}
	if err := seedDefaultAdmin(ctx, authRepo); err != nil {
		log.Fatal(err)
	}

	authHandler := &handlers.AuthHandler{
		Repo:       authRepo,
		Secret:     []byte(cfg.AuthSecret),
		CookieName: cfg.AuthCookieName,
		SessionTTL: 24 * time.Hour,
		TenantID:   "default",
	}
	usersHandler := &handlers.UsersHandler{Repo: authRepo}
	servicesHandler := &handlers.ServicesHandler{Service: queryService}
	authMiddleware := auth.Middleware(auth.MiddlewareOptions{
		Mode:           cfg.AuthMode,
		CookieName:     cfg.AuthCookieName,
		JWTSecret:      []byte(cfg.AuthSecret),
		Repo:           authRepo,
		TenantID:       "default",
		AllowlistPaths: []string{"/healthz", "/api/auth/login"},
		SessionDuration: 24 * time.Hour,
	})

	handler := api.NewRouter(api.RouterConfig{
		QueryService:    queryService,
		RelatedService:  relatedService,
		SavedRepo:       savedRepo,
		AuthHandler:     authHandler,
		UsersHandler:    usersHandler,
		ServicesHandler: servicesHandler,
		AuthMiddleware:  authMiddleware,
	})
	log.Printf("listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, handler))
}

func seedDefaultAdmin(ctx context.Context, repo *auth.Repo) error {
	_, err := repo.GetByUsername(ctx, "admin")
	if err == nil {
		return nil
	}
	hash, err := auth.HashPassword("admin")
	if err != nil {
		return err
	}
	_, err = repo.Create(ctx, auth.User{
		Username:           "admin",
		PasswordHash:       hash,
		Role:               auth.RoleAdmin,
		MustChangePassword: true,
		IsDisabled:         false,
	})
	return err
}
