// Package config loads all runtime configuration from environment variables. It is the only
// place the API reads the environment, which keeps it portable between a local process and
// AWS Lambda (where SST injects the same variables).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// "development" or "production".
	Env string
	// Public origin of the web app, e.g. http://localhost:3000. Drives the OAuth redirect URL and
	// whether cookies are Secure.
	PublicURL *url.URL
	// Listen address for the local server (ignored on Lambda).
	Addr string
	// Path prefix stripped from incoming requests when present (gateway stage or CloudFront path).
	APIBasePath string

	DatabaseURL string
	DBMaxConns  int32

	GoogleClientID     string
	GoogleClientSecret string
	// Lower-cased emails allowed to sign in.
	AllowedEmails map[string]struct{}
	SessionTTL    time.Duration
	// Dev-only email login for end-to-end tests. Only honoured when Env is development.
	DevLogin bool

	LogLevel slog.Level
}

func (c Config) IsDev() bool { return c.Env == "development" }

// DevLoginEnabled reports whether the test-only login endpoint may be served.
func (c Config) DevLoginEnabled() bool { return c.IsDev() && c.DevLogin }

// SecureCookies is true unless the app is served over plain http (local development).
func (c Config) SecureCookies() bool { return c.PublicURL.Scheme == "https" }

// OAuthRedirectURL is where Google sends the user back to. It goes through the web app's origin
// (Next.js rewrites /auth/* to the API), so the session cookie is first-party.
func (c Config) OAuthRedirectURL() string {
	return strings.TrimRight(c.PublicURL.String(), "/") + "/auth/google/callback"
}

func (c Config) EmailAllowed(email string) bool {
	_, ok := c.AllowedEmails[strings.ToLower(strings.TrimSpace(email))]
	return ok
}

// Load reads the configuration from the environment.
func Load() (Config, error) {
	return FromLookup(os.LookupEnv)
}

// FromLookup builds a Config from any key lookup function (tests pass a map).
func FromLookup(lookup func(string) (string, bool)) (Config, error) {
	get := func(key, def string) string {
		if v, ok := lookup(key); ok && v != "" {
			return v
		}
		return def
	}
	var errs []error

	c := Config{
		Env:                get("APP_ENV", "development"),
		Addr:               get("API_ADDR", "127.0.0.1:8080"),
		APIBasePath:        strings.TrimRight(get("API_BASE_PATH", ""), "/"),
		DatabaseURL:        get("DATABASE_URL", ""),
		GoogleClientID:     get("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: get("GOOGLE_CLIENT_SECRET", ""),
		AllowedEmails:      map[string]struct{}{},
	}
	if c.Env != "development" && c.Env != "production" {
		errs = append(errs, fmt.Errorf("APP_ENV must be development or production, got %q", c.Env))
	}

	u, err := url.Parse(get("PUBLIC_URL", "http://localhost:3000"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, fmt.Errorf("PUBLIC_URL must be an absolute URL"))
		u = &url.URL{Scheme: "http", Host: "localhost:3000"}
	}
	c.PublicURL = u

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}

	maxConns, err := strconv.Atoi(get("DB_MAX_CONNS", "4"))
	if err != nil || maxConns < 1 {
		errs = append(errs, errors.New("DB_MAX_CONNS must be a positive integer"))
		maxConns = 4
	}
	c.DBMaxConns = int32(maxConns)

	for _, e := range strings.Split(get("ALLOWED_EMAILS", ""), ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			c.AllowedEmails[e] = struct{}{}
		}
	}

	ttl, err := time.ParseDuration(get("SESSION_TTL", "720h"))
	if err != nil || ttl < time.Hour {
		errs = append(errs, errors.New("SESSION_TTL must be a duration of at least 1h"))
		ttl = 720 * time.Hour
	}
	c.SessionTTL = ttl

	c.DevLogin = get("AUTH_DEV_LOGIN", "0") == "1"

	if err := c.LogLevel.UnmarshalText([]byte(get("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}

	if c.Env == "production" {
		if c.PublicURL.Scheme != "https" {
			errs = append(errs, errors.New("PUBLIC_URL must use https in production"))
		}
		if c.GoogleClientID == "" || c.GoogleClientSecret == "" {
			errs = append(errs, errors.New("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required in production"))
		}
		if len(c.AllowedEmails) == 0 {
			errs = append(errs, errors.New("ALLOWED_EMAILS must list at least one email in production"))
		}
	}

	return c, errors.Join(errs...)
}
