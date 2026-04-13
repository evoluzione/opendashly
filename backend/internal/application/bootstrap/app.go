package bootstrap

import (
	"context"
	"net/http"
	"time"

	"opendashly/backend/internal/application/ai"
	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/dashboard"
	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/application/retention"
	"opendashly/backend/internal/application/status"
	"opendashly/backend/internal/infrastructure/config"
	"opendashly/backend/internal/infrastructure/storage"
	httpapi "opendashly/backend/internal/interfaces/http"
	"opendashly/backend/internal/interfaces/http/handlers"
)

// App groups runtime dependencies needed by the API entrypoint.
type App struct {
	ListenAddr string
	Handler    http.Handler
}

// Build initializes services, repositories and HTTP routes.
func Build(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClientWithOptions(ctx, storage.ClientOptions{
		DSN:                           cfg.ClickHouseAddr,
		User:                          cfg.ClickHouseUser,
		Password:                      cfg.ClickHousePassword,
		MaxOpenConns:                  cfg.ClickHouseMaxOpenConns,
		MaxIdleConns:                  cfg.ClickHouseMaxIdleConns,
		DialTimeout:                   time.Duration(cfg.ClickHouseDialTimeout) * time.Second,
		ReadTimeout:                   time.Duration(cfg.ClickHouseReadTimeout) * time.Second,
		MaxMemoryUsageBytes:           cfg.ClickHouseMaxMemoryMiB * 1024 * 1024,
		MaxBytesBeforeExternalGroupBy: cfg.ClickHouseExternalGroupByMiB * 1024 * 1024,
		MaxBytesBeforeExternalSort:    cfg.ClickHouseExternalSortMiB * 1024 * 1024,
		MaxTempDataOnDiskBytes:        cfg.ClickHouseTempDiskMiB * 1024 * 1024,
		MaxExecutionTimeSec:           cfg.ClickHouseMaxExecSec,
	})
	if err != nil {
		return nil, err
	}
	if err := storage.ApplyMigrations(ctx, client.Conn); err != nil {
		return nil, err
	}

	queryService := &query.Service{Storage: client, Debug: cfg.DebugQuery}
	relatedService := &query.RelatedService{Storage: client}
	traceSpansService := &query.TraceSpansService{Storage: client}
	statusService := &status.Service{Storage: client, CollectorHealthURL: cfg.CollectorHealthURL}
	dashboardService := &metrics.Service{Storage: client}
	savedRepo := query.NewSavedQueryRepo()
	authRepo := &auth.Repo{Conn: client.Conn}
	if err := seedDefaultAdmin(ctx, authRepo); err != nil {
		return nil, err
	}

	retentionRepo := &retention.Repo{Conn: client.Conn}
	cleanupService := &retention.CleanupService{
		Repo:              retentionRepo,
		Conn:              client.Conn,
		EnableCountBefore: cfg.RetentionPreCount,
	}
	retentionHandler := &handlers.RetentionHandler{
		Repo:    retentionRepo,
		Service: cleanupService,
	}

	aiRepo := &ai.Repo{Conn: client.Conn}
	aiService := &ai.Service{Repo: aiRepo}
	dashboardSettingsRepo := &dashboard.Repo{Conn: client.Conn}
	dashboardSettingsService := &dashboard.Service{Repo: dashboardSettingsRepo}

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
	servicesHandler := &handlers.ServicesHandler{
		Service: queryService,
		Timeout: time.Duration(cfg.ServiceListTimeoutSec) * time.Second,
	}
	authMiddleware := auth.Middleware(auth.MiddlewareOptions{
		Mode:            cfg.AuthMode,
		CookieName:      cfg.AuthCookieName,
		JWTSecret:       []byte(cfg.AuthSecret),
		Repo:            authRepo,
		TenantID:        "default",
		AllowlistPaths:  []string{"/healthz", "/api/auth/login"},
		SessionDuration: 24 * time.Hour,
	})

	handler := httpapi.NewRouter(httpapi.RouterConfig{
		Config:             cfg,
		QueryService:       queryService,
		RelatedService:     relatedService,
		TraceSpansService:  traceSpansService,
		StatusService:      statusService,
		DashboardService:   dashboardService,
		AIService:          aiService,
		DashboardSettings:  dashboardSettingsService,
		SavedRepo:          savedRepo,
		AuthHandler:        authHandler,
		UsersHandler:       usersHandler,
		ServicesHandler:    servicesHandler,
		RetentionHandler:   retentionHandler,
		AuthMiddleware:     authMiddleware,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
	})

	return &App{ListenAddr: cfg.ListenAddr, Handler: handler}, nil
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
