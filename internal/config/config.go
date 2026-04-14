package config

import (
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
)

const (
	defaultListenAddr  = ":8080"
	defaultBasePath    = "/"
	defaultMaxBodySize = 1 << 20
	defaultCORSOrigins = "*"
)

type Config struct {
	ListenAddr     string
	BasePath       string
	MaxBodyBytes   int64
	AllowedOrigins []string
}

func LoadFromEnv() (Config, error) {
	reportsAllowedOrigins := strings.TrimSpace(os.Getenv("REPORTS_ALLOWED_ORIGINS"))
	legacyReportsAllowedOrigin := strings.TrimSpace(os.Getenv("REPORTS_ALLOWED_ORIGIN"))
	if reportsAllowedOrigins == "" && legacyReportsAllowedOrigin != "" {
		slog.Warn("REPORTS_ALLOWED_ORIGIN is deprecated, use REPORTS_ALLOWED_ORIGINS")
	}

	originsRaw := reportsAllowedOrigins
	if originsRaw == "" {
		originsRaw = valueOrDefault("REPORTS_ALLOWED_ORIGIN", defaultCORSOrigins)
	}

	cfg := Config{
		ListenAddr:     valueOrDefault("LISTEN_ADDR", defaultListenAddr),
		BasePath:       normalizeBasePath(valueOrDefault("BASE_PATH", defaultBasePath)),
		AllowedOrigins: parseList(originsRaw),
	}

	if err := validateAllowedOrigins(cfg.AllowedOrigins); err != nil {
		return Config{}, err
	}

	maxBody := valueOrDefault("MAX_BODY_BYTES", strconv.Itoa(defaultMaxBodySize))
	maxBodyInt, err := strconv.ParseInt(maxBody, 10, 64)
	if err != nil || maxBodyInt <= 0 {
		return Config{}, errors.Newf("invalid MAX_BODY_BYTES: %q", maxBody)
	}
	cfg.MaxBodyBytes = maxBodyInt

	return cfg, nil
}

var wildcardPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://.*\*.*$`)

func validateAllowedOrigins(origins []string) error {
	if len(origins) == 0 {
		return errors.New("REPORTS_ALLOWED_ORIGINS cannot be empty")
	}

	for _, origin := range origins {
		if origin == "*" {
			continue
		}

		if strings.Contains(origin, "*") {
			if !wildcardPattern.MatchString(origin) {
				return errors.Newf("invalid wildcard origin pattern: %q", origin)
			}
			continue
		}

		if !strings.Contains(origin, "://") {
			return errors.Newf("invalid origin %q: must include scheme", origin)
		}

		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errors.Newf("invalid origin %q: must be scheme://host[:port]", origin)
		}

		if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return errors.Newf("invalid origin %q: must be scheme://host[:port]", origin)
		}
	}

	return nil
}

func valueOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func parseList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func normalizeBasePath(basePath string) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" || basePath == "/" {
		return "/"
	}

	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	for strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimSuffix(basePath, "/")
	}

	if basePath == "" {
		return "/"
	}

	return basePath
}
