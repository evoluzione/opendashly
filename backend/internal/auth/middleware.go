package auth

import (
	"context"
	"net/http"
)

type contextKey string

const (
	tenantKey contextKey = "tenant_id"
	userKey   contextKey = "user_id"
)

// Middleware enforces a simple auth/tenant model using request headers.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		userID := r.Header.Get("X-User-ID")
		if tenantID == "" || userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), tenantKey, tenantID)
		ctx = context.WithValue(ctx, userKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TenantID extracts tenant id from context.
func TenantID(ctx context.Context) string {
	if v, ok := ctx.Value(tenantKey).(string); ok {
		return v
	}
	return ""
}

// UserID extracts user id from context.
func UserID(ctx context.Context) string {
	if v, ok := ctx.Value(userKey).(string); ok {
		return v
	}
	return ""
}
