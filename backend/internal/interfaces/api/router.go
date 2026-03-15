package api

import (
	"net/http"

	httpapi "opendashly/backend/internal/interfaces/http"
)

type RouterConfig = httpapi.RouterConfig

// NewRouter builds the API router.
func NewRouter(cfg RouterConfig) http.Handler {
	return httpapi.NewRouter(cfg)
}
