package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"opentelemetry-dashboard/backend/internal/api/handlers"
	"opentelemetry-dashboard/backend/internal/query"
)

type RouterConfig struct {
	QueryService    *query.Service
	RelatedService  *query.RelatedService
	SavedRepo       *query.SavedQueryRepo
	AuthHandler     *handlers.AuthHandler
	UsersHandler    *handlers.UsersHandler
	ServicesHandler *handlers.ServicesHandler
	AuthMiddleware  func(http.Handler) http.Handler
}

// NewRouter builds the API router.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	if cfg.AuthMiddleware != nil {
		r.Use(cfg.AuthMiddleware)
	}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	queryHandler := &handlers.QueryHandler{Service: cfg.QueryService}
	traceRelated := &handlers.TraceRelatedHandler{Service: cfg.RelatedService}
	savedHandler := &handlers.SavedQueriesHandler{Repo: cfg.SavedRepo, Runner: cfg.QueryService}

	r.Post("/api/query/run", queryHandler.ServeHTTP)
	r.Get("/api/queries", savedHandler.List)
	r.Post("/api/queries", savedHandler.Create)
	r.Get("/api/queries/{queryId}", savedHandler.Get)
	r.Delete("/api/queries/{queryId}", savedHandler.Delete)
	r.Post("/api/queries/{queryId}/run", savedHandler.Run)
	r.Get("/api/traces/{traceId}/related", traceRelated.ServeHTTP)

	if cfg.AuthHandler != nil {
		r.Post("/api/auth/login", cfg.AuthHandler.Login)
		r.Post("/api/auth/logout", cfg.AuthHandler.Logout)
		r.Post("/api/auth/change-password", cfg.AuthHandler.ChangePassword)
		r.Get("/api/auth/session", cfg.AuthHandler.Session)
	}
	if cfg.UsersHandler != nil {
		r.Get("/api/users", cfg.UsersHandler.List)
		r.Post("/api/users", cfg.UsersHandler.Create)
	}
	if cfg.ServicesHandler != nil {
		r.Get("/api/services", cfg.ServicesHandler.List)
	}

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
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
