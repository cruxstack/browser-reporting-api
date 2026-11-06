package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// manageRoutes serves the health endpoints. readyz is the place for
// readiness checks of dependencies the service gains, such as a database.
func manageRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	return r
}
