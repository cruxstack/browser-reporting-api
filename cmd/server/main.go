package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cruxstack/browser-reporting-api/internal/config"
	"github.com/cruxstack/browser-reporting-api/internal/httpx"
	"github.com/cruxstack/browser-reporting-api/internal/management"
	"github.com/cruxstack/browser-reporting-api/internal/parser"
	"github.com/cruxstack/browser-reporting-api/internal/reporting"
	"github.com/cruxstack/browser-reporting-api/internal/stream"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	writer := stream.NewNDJSONWriter(os.Stdout)
	reportsParser := parser.NewJSONBatchParser()
	reportsService := reporting.NewService(reportsParser, writer)
	originMatcher, err := httpx.NewOriginMatcher(cfg.AllowedOrigins)
	if err != nil {
		slog.Error("build origin matcher", "error", err)
		os.Exit(1)
	}

	reportsHandler := reporting.NewHandler(reportsService, cfg.MaxBodyBytes, originMatcher)
	managementHandler := management.NewHandler()
	handler := httpx.NewRouter(cfg.BasePath, reportsHandler.Routes(), managementHandler.Routes())

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr, "base_path", cfg.BasePath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		if closeErr := srv.Close(); closeErr != nil {
			slog.Error("force close failed", "error", closeErr)
		}
	}
}
