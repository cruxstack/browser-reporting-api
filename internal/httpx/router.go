package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(basePath string, reports, management chi.Router) http.Handler {
	api := chi.NewRouter()
	api.Use(middleware.RequestID)
	api.Use(middleware.RealIP)
	api.Use(middleware.Recoverer)

	api.Mount("/v1/reports", reports)
	api.Mount("/v1/manage", management)

	if basePath == "/" {
		return api
	}

	r := chi.NewRouter()
	r.Mount(basePath, api)
	return r
}
