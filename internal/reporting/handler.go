package reporting

import (
	"mime"
	"net/http"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cruxstack/browser-reporting-api/internal/domain"
	"github.com/cruxstack/browser-reporting-api/internal/httpx"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the reporting ingestion endpoint.
type Handler struct {
	service      *Service
	maxBodyBytes int64
	origins      *httpx.OriginMatcher
}

func NewHandler(service *Service, maxBodyBytes int64, origins *httpx.OriginMatcher) *Handler {
	return &Handler{service: service, maxBodyBytes: maxBodyBytes, origins: origins}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.MethodFunc(http.MethodPost, "/", h.handleIngest)
	r.MethodFunc(http.MethodOptions, "/", h.handlePreflight)
	return r
}

func (h *Handler) handlePreflight(w http.ResponseWriter, r *http.Request) {
	if !h.setCORSHeaders(w, r) {
		httpx.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleIngest(w http.ResponseWriter, r *http.Request) {
	if !h.setCORSHeaders(w, r) {
		httpx.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
		return
	}

	if err := ensureReportsContentType(r.Header.Get("Content-Type")); err != nil {
		httpx.WriteJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": err.Error()})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBodyBytes)
	result, err := h.service.Ingest(r.Context(), r.Body)
	if err != nil {
		if isPayloadTooLargeErr(err) {
			httpx.WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "payload too large"})
			return
		}

		if isInvalidPayloadErr(err) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to ingest reports"})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) setCORSHeaders(w http.ResponseWriter, r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	allowOrigin, ok := h.origins.AllowHeaderValue(origin)
	if !ok {
		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Vary", "Origin")
	return true
}

func ensureReportsContentType(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("content-type must be application/reports+json or application/csp-report")
	}

	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return errors.New("invalid content-type")
	}

	if mediaType != "application/reports+json" && mediaType != "application/csp-report" {
		return errors.New("content-type must be application/reports+json or application/csp-report")
	}

	return nil
}

func isInvalidPayloadErr(err error) bool {
	if err == nil {
		return false
	}

	var badPayload *domain.BadPayloadError
	return errors.As(err, &badPayload)
}

func isPayloadTooLargeErr(err error) bool {
	if err == nil {
		return false
	}

	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}
