// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

var testClient *tailscale.Client
var testServer *TestServer

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// TestProvider_SchemaSurface asserts that the v0.1 provider exposes
// exactly one resource type and zero data sources, matching FR-001/002/003
// and US1 acceptance scenario 2 from
// specs/002-standalone-membership-provider/spec.md.
func TestProvider_SchemaSurface(t *testing.T) {
	t.Parallel()

	p := Provider()

	if got, want := len(p.ResourcesMap), 1; got != want {
		t.Errorf("Provider().ResourcesMap has %d entries, want %d (membership-only fork)", got, want)
	}
	if got, want := len(p.DataSourcesMap), 0; got != want {
		t.Errorf("Provider().DataSourcesMap has %d entries, want %d (no data sources in v0.1)", got, want)
	}

	const wantKey = "tailscale_membership_tailnet_membership"
	if _, ok := p.ResourcesMap[wantKey]; !ok {
		gotKeys := make([]string, 0, len(p.ResourcesMap))
		for k := range p.ResourcesMap {
			gotKeys = append(gotKeys, k)
		}
		t.Errorf("Provider().ResourcesMap missing expected key %q; have %v", wantKey, gotKeys)
	}
}

// TestProvider_RejectsConflictingAuthModes pins FR-008 mechanically: any
// invalid combination of credentials MUST surface a diagnostic error whose
// Summary contains either "conflicting" or "mandatory" — i.e. the provider
// never silently picks an auth mode for the operator. Covers all five
// invalid combinations from US2 acceptance scenario 4 in
// specs/002-standalone-membership-provider/spec.md (T074).
func TestProvider_RejectsConflictingAuthModes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		apiKey        string
		oauthClientID string
		oauthSecret   string
		idToken       string
		wantSubstring string // MUST be one of {"conflicting", "mandatory"}
	}{
		{"api_key + oauth_client_id", "k", "id", "", "", "conflicting"},
		{"api_key + oauth_client_secret", "k", "", "s", "", "conflicting"},
		{"api_key + identity_token", "k", "", "", "tok", "conflicting"},
		{"oauth_client_id without secret or token", "", "id", "", "", "mandatory"},
		{"oauth_client_id + secret + identity_token (mutually exclusive)", "", "id", "s", "tok", "conflicting"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diags := validateProviderCreds(tc.apiKey, tc.oauthClientID, tc.oauthSecret, tc.idToken)

			if !diags.HasError() {
				t.Fatalf("expected diag.Diagnostics.HasError() to be true for case %q (FR-008: silent auth-mode selection forbidden); got %+v", tc.name, diags)
			}

			matched := false
			for _, d := range diags {
				if d.Severity != diag.Error {
					continue
				}
				if strings.Contains(d.Summary, tc.wantSubstring) {
					matched = true
					break
				}
			}
			if !matched {
				summaries := make([]string, 0, len(diags))
				for _, d := range diags {
					summaries = append(summaries, d.Summary)
				}
				t.Errorf("case %q: no error diagnostic Summary contains %q; got summaries: %v", tc.name, tc.wantSubstring, summaries)
			}
		})
	}
}

// TestProvider_UnknownUpstreamResourceTypeRejected asserts that a representative
// upstream resource type (one that existed in the upstream provider but is
// out-of-scope for the v0.1 membership fork) is no longer registered, so HCL
// referencing it surfaces a clean "unknown resource type" error from
// Terraform's schema check (US1 acceptance scenario 3, FR-002/003).
func TestProvider_UnknownUpstreamResourceTypeRejected(t *testing.T) {
	t.Parallel()

	p := Provider()

	// Sample a few representative removed upstream resource/data source types.
	// All MUST be absent under both their original names AND the new
	// tailscale_membership_-prefixed namespace (we only reserved the membership
	// resource under the new prefix).
	removed := []string{
		"tailscale_acl",
		"tailscale_dns_nameservers",
		"tailscale_tailnet_key",
		"tailscale_webhook",
		"tailscale_membership_dns_nameservers",
		"tailscale_membership_acl",
		"tailscale_tailnet_membership", // the pre-rename name; v0.1 only uses the prefixed form
	}

	for _, name := range removed {
		if _, ok := p.ResourcesMap[name]; ok {
			t.Errorf("ResourcesMap unexpectedly contains removed upstream resource %q", name)
		}
		if _, ok := p.DataSourcesMap[name]; ok {
			t.Errorf("DataSourcesMap unexpectedly contains removed upstream data source %q", name)
		}
	}
}

func testProviderFactories(t *testing.T) map[string]func() (*schema.Provider, error) {
	t.Helper()

	testClient, testServer = NewTestHarness(t)
	return map[string]func() (*schema.Provider, error){
		"tailscale": func() (*schema.Provider, error) {
			return Provider(func(p *schema.Provider) {
				// Set up a test harness for the provider
				p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
					return testClient, nil
				}

				// Don't require any of the global configuration
				p.Schema = nil
			}), nil
		},
	}
}

func TestValidateProviderCreds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		apiKey        string
		oauthClientID string
		oauthSecret   string
		idToken       string
		wantErr       string
	}{
		{
			name:    "valid api_key only",
			apiKey:  "test-api-key",
			wantErr: "",
		},
		{
			name:          "valid oauth with client secret",
			oauthClientID: "client-id",
			oauthSecret:   "client-secret",
			wantErr:       "",
		},
		{
			name:          "valid oauth with identity token",
			oauthClientID: "client-id",
			idToken:       "id-token",
			wantErr:       "",
		},
		{
			name:    "all credentials empty",
			wantErr: "credentials are empty",
		},
		{
			name:          "api_key conflicts with oauth_client_id",
			apiKey:        "test-api-key",
			oauthClientID: "client-id",
			wantErr:       "credentials are conflicting",
		},
		{
			name:        "api_key conflicts with oauth_client_secret",
			apiKey:      "test-api-key",
			oauthSecret: "client-secret",
			wantErr:     "credentials are conflicting",
		},
		{
			name:    "api_key conflicts with identity_token",
			apiKey:  "test-api-key",
			idToken: "id-token",
			wantErr: "credentials are conflicting",
		},
		{
			name:        "oauth_client_id missing with only oauth_client_secret",
			oauthSecret: "client-secret",
			wantErr:     "oauth_client_id' is empty",
		},
		{
			name:    "oauth_client_id missing with only identity_token",
			idToken: "id-token",
			wantErr: "oauth_client_id' is empty",
		},
		{
			name:          "oauth_client_id without secret or token",
			oauthClientID: "client-id",
			wantErr:       "oauth_client_secret' or 'identity_token' are mandatory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateProviderCreds(tt.apiKey, tt.oauthClientID, tt.oauthSecret, tt.idToken)

			if tt.wantErr == "" && diags.HasError() {
				t.Errorf("unexpected error: %v", diags)

			}

			if tt.wantErr != "" && !diags.HasError() {
				t.Errorf("expected error containing %q but got none", tt.wantErr)
				return
			}

			if tt.wantErr != "" {
				match := false
				for _, d := range diags {
					if d.Severity == diag.Error {
						errMsg := d.Summary + d.Detail
						if strings.Contains(errMsg, tt.wantErr) {
							match = true
							break
						}
					}
				}
				if !match {
					t.Errorf("expected error containing %q but got: %v", tt.wantErr, diags)
				}
			}
		})
	}
}
