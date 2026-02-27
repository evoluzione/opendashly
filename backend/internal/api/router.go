package api

import (
	"net/http"

	"opendashly/backend/internal/ai"
	"opendashly/backend/internal/api/handlers"
	"opendashly/backend/internal/config"
	"opendashly/backend/internal/dashboard"
	"opendashly/backend/internal/metrics"
	"opendashly/backend/internal/query"
	"opendashly/backend/internal/status"

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

	queryHandler := &handlers.QueryHandler{Service: cfg.QueryService}
	traceRelated := &handlers.TraceRelatedHandler{Service: cfg.RelatedService}
	traceSpans := &handlers.TraceSpansHandler{Service: cfg.TraceSpansService}
	statusHandler := &handlers.StatusHandler{Service: cfg.StatusService}
	savedHandler := &handlers.SavedQueriesHandler{Repo: cfg.SavedRepo, Runner: cfg.QueryService}
	smartQuery := &handlers.SmartQueryHandler{AIService: cfg.AIService}
	aiAvailability := &handlers.AIAvailabilityHandler{Service: cfg.AIService}
	aiAssistantChat := &handlers.AIAssistantChatHandler{
		AIService:    cfg.AIService,
		QueryService: cfg.QueryService,
	}
	aiAssistantSession := &handlers.AIAssistantSessionHandler{AIService: cfg.AIService}
	dashboardHandler := &handlers.DashboardHandler{Service: cfg.DashboardService}
	aiHandler := &handlers.AISettingsHandler{Service: cfg.AIService}
	dashboardSettingsHandler := &handlers.DashboardSettingsHandler{Service: cfg.DashboardSettings}

	r.Post("/api/query/run", queryHandler.ServeHTTP)
	r.Post("/api/query/smart", smartQuery.ServeHTTP)
	r.Get("/api/ai/availability", aiAvailability.Get)
	r.Post("/api/ai/assistant/chat", aiAssistantChat.ServeHTTP)
	r.Get("/api/ai/assistant/session", aiAssistantSession.Get)
	r.Put("/api/ai/assistant/session", aiAssistantSession.Put)
	r.Delete("/api/ai/assistant/session", aiAssistantSession.Delete)
	// New attributes endpoint
	attributesHandler := &handlers.AttributesHandler{Service: cfg.QueryService}
	r.Get("/api/query/attributes", attributesHandler.ServeHTTP)

	r.Get("/api/queries", savedHandler.List)
	r.Post("/api/queries", savedHandler.Create)
	r.Get("/api/queries/{queryId}", savedHandler.Get)
	r.Delete("/api/queries/{queryId}", savedHandler.Delete)
	r.Post("/api/queries/{queryId}/run", savedHandler.Run)
	r.Get("/api/traces/{traceId}/related", traceRelated.ServeHTTP)
	r.Get("/api/traces/{traceId}/spans", traceSpans.ServeHTTP)
	r.Get("/api/status/summary", statusHandler.ServeHTTP)
	r.Post("/api/dashboard/metrics", dashboardHandler.ServeHTTP)
	r.Get("/api/dashboard/settings", dashboardSettingsHandler.Get)

	// Admin Settings
	r.Get("/api/admin/ai/settings", aiHandler.Get)
	r.Put("/api/admin/ai/settings", aiHandler.Update)
	r.Get("/api/admin/dashboard/settings", dashboardSettingsHandler.Get)
	r.Put("/api/admin/dashboard/settings", dashboardSettingsHandler.Update)

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

	return r
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
