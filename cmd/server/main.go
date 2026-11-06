// Command server runs browser-reporting-api: it receives browser reports
// over HTTP and writes each accepted report to stdout as one JSON line. Its
// own logs go to stderr.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/cruxstack/browser-reporting-api/v2/internal/app"
	"github.com/cruxstack/browser-reporting-api/v2/internal/config"
	"github.com/cruxstack/browser-reporting-api/v2/internal/logging"
)

// version is set at build time with -ldflags "-X main.version=v2.0.0".
var version = "dev"

func main() {
	var level slog.LevelVar
	slog.SetDefault(slog.New(logging.NewJSONHandler(os.Stderr, &level)))

	if err := run(&level); err != nil {
		slog.Error("server failed", "version", version, "error", err)
		os.Exit(1)
	}
}

func run(level *slog.LevelVar) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	level.Set(cfg.LogLevel)

	a, err := app.New(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.InfoContext(ctx, "starting",
		"version", version,
		"revision", revision(),
		"log_level", cfg.LogLevel.String(),
		"max_body_bytes", cfg.Reports.MaxBodyBytes,
		"allowed_origins", cfg.Reports.AllowedOrigins,
	)
	return a.Run(ctx)
}

// revision returns the VCS revision the binary was built from, if known.
func revision() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				return s.Value
			}
		}
	}
	return "unknown"
}
