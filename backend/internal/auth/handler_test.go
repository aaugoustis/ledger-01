package auth

import "testing"

func TestSafeNext(t *testing.T) {
	for next, want := range map[string]string{
		"":                     "/dashboard",
		"/wallets/abc?from=x":  "/wallets/abc?from=x",
		"/oauth/authorize?a=b": "/oauth/authorize?a=b",
		"https://evil.example": "/dashboard",
		"//evil.example":       "/dashboard",
		`/\evil.example`:       "/dashboard",
		`/a\b`:                 "/dashboard",
		"/\t/evil.example":     "/dashboard",
		"/\n/evil.example":     "/dashboard",
		"/\r\n/evil.example":   "/dashboard",
		"/\x00x":               "/dashboard",
		"relative":             "/dashboard",
		// Still percent-encoded when it reaches the browser, so it stays a path.
		"/%09/evil.example": "/%09/evil.example",
	} {
		if got := safeNext(next); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", next, got, want)
		}
	}
}
