// Package apitest drives the API over HTTP for tests: a small JSON client (Connect speaks plain
// JSON over POST) that can target either a real HTTP server or any other transport.
package apitest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/franco-roura/ledger-01/backend/internal/config"
	"github.com/franco-roura/ledger-01/backend/internal/server"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"log/slog"
)

// Response is what a driver returns for one request.
type Response struct {
	Status  int
	Header  http.Header
	Cookies []*http.Cookie
	Body    []byte
}

// JSON decodes the body into a map for assertions.
func (r Response) JSON(t testing.TB) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
	return m
}

// Driver sends one request (with an optional session cookie) and returns the response.
type Driver func(t testing.TB, method, path string, body any, session string) Response

// Config is a development config pointing at the shared test database, with the dev login on.
func Config(t testing.TB, extra map[string]string) config.Config {
	t.Helper()
	env := map[string]string{
		"APP_ENV":          "development",
		"DATABASE_URL":     testdb.DSN(t),
		"PUBLIC_URL":       "http://localhost:3000",
		"ALLOWED_EMAILS":   "owner@example.com, partner@example.com",
		"AUTH_DEV_LOGIN":   "1",
		"GOOGLE_CLIENT_ID": "test-client.apps.googleusercontent.com",
	}
	for k, v := range extra {
		env[k] = v
	}
	cfg, err := config.FromLookup(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// Handler builds the real API handler for a config.
func Handler(t testing.TB, cfg config.Config) http.Handler {
	t.Helper()
	return server.New(server.Deps{Config: cfg, Pool: testdb.Pool(t), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

// Raw is a request body sent as-is with its own headers (form posts, bearer tokens), where the
// default is JSON with the session cookie.
type Raw struct {
	Body   string
	Header http.Header
}

// Encode marshals a request body; strings are sent as-is.
func Encode(t testing.TB, body any) []byte {
	t.Helper()
	switch b := body.(type) {
	case nil:
		return nil
	case string:
		return []byte(b)
	case Raw:
		return []byte(b.Body)
	default:
		out, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
}

// HTTPDriver sends requests to an in-process HTTP server running the handler.
func HTTPDriver(t testing.TB, h http.Handler) Driver {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return func(t testing.TB, method, path string, body any, session string) Response {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+path, bytes.NewReader(Encode(t, body)))
		if err != nil {
			t.Fatal(err)
		}
		if raw, ok := body.(Raw); ok {
			for k, vs := range raw.Header {
				req.Header[k] = vs
			}
		} else if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if session != "" {
			req.AddCookie(&http.Cookie{Name: "ledger_session", Value: session})
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return Response{Status: res.StatusCode, Header: res.Header, Cookies: res.Cookies(), Body: b}
	}
}

// SessionCookie returns the session token set by a response, if any.
func SessionCookie(r Response) string {
	for _, c := range r.Cookies {
		if c.Name == "ledger_session" && c.MaxAge >= 0 {
			return c.Value
		}
	}
	return ""
}
