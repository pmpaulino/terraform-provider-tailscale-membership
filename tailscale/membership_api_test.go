// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"tailscale.com/client/tailscale/v2"
)

// markerHeader is added by stubAuth's RoundTripper so tests can confirm
// the request actually flowed through the auth-decorated *http.Client.
const markerHeader = "X-Membership-Test-Marker"

// stubAuth implements tailscale.Auth. Its HTTPClient returns an
// *http.Client whose Transport injects markerHeader on every outbound
// request. This lets TestMembershipAPI_RoutesThroughAuthHTTPClient
// assert that membership_api.go::do() routes requests through
// (*tailscale.Client).Auth.HTTPClient(...) rather than around it.
type stubAuth struct {
	tripped       *atomic.Int32      // incremented on every outbound request
	gotAuthHeader *atomic.Value      // last observed Authorization header (string)
	gotBaseURL    *atomic.Value      // baseURL passed to HTTPClient (string)
	respond       func(*http.Request) (*http.Response, error)
}

func (s *stubAuth) HTTPClient(orig *http.Client, baseURL string) *http.Client {
	s.gotBaseURL.Store(baseURL)
	return &http.Client{
		Transport: &markerRoundTripper{
			tripped:       s.tripped,
			gotAuthHeader: s.gotAuthHeader,
			respond:       s.respond,
		},
	}
}

type markerRoundTripper struct {
	tripped       *atomic.Int32
	gotAuthHeader *atomic.Value
	respond       func(*http.Request) (*http.Response, error)
}

func (m *markerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.tripped.Add(1)
	m.gotAuthHeader.Store(req.Header.Get("Authorization"))
	req.Header.Set(markerHeader, "yes")
	if m.respond != nil {
		return m.respond(req)
	}
	body := "[]"
	// createUserInvite expects at least one element back.
	if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/user-invites") {
		body = `[{"id":"inv-1","email":"u@example.com","role":"member"}]`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{markerHeader: []string{"yes"}},
		Request:    req,
	}, nil
}

// newStubAuthClient builds a tailscale.Client whose Auth is a stubAuth
// that records every outbound request. The returned helper closures
// expose the recorded counters for assertions.
func newStubAuthClient(t *testing.T) (*tailscale.Client, *stubAuth) {
	t.Helper()

	tripped := &atomic.Int32{}
	gotAuthHeader := &atomic.Value{}
	gotBaseURL := &atomic.Value{}
	gotAuthHeader.Store("") // initialise so .Load().(string) is always safe
	gotBaseURL.Store("")

	stub := &stubAuth{
		tripped:       tripped,
		gotAuthHeader: gotAuthHeader,
		gotBaseURL:    gotBaseURL,
	}

	parsed, err := url.Parse("https://api.tailscale.example")
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}
	c := &tailscale.Client{
		BaseURL: parsed,
		Tailnet: "test-tailnet",
		Auth:    stub,
	}
	return c, stub
}

// TestMembershipAPI_RoutesThroughAuthHTTPClient enforces FR-006/FR-007 and
// the contract in specs/002-standalone-membership-provider/contracts/auth-transport.md:
// every helper method MUST send its request through the *http.Client returned
// by tailscale.Client.Auth.HTTPClient(...), and MUST NOT add an
// Authorization: Basic header itself when Auth is configured.
func TestMembershipAPI_RoutesThroughAuthHTTPClient(t *testing.T) {
	t.Parallel()

	type call struct {
		name string
		fn   func(*membershipAPIClient) error
	}
	calls := []call{
		{"listUserInvites", func(m *membershipAPIClient) error {
			_, err := m.listUserInvites(context.Background())
			return err
		}},
		{"createUserInvite", func(m *membershipAPIClient) error {
			_, err := m.createUserInvite(context.Background(), "u@example.com", "member")
			return err
		}},
		{"deleteUserInvite", func(m *membershipAPIClient) error {
			return m.deleteUserInvite(context.Background(), "invite-123")
		}},
		{"suspendUser", func(m *membershipAPIClient) error {
			return m.suspendUser(context.Background(), "user-123")
		}},
		{"restoreUser", func(m *membershipAPIClient) error {
			return m.restoreUser(context.Background(), "user-123")
		}},
		{"deleteUser", func(m *membershipAPIClient) error {
			return m.deleteUser(context.Background(), "user-123")
		}},
		{"updateUserRole", func(m *membershipAPIClient) error {
			return m.updateUserRole(context.Background(), "user-123", "admin")
		}},
	}

	for _, c := range calls {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			client, stub := newStubAuthClient(t)
			m := membershipAPI(client)

			before := stub.tripped.Load()
			if err := c.fn(m); err != nil {
				t.Fatalf("%s: unexpected error: %v", c.name, err)
			}
			after := stub.tripped.Load()

			if got := after - before; got != 1 {
				t.Fatalf("%s: expected exactly 1 request through Auth.HTTPClient, got %d", c.name, got)
			}

			if got := stub.gotAuthHeader.Load().(string); got != "" {
				t.Errorf("%s: expected NO Authorization header from helper (Auth-decorated client owns auth), got %q", c.name, got)
			}

			if got := stub.gotBaseURL.Load().(string); got != "https://api.tailscale.example" {
				t.Errorf("%s: Auth.HTTPClient called with baseURL %q, want %q", c.name, got, "https://api.tailscale.example")
			}
		})
	}
}

// TestMembershipAPI_APIKeyStillUsesBasicAuth is the regression-safety
// counterpart: in API-key mode (Auth == nil, APIKey != ""), the helper
// MUST send Authorization: Basic <base64("test-key:")> on every request,
// and the configured BaseURL MUST be honored. This pins FR-006/FR-007 to
// "no auth-mode silently switches to a different transport".
func TestMembershipAPI_APIKeyStillUsesBasicAuth(t *testing.T) {
	t.Parallel()

	type call struct {
		name           string
		expectedMethod string
		expectedPath   string
		fn             func(*membershipAPIClient) error
	}
	calls := []call{
		{"listUserInvites", http.MethodGet, "/api/v2/tailnet/test-tailnet/user-invites", func(m *membershipAPIClient) error {
			_, err := m.listUserInvites(context.Background())
			return err
		}},
		{"createUserInvite", http.MethodPost, "/api/v2/tailnet/test-tailnet/user-invites", func(m *membershipAPIClient) error {
			_, err := m.createUserInvite(context.Background(), "u@example.com", "member")
			return err
		}},
		{"deleteUserInvite", http.MethodDelete, "/api/v2/user-invites/invite-123", func(m *membershipAPIClient) error {
			return m.deleteUserInvite(context.Background(), "invite-123")
		}},
		{"suspendUser", http.MethodPost, "/api/v2/users/user-123/suspend", func(m *membershipAPIClient) error {
			return m.suspendUser(context.Background(), "user-123")
		}},
		{"restoreUser", http.MethodPost, "/api/v2/users/user-123/restore", func(m *membershipAPIClient) error {
			return m.restoreUser(context.Background(), "user-123")
		}},
		{"deleteUser", http.MethodPost, "/api/v2/users/user-123/delete", func(m *membershipAPIClient) error {
			return m.deleteUser(context.Background(), "user-123")
		}},
		{"updateUserRole", http.MethodPost, "/api/v2/users/user-123/role", func(m *membershipAPIClient) error {
			return m.updateUserRole(context.Background(), "user-123", "admin")
		}},
	}

	const apiKey = "test-key"
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(apiKey+":"))

	for _, c := range calls {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var (
				gotAuth   string
				gotMethod string
				gotPath   string
				called    int32
			)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&called, 1)
				gotAuth = r.Header.Get("Authorization")
				gotMethod = r.Method
				gotPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/user-invites") {
					_, _ = w.Write([]byte(`[{"id":"inv-1","email":"u@example.com","role":"member"}]`))
					return
				}
				_, _ = w.Write([]byte(`[]`))
			}))
			t.Cleanup(srv.Close)

			parsed, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatalf("parse server url: %v", err)
			}
			client := &tailscale.Client{
				BaseURL: parsed,
				Tailnet: "test-tailnet",
				APIKey:  apiKey,
			}

			m := membershipAPI(client)
			if err := c.fn(m); err != nil {
				t.Fatalf("%s: unexpected error: %v", c.name, err)
			}

			if got := atomic.LoadInt32(&called); got != 1 {
				t.Fatalf("%s: expected exactly 1 request to test server, got %d", c.name, got)
			}
			if gotAuth != wantAuth {
				t.Errorf("%s: Authorization header = %q, want %q", c.name, gotAuth, wantAuth)
			}
			if gotMethod != c.expectedMethod {
				t.Errorf("%s: HTTP method = %q, want %q", c.name, gotMethod, c.expectedMethod)
			}
			if gotPath != c.expectedPath {
				t.Errorf("%s: request path = %q, want %q", c.name, gotPath, c.expectedPath)
			}
		})
	}
}
