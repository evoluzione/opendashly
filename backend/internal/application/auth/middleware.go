package auth

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	tenantKey contextKey = "tenant_id"
	userKey   contextKey = "user_id"
	roleKey   contextKey = "role"
)

type MiddlewareOptions struct {
	Mode              string
	CookieName        string
	JWTSecret         []byte
	Repo              Repository
	UserCache         *UserCache
	TenantID          string
	AllowlistPaths    []string
	SessionDuration   time.Duration
	UserLookupTimeout time.Duration
}

// Middleware enforces authentication using the configured auth mode.
func Middleware(opts MiddlewareOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			for _, path := range opts.AllowlistPaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}
			if strings.EqualFold(opts.Mode, "header") {
				tenantID := r.Header.Get("X-Tenant-ID")
				userID := r.Header.Get("X-User-ID")
				if tenantID == "" || userID == "" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), tenantKey, tenantID)
				ctx = context.WithValue(ctx, userKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			cookie, err := r.Cookie(opts.CookieName)
			if err != nil || cookie.Value == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			claims, err := ParseToken(opts.JWTSecret, cookie.Value)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			var user *User
			if cached, ok := opts.UserCache.Get(claims.UserID); ok {
				user = &cached
			} else {
				if opts.Repo == nil {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				lookupCtx := r.Context()
				cancel := func() {}
				if opts.UserLookupTimeout > 0 {
					lookupCtx, cancel = context.WithTimeout(r.Context(), opts.UserLookupTimeout)
				}
				dbUser, err := opts.Repo.GetByID(lookupCtx, claims.UserID)
				cancel()
				if err != nil {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				opts.UserCache.Set(*dbUser)
				user = dbUser
			}
			if user.IsDisabled {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if user.MustChangePassword && !isAllowedDuringPasswordChange(r.URL.Path) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), tenantKey, opts.TenantID)
			ctx = context.WithValue(ctx, userKey, user.ID)
			ctx = context.WithValue(ctx, roleKey, user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

// Role extracts role from context.
func Role(ctx context.Context) string {
	if v, ok := ctx.Value(roleKey).(string); ok {
		return v
	}
	return ""
}

func isAllowedDuringPasswordChange(path string) bool {
	switch path {
	case "/api/auth/logout", "/api/auth/change-password", "/api/auth/session":
		return true
	case "/api/auth/first-login-change-password":
		return true
	default:
		return false
	}
}
