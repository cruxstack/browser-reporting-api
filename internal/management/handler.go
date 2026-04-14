package management

import (
	"net/http"

	"github.com/cruxstack/browser-reporting-api/internal/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/healthz", h.handleHealthz)
	r.Get("/readyz", h.handleReadyz)
	return r
}

func (h *Handler) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
