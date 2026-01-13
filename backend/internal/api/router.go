package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"opentelemetry-dashboard/backend/internal/auth"
	"opentelemetry-dashboard/backend/internal/api/handlers"
	"opentelemetry-dashboard/backend/internal/query"
)

// NewRouter builds the API router.
func NewRouter(queryService *query.Service, relatedService *query.RelatedService, savedRepo *query.SavedQueryRepo) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(auth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	queryHandler := &handlers.QueryHandler{Service: queryService}
	traceRelated := &handlers.TraceRelatedHandler{Service: relatedService}
	savedHandler := &handlers.SavedQueriesHandler{Repo: savedRepo, Runner: queryService}

	r.Post("/api/query/run", queryHandler.ServeHTTP)
		r.Get("/api/queries", savedHandler.List)
		r.Post("/api/queries", savedHandler.Create)
		r.Get("/api/queries/{queryId}", savedHandler.Get)
		r.Delete("/api/queries/{queryId}", savedHandler.Delete)
		r.Post("/api/queries/{queryId}/run", savedHandler.Run)
		r.Get("/api/traces/{traceId}/related", traceRelated.ServeHTTP)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID, X-User-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
