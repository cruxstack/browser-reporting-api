package httpapi

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/cruxstack/browser-reporting-api/v2/internal/ingest"
	"github.com/cruxstack/browser-reporting-api/v2/internal/origin"
	"github.com/cruxstack/browser-reporting-api/v2/internal/report"
)

// reportsHandler serves the reports endpoint.
type reportsHandler struct {
	ingester     Ingester
	maxBodyBytes int64
	origins      *origin.Matcher
}

func newReportsHandler(c Config) *reportsHandler {
	return &reportsHandler{ingester: c.Ingester, maxBodyBytes: c.MaxBodyBytes, origins: c.Origins}
}

func (h *reportsHandler) routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.handleIngest)
	r.Options("/", h.handlePreflight)
	return r
}

func (h *reportsHandler) handlePreflight(w http.ResponseWriter, r *http.Request) {
	if !h.setCORSHeaders(w, r) {
		writeError(w, http.StatusForbidden, "origin not allowed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *reportsHandler) handleIngest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !h.setCORSHeaders(w, r) {
		slog.DebugContext(ctx, "origin not allowed", "origin", r.Header.Get("Origin"))
		writeError(w, http.StatusForbidden, "origin not allowed")
		return
	}

	if err := checkContentType(r.Header.Get("Content-Type")); err != nil {
		writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBodyBytes)
	result, err := h.ingester.Ingest(ctx, r.Body)

	var tooLarge *http.MaxBytesError
	var badPayload *ingest.BadPayloadError
	switch {
	case err == nil:
		slog.DebugContext(ctx, "ingested reports", "received", result.Received, "accepted", result.Accepted, "rejected", result.Rejected)
		writeJSON(w, http.StatusOK, result)
	case errors.As(err, &tooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "payload too large")
	case errors.As(err, &badPayload):
		slog.DebugContext(ctx, "bad payload", "error", err)
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		slog.ErrorContext(ctx, "failed to ingest reports", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to ingest reports")
	}
}

// setCORSHeaders sets the CORS response headers for a request from an
// allowed origin. It reports false for an origin that isn't allowed. A
// request without an Origin header isn't a cross-origin browser request, so
// it needs no headers.
func (h *reportsHandler) setCORSHeaders(w http.ResponseWriter, r *http.Request) bool {
	requestOrigin := strings.TrimSpace(r.Header.Get("Origin"))
	if requestOrigin == "" {
		return true
	}

	allowOrigin, ok := h.origins.Allow(requestOrigin)
	if !ok {
		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Vary", "Origin")
	return true
}

var errContentType = errors.New("content-type must be " + report.ContentTypeReports + " or " + report.ContentTypeCSPReport)

func checkContentType(value string) error {
	if strings.TrimSpace(value) == "" {
		return errContentType
	}

	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return errors.New("invalid content-type")
	}

	if mediaType != report.ContentTypeReports && mediaType != report.ContentTypeCSPReport {
		return errContentType
	}

	return nil
}
