// Package app wires the service together from its configuration and runs
// it. It is the only package that knows about every other one; main calls
// it, and tests use its options to replace the report stream and the clock.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cruxstack/browser-reporting-api/v2/internal/config"
	"github.com/cruxstack/browser-reporting-api/v2/internal/httpapi"
	"github.com/cruxstack/browser-reporting-api/v2/internal/ingest"
	"github.com/cruxstack/browser-reporting-api/v2/internal/origin"
	"github.com/cruxstack/browser-reporting-api/v2/internal/stream"
)

// HTTP server timeouts. Browsers send small batches, so a request that
// takes longer than these is stalled or hostile.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)

// ShutdownTimeout is how long Serve waits for in-flight requests to finish
// once its context is done. Keep the platform's grace period (such as the
// container stop timeout) longer than this.
const ShutdownTimeout = 10 * time.Second

// Option replaces a dependency that New would otherwise build.
type Option func(*deps)

type deps struct {
	reports io.Writer
	now     func() time.Time
}

// WithReportWriter sets where accepted reports are written, instead of
// stdout.
func WithReportWriter(w io.Writer) Option {
	return func(d *deps) { d.reports = w }
}

// WithClock sets the clock that stamps accepted reports.
func WithClock(now func() time.Time) Option {
	return func(d *deps) { d.now = now }
}

// App is the configured service.
type App struct {
	cfg     *config.Config
	handler http.Handler
}

// New builds the service from cfg. Everything is built up front, so
// configuration problems surface at startup.
func New(cfg *config.Config, opts ...Option) (*App, error) {
	d := deps{reports: os.Stdout, now: time.Now}
	for _, opt := range opts {
		opt(&d)
	}

	origins, err := origin.NewMatcher(cfg.Reports.AllowedOrigins)
	if err != nil {
		return nil, fmt.Errorf("allowed origins: %w", err)
	}

	ingester := &ingest.Service{
		Sink: stream.NewNDJSONWriter(d.reports),
		Now:  d.now,
	}

	return &App{
		cfg: cfg,
		handler: httpapi.NewRouter(httpapi.Config{
			BasePath:     cfg.Server.BasePath,
			MaxBodyBytes: cfg.Reports.MaxBodyBytes,
			Origins:      origins,
			Ingester:     ingester,
		}),
	}, nil
}

// Handler returns the service's HTTP handler.
func (a *App) Handler() http.Handler {
	return a.handler
}

// Run listens on the configured address and serves until ctx is done. See
// Serve.
func (a *App) Run(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", a.cfg.Server.ListenAddr)
	if err != nil {
		return err
	}
	return a.Serve(ctx, ln)
}

// Serve serves requests on ln until ctx is done. It then stops accepting
// connections and waits up to ShutdownTimeout for in-flight requests to
// finish. It returns nil after a clean shutdown.
func (a *App) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           a.handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn),
	}

	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	slog.InfoContext(ctx, "listening", "addr", ln.Addr().String(), "base_path", a.cfg.Server.BasePath)

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}

	slog.InfoContext(ctx, "shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("shut down: %w", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.InfoContext(ctx, "stopped")
	return nil
}
