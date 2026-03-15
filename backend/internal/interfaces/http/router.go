package http

import (
	"net/http"

	"opendashly/backend/internal/application/ai"
	"opendashly/backend/internal/application/dashboard"
	"opendashly/backend/internal/application/metrics"
	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/application/status"
	"opendashly/backend/internal/infrastructure/config"
	handlers "opendashly/backend/internal/interfaces/http/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterConfig struct {
	Config             *config.Config
	QueryService       *query.Service
	RelatedService     *query.RelatedService
	TraceSpansService  *query.TraceSpansService
	SavedRepo          *query.SavedQueryRepo
	StatusService      *status.Service
	DashboardService   *metrics.Service
	AIService          *ai.Service
	DashboardSettings  *dashboard.Service
	AuthHandler        *handlers.AuthHandler
	UsersHandler       *handlers.UsersHandler
	ServicesHandler    *handlers.ServicesHandler
	RetentionHandler   *handlers.RetentionHandler
	AuthMiddleware     func(http.Handler) http.Handler
	CORSAllowedOrigins []string
}

type routeHandlers struct {
	queryHandler           *handlers.QueryHandler
	traceRelatedHandler    *handlers.TraceRelatedHandler
	traceSpansHandler      *handlers.TraceSpansHandler
	statusHandler          *handlers.StatusHandler
	savedQueriesHandler    *handlers.SavedQueriesHandler
	smartQueryHandler      *handlers.SmartQueryHandler
	aiAvailabilityHandler  *handlers.AIAvailabilityHandler
	aiAssistantChatHandler *handlers.AIAssistantChatHandler
	aiAssistantSession     *handlers.AIAssistantSessionHandler
	dashboardHandler       *handlers.DashboardHandler
	aiSettingsHandler      *handlers.AISettingsHandler
	attributesHandler      *handlers.AttributesHandler
	dashboardSettings      *handlers.DashboardSettingsHandler
}

// NewRouter builds the API router.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware(cfg.CORSAllowedOrigins))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	if cfg.AuthMiddleware != nil {
		r.Use(cfg.AuthMiddleware)
	}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	routeHandlers := buildRouteHandlers(cfg)
	registerAPIRoutes(r, cfg, routeHandlers)

	return r
}

func buildRouteHandlers(cfg RouterConfig) routeHandlers {
	return routeHandlers{
		queryHandler:          &handlers.QueryHandler{Service: cfg.QueryService},
		traceRelatedHandler:   &handlers.TraceRelatedHandler{Service: cfg.RelatedService},
		traceSpansHandler:     &handlers.TraceSpansHandler{Service: cfg.TraceSpansService},
		statusHandler:         &handlers.StatusHandler{Service: cfg.StatusService},
		savedQueriesHandler:   &handlers.SavedQueriesHandler{Repo: cfg.SavedRepo, Runner: cfg.QueryService},
		smartQueryHandler:     &handlers.SmartQueryHandler{AIService: cfg.AIService},
		aiAvailabilityHandler: &handlers.AIAvailabilityHandler{Service: cfg.AIService},
		aiAssistantChatHandler: &handlers.AIAssistantChatHandler{
			AIService:    cfg.AIService,
			QueryService: cfg.QueryService,
		},
		aiAssistantSession: &handlers.AIAssistantSessionHandler{AIService: cfg.AIService},
		dashboardHandler:   &handlers.DashboardHandler{Service: cfg.DashboardService},
		aiSettingsHandler:  &handlers.AISettingsHandler{Service: cfg.AIService},
		attributesHandler:  &handlers.AttributesHandler{Service: cfg.QueryService},
		dashboardSettings:  &handlers.DashboardSettingsHandler{Service: cfg.DashboardSettings},
	}
}

func registerAPIRoutes(r *chi.Mux, cfg RouterConfig, h routeHandlers) {
	r.Post("/api/query/run", h.queryHandler.ServeHTTP)
	r.Post("/api/query/smart", h.smartQueryHandler.ServeHTTP)
	r.Get("/api/ai/availability", h.aiAvailabilityHandler.Get)
	r.Post("/api/ai/assistant/chat", h.aiAssistantChatHandler.ServeHTTP)
	r.Get("/api/ai/assistant/session", h.aiAssistantSession.Get)
	r.Put("/api/ai/assistant/session", h.aiAssistantSession.Put)
	r.Delete("/api/ai/assistant/session", h.aiAssistantSession.Delete)
	r.Get("/api/query/attributes", h.attributesHandler.ServeHTTP)

	r.Get("/api/queries", h.savedQueriesHandler.List)
	r.Post("/api/queries", h.savedQueriesHandler.Create)
	r.Get("/api/queries/{queryId}", h.savedQueriesHandler.Get)
	r.Delete("/api/queries/{queryId}", h.savedQueriesHandler.Delete)
	r.Post("/api/queries/{queryId}/run", h.savedQueriesHandler.Run)
	r.Get("/api/traces/{traceId}/related", h.traceRelatedHandler.ServeHTTP)
	r.Get("/api/traces/{traceId}/spans", h.traceSpansHandler.ServeHTTP)
	r.Get("/api/status/summary", h.statusHandler.ServeHTTP)
	r.Get("/api/status/runtime", h.statusHandler.ServeRuntimeHTTP)
	r.Post("/api/dashboard/metrics", h.dashboardHandler.ServeHTTP)
	r.Get("/api/dashboard/settings", h.dashboardSettings.Get)

	// Admin Settings
	r.Get("/api/admin/ai/settings", h.aiSettingsHandler.Get)
	r.Put("/api/admin/ai/settings", h.aiSettingsHandler.Update)
	r.Get("/api/admin/dashboard/settings", h.dashboardSettings.Get)
	r.Put("/api/admin/dashboard/settings", h.dashboardSettings.Update)

	registerOptionalRoutes(r, cfg)
}

func registerOptionalRoutes(r *chi.Mux, cfg RouterConfig) {
	if cfg.AuthHandler != nil {
		r.Post("/api/auth/login", cfg.AuthHandler.Login)
		r.Post("/api/auth/logout", cfg.AuthHandler.Logout)
		r.Post("/api/auth/change-password", cfg.AuthHandler.ChangePassword)
		r.Post("/api/auth/first-login-change-password", cfg.AuthHandler.FirstLoginChangePassword)
		r.Get("/api/auth/session", cfg.AuthHandler.Session)
	}
	if cfg.UsersHandler != nil {
		r.Get("/api/users", cfg.UsersHandler.List)
		r.Post("/api/users", cfg.UsersHandler.Create)
	}
	if cfg.ServicesHandler != nil {
		r.Get("/api/services", cfg.ServicesHandler.List)
	}
	if cfg.RetentionHandler != nil {
		r.Get("/api/admin/retention/settings", cfg.RetentionHandler.GetSettings)
		r.Put("/api/admin/retention/settings", cfg.RetentionHandler.UpdateSettings)
		r.Post("/api/admin/retention/cleanup", cfg.RetentionHandler.ManualCleanup)
		r.Get("/api/admin/retention/jobs", cfg.RetentionHandler.ListJobs)
	}
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			continue
		}
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || isAllowedOrigin(allowed, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID, X-User-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(allowed map[string]struct{}, origin string) bool {
	_, ok := allowed[origin]
	return ok
}
