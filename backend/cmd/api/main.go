package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"opendashly/backend/internal/api"
	"opendashly/backend/internal/api/handlers"
	"opendashly/backend/internal/auth"
	"opendashly/backend/internal/config"
	"opendashly/backend/internal/query"
	"opendashly/backend/internal/retention"
	"opendashly/backend/internal/status"
	"opendashly/backend/internal/storage"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	client, err := storage.NewClient(ctx, cfg.ClickHouseAddr, cfg.ClickHouseUser, cfg.ClickHousePassword)
	if err != nil {
		log.Fatal(err)
	}
	if err := storage.ApplyMigrations(ctx, client.Conn); err != nil {
		log.Fatal(err)
	}

	queryService := &query.Service{Storage: client}
	relatedService := &query.RelatedService{Storage: client}
	traceSpansService := &query.TraceSpansService{Storage: client}
	statusService := &status.Service{Storage: client}
	savedRepo := query.NewSavedQueryRepo()
	authRepo := &auth.Repo{Conn: client.Conn}
	if err := seedDefaultAdmin(ctx, authRepo); err != nil {
		log.Fatal(err)
	}

	retentionRepo := &retention.Repo{Conn: client.Conn}
	cleanupService := &retention.CleanupService{
		Repo: retentionRepo,
		Conn: client.Conn,
	}
	retentionHandler := &handlers.RetentionHandler{
		Repo:    retentionRepo,
		Service: cleanupService,
	}

	scheduler := retention.NewScheduler(cleanupService, cfg.CleanupIntervalMinutes)
	go scheduler.Start(context.Background())

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
		QueryService:      queryService,
		RelatedService:    relatedService,
		TraceSpansService: traceSpansService,
		StatusService:     statusService,
		SavedRepo:         savedRepo,
		AuthHandler:       authHandler,
		UsersHandler:      usersHandler,
		ServicesHandler:   servicesHandler,
		RetentionHandler:  retentionHandler,
		AuthMiddleware:    authMiddleware,
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
