// Package httpapi is the service's HTTP interface: the reports endpoint, the
// management endpoints, and the middleware around them.
package httpapi

import (
	"context"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cruxstack/browser-reporting-api/v2/internal/ingest"
	"github.com/cruxstack/browser-reporting-api/v2/internal/origin"
)

// Ingester ingests one batch of reports from a request body.
type Ingester interface {
	Ingest(ctx context.Context, body io.Reader) (ingest.Result, error)
}

// Config is what the router serves.
type Config struct {
	// BasePath is the path every route is mounted under: "/", or a path
	// with a leading slash and no trailing slash.
	BasePath string
	// MaxBodyBytes limits the size of a reports request body.
	MaxBodyBytes int64
	// Origins decides which origins browsers may send reports from.
	Origins *origin.Matcher
	// Ingester handles the body of each reports request.
	Ingester Ingester
}

// NewRouter returns the handler for every route:
//
//	POST, OPTIONS  {BasePath}/v1/reports
//	GET            {BasePath}/v1/manage/healthz
//	GET            {BasePath}/v1/manage/readyz
func NewRouter(c Config) http.Handler {
	api := chi.NewRouter()
	api.Use(middleware.RequestID, logContext, recoverPanic)

	api.Mount("/v1/reports", newReportsHandler(c).routes())
	api.Mount("/v1/manage", manageRoutes())

	if c.BasePath == "/" {
		return api
	}

	r := chi.NewRouter()
	r.Mount(c.BasePath, api)
	return r
}
