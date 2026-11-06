// Package config reads the configuration from environment variables. Invalid
// or contradictory settings are errors, so a misconfigured server fails at
// startup instead of rejecting or misrouting reports.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/cruxstack/browser-reporting-api/v2/internal/origin"
)

// Defaults for unset variables.
const (
	DefaultListenAddr   = ":8080"
	DefaultBasePath     = "/"
	DefaultMaxBodyBytes = 1 << 20 // 1 MiB
)

// Config is the service's configuration.
type Config struct {
	LogLevel slog.Level

	Server  Server
	Reports Reports
}

// Server configures the HTTP server.
type Server struct {
	// ListenAddr is the TCP address to listen on, such as ":8080".
	ListenAddr string
	// BasePath is the path every route is mounted under: "/", or a path
	// with a leading slash and no trailing slash.
	BasePath string
}

// Reports configures the reports endpoint.
type Reports struct {
	// MaxBodyBytes limits the size of a request body.
	MaxBodyBytes int64
	// AllowedOrigins are the origins browsers may send reports from: "*",
	// origins, or origins with wildcards. See origin.Validate.
	AllowedOrigins []string
}

// removed lists variables from v1 that are no longer read, with what to use
// instead. Setting one is an error so an old configuration can't silently
// fall back to a default, such as allowing every origin.
var removed = map[string]string{
	"LISTEN_ADDR":             "APP_LISTEN_ADDR",
	"BASE_PATH":               "APP_BASE_PATH",
	"MAX_BODY_BYTES":          "APP_MAX_BODY_BYTES",
	"REPORTS_ALLOWED_ORIGINS": "APP_REPORTS_ALLOWED_ORIGINS",
	"REPORTS_ALLOWED_ORIGIN":  "APP_REPORTS_ALLOWED_ORIGINS",
}

// FromEnv reads the configuration from the process environment.
func FromEnv() (*Config, error) {
	return Load(os.LookupEnv)
}

// Load reads the configuration with lookup, which has the signature of
// os.LookupEnv. All problems are reported together.
func Load(lookup func(string) (string, bool)) (*Config, error) {
	r := reader{lookup: lookup}

	for _, key := range slices.Sorted(maps.Keys(removed)) {
		if r.str(key) != "" {
			r.fail(fmt.Errorf("%s was removed in v2: use %s", key, removed[key]))
		}
	}

	cfg := &Config{
		LogLevel: r.level("APP_LOG_LEVEL", slog.LevelInfo),
		Server: Server{
			ListenAddr: r.strOr("APP_LISTEN_ADDR", DefaultListenAddr),
			BasePath:   normalizeBasePath(r.strOr("APP_BASE_PATH", DefaultBasePath)),
		},
		Reports: Reports{
			MaxBodyBytes:   r.positiveInt("APP_MAX_BODY_BYTES", DefaultMaxBodyBytes),
			AllowedOrigins: r.list("APP_REPORTS_ALLOWED_ORIGINS", []string{origin.Any}),
		},
	}

	r.errs = append(r.errs, cfg.validate()...)
	if len(r.errs) > 0 {
		return nil, fmt.Errorf("invalid configuration:\n  %w", joinIndented(r.errs))
	}
	return cfg, nil
}

// basePathSegment is one segment of a base path: unreserved URL characters
// only, so it can't hold route patterns, escapes, or query strings.
var basePathSegment = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

func (c *Config) validate() []error {
	var errs []error

	if _, _, err := net.SplitHostPort(c.Server.ListenAddr); err != nil {
		errs = append(errs, fmt.Errorf("APP_LISTEN_ADDR: %q must be host:port or :port, such as %s", c.Server.ListenAddr, DefaultListenAddr))
	}

	if p := c.Server.BasePath; p != "/" {
		for seg := range strings.SplitSeq(strings.TrimPrefix(p, "/"), "/") {
			if !basePathSegment.MatchString(seg) || seg == "." || seg == ".." {
				errs = append(errs, fmt.Errorf("APP_BASE_PATH: %q must be a path such as /reports, with letters, digits, and . _ ~ - in each segment", p))
				break
			}
		}
	}

	origins := c.Reports.AllowedOrigins
	for _, o := range origins {
		if err := origin.Validate(o); err != nil {
			errs = append(errs, fmt.Errorf("APP_REPORTS_ALLOWED_ORIGINS: %w", err))
		}
	}
	if len(origins) > 1 && slices.Contains(origins, origin.Any) {
		errs = append(errs, fmt.Errorf("APP_REPORTS_ALLOWED_ORIGINS: %s allows every origin, so it can't be combined with other origins", origin.Any))
	}

	return errs
}

// normalizeBasePath adds a leading slash and removes trailing slashes, so
// "collector/" becomes "/collector". An empty path is "/".
func normalizeBasePath(p string) string {
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// reader parses environment variables and collects errors.
type reader struct {
	lookup func(string) (string, bool)
	errs   []error
}

func (r *reader) fail(err error) {
	r.errs = append(r.errs, err)
}

// str returns the trimmed value of key, or "" when unset.
func (r *reader) str(key string) string {
	v, _ := r.lookup(key)
	return strings.TrimSpace(v)
}

// strOr returns the value of key, or def when unset or empty.
func (r *reader) strOr(key, def string) string {
	if v := r.str(key); v != "" {
		return v
	}
	return def
}

// positiveInt parses an integer greater than zero. It returns def when the
// variable is unset.
func (r *reader) positiveInt(key string, def int64) int64 {
	s := r.str(key)
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v <= 0 {
		r.fail(fmt.Errorf("%s: %q must be a whole number greater than 0", key, s))
		return def
	}
	return v
}

func (r *reader) level(key string, def slog.Level) slog.Level {
	s := r.str(key)
	if s == "" {
		return def
	}
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	r.fail(fmt.Errorf("%s: %q must be one of debug, info, warn, or error", key, s))
	return def
}

// list splits a comma-separated value into trimmed items. It returns def
// when the variable is unset or has no items.
func (r *reader) list(key string, def []string) []string {
	var out []string
	for item := range strings.SplitSeq(r.str(key), ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}

// joinIndented joins errors one per line for a readable startup error.
func joinIndented(errs []error) error {
	msgs := make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = err.Error()
	}
	return errors.New(strings.Join(msgs, "\n  "))
}
