package bootstrap

import (
	"context"
	"math"
	"net/http"
	"time"

	"opendashly/backend/internal/application/ai"
	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/dashboard"
	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/pressure"
	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/application/retention"
	"opendashly/backend/internal/application/status"
	"opendashly/backend/internal/application/tuning"
	"opendashly/backend/internal/application/workspace"
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

	maintenanceClient, err := storage.NewClientWithOptions(ctx, maintenanceClickHouseOptions(cfg))
	if err != nil {
		return nil, err
	}
	if err := storage.ApplyMigrations(ctx, maintenanceClient.Conn); err != nil {
		return nil, err
	}
	controlClient, err := storage.NewClientWithOptions(ctx, controlClickHouseOptions(cfg))
	if err != nil {
		return nil, err
	}
	telemetryClient, err := storage.NewClientWithOptions(ctx, telemetryClickHouseOptions(cfg))
	if err != nil {
		return nil, err
	}
	telemetryLimiter := pressure.NewLimiter(cfg.TelemetryQueryConcurrency)

	queryService := &query.Service{Storage: telemetryClient, Debug: cfg.DebugQuery, Limiter: telemetryLimiter}
	relatedService := &query.RelatedService{Storage: telemetryClient}
	traceSpansService := &query.TraceSpansService{Storage: telemetryClient}
	statusService := &status.Service{Storage: controlClient, MaintenanceStorage: maintenanceClient, CollectorHealthURL: cfg.CollectorHealthURL}
	dashboardService := &metrics.Service{Storage: telemetryClient, MaintenanceStore: maintenanceClient, Limiter: telemetryLimiter}
	dashboardService.FreshCacheTTL = time.Duration(cfg.DashboardFreshCacheTTLSec) * time.Second
	dashboardService.StaleCacheTTL = time.Duration(cfg.DashboardStaleCacheTTLSec) * time.Second
	dashboardService.LastGoodCacheTTL = time.Duration(cfg.DashboardLastGoodTTLSec) * time.Second
	dashboardService.QueryParallelism = cfg.DashboardQueryParallelism
	dashboardService.HalveOnOOM = cfg.DashboardHalveOnOOM
	dashboardService.RawFallback = cfg.DashboardRawFallback
	dashboardService.PressureCooldown = time.Duration(cfg.DashboardPressureCooldownSec) * time.Second
	if cfg.DashboardRollupBackfill {
		dashboardService.StartRollupBackfill(context.Background(), cfg.DashboardRollupBackfillHours)
		statusService.StartRollupBackfill(context.Background(), cfg.DashboardRollupBackfillHours)
	}
	savedRepo := query.NewSavedQueryRepo()
	authRepo := &auth.Repo{Conn: controlClient.Conn}
	if err := seedDefaultAdmin(ctx, authRepo); err != nil {
		return nil, err
	}

	retentionRepo := &retention.Repo{Conn: maintenanceClient.Conn}
	defaultLogsRetention := boundedDefaultRetentionDays(7, uint32(cfg.MaxLogRetentionDays))
	defaultTracesRetention := boundedDefaultRetentionDays(7, uint32(cfg.MaxTraceRetentionDays))
	if err := retentionRepo.EnsureSetting(ctx, "logs", defaultLogsRetention, "system"); err != nil {
		return nil, err
	}
	if err := retentionRepo.EnsureSetting(ctx, "traces", defaultTracesRetention, "system"); err != nil {
		return nil, err
	}
	if err := clampRetentionSetting(ctx, retentionRepo, "logs", uint32(cfg.MaxLogRetentionDays)); err != nil {
		return nil, err
	}
	if err := clampRetentionSetting(ctx, retentionRepo, "traces", uint32(cfg.MaxTraceRetentionDays)); err != nil {
		return nil, err
	}
	adaptiveOptions := retention.AdaptiveRetentionOptions{
		Enabled:                             cfg.RetentionAdaptiveEnabled,
		StepDownDays:                        uint32(cfg.RetentionStepDownDays),
		MaxLevel:                            cfg.RetentionMaxLevel,
		MinTraceRetentionDays:               hoursToDaysCeil(cfg.RetentionMinTraceHours),
		MinLogRetentionDays:                 hoursToDaysCeil(cfg.RetentionMinLogHours),
		PressureCooldown:                    time.Duration(cfg.RetentionPressureCooldownSec) * time.Second,
		PressureMinActiveSignals:            cfg.RetentionPressureMinSignals,
		PressureErrorWindow:                 time.Duration(cfg.RetentionPressureWindowSec) * time.Second,
		PressureErrorThreshold:              cfg.RetentionPressureErrorCount,
		PressureMemoryThresholdPercent:      cfg.RetentionPressureMemPct,
		PressureMemoryBudgetMiB:             cfg.RetentionPressureMemBudgetMB,
		PressureClickHouseDiskThresholdPerc: cfg.RetentionPressureDiskPct,
	}
	pressureMonitor := retention.NewPressureMonitor(maintenanceClient.Conn, adaptiveOptions)

	// Replace the old static machine profiles with a runtime self-tuning loop:
	// it starts the load-sensitive knobs at their safe floor and adapts them to
	// real pressure (AIMD), keeping the shared telemetry limiter resized live.
	tuningController := tuning.New(tuning.DefaultBounds(), func(ctx context.Context) bool {
		return pressureMonitor.Snapshot(ctx).Pressure
	}, telemetryLimiter)
	dashboardService.Tuner = tuningController
	go tuningController.Run(context.Background())

	cleanupService := &retention.CleanupService{
		Repo:              retentionRepo,
		Conn:              maintenanceClient.Conn,
		EnableCountBefore: cfg.RetentionPreCount,
		AdaptiveOptions:   adaptiveOptions,
		PressureMonitor:   pressureMonitor,
	}
	cleanupService.EnsureDefaults()
	queryService.PressureObserver = cleanupService
	retentionHandler := &handlers.RetentionHandler{
		Repo:                  retentionRepo,
		Service:               cleanupService,
		MaxLogRetentionDays:   uint32(cfg.MaxLogRetentionDays),
		MaxTraceRetentionDays: uint32(cfg.MaxTraceRetentionDays),
	}

	aiRepo := &ai.Repo{Conn: controlClient.Conn}
	aiService := &ai.Service{Repo: aiRepo}
	dashboardSettingsRepo := &dashboard.Repo{Conn: controlClient.Conn}
	dashboardSettingsService := &dashboard.Service{Repo: dashboardSettingsRepo}
	workspaceRepo := &workspace.Repo{Conn: controlClient.Conn}
	workspaceService := &workspace.Service{Repo: workspaceRepo}

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
		Mode:              cfg.AuthMode,
		CookieName:        cfg.AuthCookieName,
		JWTSecret:         []byte(cfg.AuthSecret),
		Repo:              authRepo,
		UserCache:         auth.NewUserCache(time.Minute),
		TenantID:          "default",
		AllowlistPaths:    []string{"/healthz", "/api/auth/login"},
		SessionDuration:   24 * time.Hour,
		UserLookupTimeout: 750 * time.Millisecond,
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
		WorkspaceSettings:  workspaceService,
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

func baseClickHouseOptions(cfg *config.Config) storage.ClientOptions {
	return storage.ClientOptions{
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
		MaxThreads:                    1,
	}
}

func telemetryClickHouseOptions(cfg *config.Config) storage.ClientOptions {
	opts := baseClickHouseOptions(cfg)
	opts.MaxThreads = 1
	return opts
}

func controlClickHouseOptions(cfg *config.Config) storage.ClientOptions {
	opts := baseClickHouseOptions(cfg)
	opts.MaxOpenConns = minPositiveInt(cfg.ClickHouseMaxOpenConns, 2)
	opts.MaxIdleConns = 1
	opts.ReadTimeout = 5 * time.Second
	opts.MaxMemoryUsageBytes = minPositiveInt(cfg.ClickHouseMaxMemoryMiB, 64) * 1024 * 1024
	opts.MaxBytesBeforeExternalGroupBy = 16 * 1024 * 1024
	opts.MaxBytesBeforeExternalSort = 16 * 1024 * 1024
	opts.MaxExecutionTimeSec = 3
	opts.MaxThreads = 1
	return opts
}

func maintenanceClickHouseOptions(cfg *config.Config) storage.ClientOptions {
	opts := baseClickHouseOptions(cfg)
	opts.MaxOpenConns = 1
	opts.MaxIdleConns = 1
	opts.MaxThreads = 1
	return opts
}

func minPositiveInt(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
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

func hoursToDaysCeil(hours int) uint32 {
	if hours <= 0 {
		return 1
	}
	return uint32(math.Ceil(float64(hours) / 24.0))
}

func boundedDefaultRetentionDays(defaultDays uint32, maxDays uint32) uint32 {
	if maxDays == 0 {
		return defaultDays
	}
	if defaultDays > maxDays {
		return maxDays
	}
	return defaultDays
}

func clampRetentionSetting(ctx context.Context, repo *retention.Repo, signalType string, maxDays uint32) error {
	if maxDays == 0 {
		return nil
	}
	setting, err := repo.GetSettingBySignal(ctx, signalType)
	if err != nil {
		return err
	}
	if setting.RetentionDays <= maxDays {
		return nil
	}
	return repo.UpdateSetting(ctx, signalType, maxDays, "system")
}
