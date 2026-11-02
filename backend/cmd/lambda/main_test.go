package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/franco-roura/ledger-01/backend/internal/apitest"
)

// lambdaDriver sends requests as API Gateway HTTP API (payload v2) events through the exact
// handler main() hands to lambda.Start, so no AWS account or emulator is needed.
func lambdaDriver(t testing.TB, h Handler) apitest.Driver {
	return func(t testing.TB, method, path string, body any, session string) apitest.Response {
		t.Helper()
		u, _ := url.Parse(path)
		ev := events.APIGatewayV2HTTPRequest{
			Version:        "2.0",
			RouteKey:       "$default",
			RawPath:        u.Path,
			RawQueryString: u.RawQuery,
			Headers:        map[string]string{"host": "api.example.com"},
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				Stage: "$default",
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
					Method: method, Path: u.Path, Protocol: "HTTP/1.1", SourceIP: "203.0.113.1", UserAgent: "parity-test",
				},
			},
		}
		if q := u.Query(); len(q) > 0 {
			ev.QueryStringParameters = map[string]string{}
			for k := range q {
				ev.QueryStringParameters[k] = q.Get(k)
			}
		}
		if body != nil {
			ev.Body = string(apitest.Encode(t, body))
			ev.Headers["content-type"] = "application/json"
		}
		if raw, ok := body.(apitest.Raw); ok {
			delete(ev.Headers, "content-type")
			for k := range raw.Header {
				ev.Headers[strings.ToLower(k)] = raw.Header.Get(k)
			}
		}
		if session != "" {
			// API Gateway v2 moves cookies out of the headers into their own array.
			ev.Cookies = []string{"ledger_session=" + session}
		}

		res, err := h(context.Background(), ev)
		if err != nil {
			t.Fatalf("lambda handler: %v", err)
		}
		header := http.Header{}
		for k, v := range res.Headers {
			header.Set(k, v)
		}
		for k, vs := range res.MultiValueHeaders {
			for _, v := range vs {
				header.Add(k, v)
			}
		}
		// API Gateway v2 returns Set-Cookie values in res.Cookies.
		for _, c := range res.Cookies {
			header.Add("Set-Cookie", c)
		}
		raw := []byte(res.Body)
		if res.IsBase64Encoded {
			raw, err = base64.StdEncoding.DecodeString(res.Body)
			if err != nil {
				t.Fatal(err)
			}
		}
		return apitest.Response{
			Status:  res.StatusCode,
			Header:  header,
			Cookies: (&http.Response{Header: header}).Cookies(),
			Body:    raw,
		}
	}
}

// TestLambdaParity runs the same end-to-end scenario through net/http and through the API
// Gateway v2 adapter and requires identical observations.
func TestLambdaParity(t *testing.T) {
	cfg := apitest.Config(t, nil)
	h := apitest.Handler(t, cfg)

	local := apitest.Scenario(t, apitest.HTTPDriver(t, h))
	lambda := apitest.Scenario(t, lambdaDriver(t, newHandler(h)))

	if !reflect.DeepEqual(local.Steps, lambda.Steps) {
		t.Fatalf("local and lambda differ:\nlocal:  %s\nlambda: %s",
			strings.Join(local.Steps, " | "), strings.Join(lambda.Steps, " | "))
	}
}

// TestLambdaMCPParity connects an MCP client through both transports.
func TestLambdaMCPParity(t *testing.T) {
	h := apitest.Handler(t, apitest.Config(t, nil))
	local := apitest.MCPScenario(t, apitest.HTTPDriver(t, h))
	lambda := apitest.MCPScenario(t, lambdaDriver(t, newHandler(h)))
	if !reflect.DeepEqual(local.Steps, lambda.Steps) {
		t.Fatalf("local and lambda differ:\nlocal:  %s\nlambda: %s",
			strings.Join(local.Steps, " | "), strings.Join(lambda.Steps, " | "))
	}
}

// TestLambdaBackupParity restores and exports a backup through both transports.
func TestLambdaBackupParity(t *testing.T) {
	h := apitest.Handler(t, apitest.Config(t, nil))
	local := apitest.BackupScenario(t, apitest.HTTPDriver(t, h))
	lambda := apitest.BackupScenario(t, lambdaDriver(t, newHandler(h)))
	if !reflect.DeepEqual(local.Steps, lambda.Steps) {
		t.Fatalf("local and lambda differ:\nlocal:  %s\nlambda: %s",
			strings.Join(local.Steps, " | "), strings.Join(lambda.Steps, " | "))
	}
}

func TestLambdaHouseholdParity(t *testing.T) {
	h := apitest.Handler(t, apitest.Config(t, nil))
	local := apitest.HouseholdScenario(t, apitest.HTTPDriver(t, h))
	lambda := apitest.HouseholdScenario(t, lambdaDriver(t, newHandler(h)))
	if !reflect.DeepEqual(local.Steps, lambda.Steps) {
		t.Fatalf("local and lambda differ:\nlocal:  %s\nlambda: %s",
			strings.Join(local.Steps, " | "), strings.Join(lambda.Steps, " | "))
	}
	if want := "member-invite 403"; !slices.Contains(local.Steps, want) {
		t.Fatalf("steps %v lack %q", local.Steps, want)
	}
	if want := "removed lost-access=true"; !slices.Contains(local.Steps, want) {
		t.Fatalf("steps %v lack %q", local.Steps, want)
	}
}

// TestLambdaBasePath covers the CloudFront /api/* path routed to API Gateway unchanged.
func TestLambdaBasePath(t *testing.T) {
	cfg := apitest.Config(t, map[string]string{"API_BASE_PATH": "/api"})
	do := lambdaDriver(t, newHandler(apitest.Handler(t, cfg)))
	if r := do(t, http.MethodPost, "/api/ledger.v1.UserService/GetMe", "{}", ""); r.Status != http.StatusUnauthorized {
		t.Fatalf("prefixed RPC via lambda = %d %s", r.Status, r.Body)
	}
}
