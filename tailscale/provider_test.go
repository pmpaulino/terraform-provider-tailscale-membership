// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

var testClient *tailscale.Client
var testServer *TestServer
var testAccProvider = Provider()

// testAccPreCheck ensures that the TAILSCALE_API_KEY and TAILSCALE_BASE_URL variables
// are set and configures the provider. This must be called before running acceptance
// tests.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if v := os.Getenv("TAILSCALE_API_KEY"); v == "" {
		t.Fatal("TAILSCALE_API_KEY must be set for acceptance tests")
	}

	if v := os.Getenv("TAILSCALE_BASE_URL"); v == "" {
		t.Fatal("TAILSCALE_BASE_URL must be set for acceptance tests")
	}

	if diags := testAccProvider.Configure(context.Background(), &terraform.ResourceConfig{}); diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				t.Fatalf("Failed to configure provider: %s", d.Summary)
			}
		}
	}
}

func testAccProviderFactories(t *testing.T) map[string]func() (*schema.Provider, error) {
	t.Helper()

	return map[string]func() (*schema.Provider, error){
		"tailscale": func() (*schema.Provider, error) {
			return Provider(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_Implemented(t *testing.T) {
	var _ *schema.Provider = Provider()
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

func testResourceCreated(name, hcl string) resource.TestStep {
	return resource.TestStep{
		ResourceName:       name,
		Config:             hcl,
		ExpectNonEmptyPlan: true,
		Check: func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources[name]

			if !ok {
				return fmt.Errorf("not found: %s", name)
			}

			if rs.Primary.ID == "" {
				return errors.New("no ID set")
			}

			return nil
		},
	}
}

func testResourceDestroyed(name string, hcl string) resource.TestStep {
	return resource.TestStep{
		ResourceName: name,
		Destroy:      true,
		Config:       hcl,
		Check: func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources[name]

			if !ok {
				return fmt.Errorf("not found: %s", name)
			}

			if rs.Primary.ID == "" {
				return errors.New("no ID set")
			}

			return nil
		},
	}
}

func checkResourceRemoteProperties(resourceName string, check func(client *tailscale.Client, rs *terraform.ResourceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*tailscale.Client)
		return check(client, rs)
	}
}

func checkResourceDestroyed(resourceName string, check func(client *tailscale.Client, rs *terraform.ResourceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*tailscale.Client)
		return check(client, rs)
	}
}

// checkPropertiesMatch compares the properties on a named resource to the
// expected values in a map. All values in the [terraform.ResourceState] will
// be strings, while the map may contain strings, booleans or ints.
// This function returns an error if the resource is not found, or if any of
// the properties don't match.
func checkPropertiesMatch(resourceName string, s *terraform.State, expected map[string]any) error {
	rs := s.RootModule().Resources[resourceName]
	if rs == nil {
		return fmt.Errorf("no resource found for user %s", resourceName)
	}

	actual := rs.Primary.Attributes
	for k, v := range expected {
		switch t := v.(type) {
		case int:
			if actual[k] != fmt.Sprint(t) {
				return fmt.Errorf("wrong value for property %s of user %s, want %d, got %s", k, resourceName, t, actual[k])
			}
		case bool:
			if actual[k] != fmt.Sprint(t) {
				return fmt.Errorf("wrong value for property %s of user %s, want %v, got %s", k, resourceName, t, actual[k])
			}
		case string:
			if actual[k] != t {
				return fmt.Errorf("wrong value for property %s of user %s, want %s, got %s", k, resourceName, t, actual[k])
			}
		}
	}

	return nil
}

// assertEqual compares the expected and actual using [cmp.Diff] and reports an
// error if they're not equal.
func assertEqual(want, got any, errorMessage string) error {
	if diff := cmp.Diff(want, got); diff != "" {
		return fmt.Errorf("%s (-want +got): %s", errorMessage, diff)
	}
	return nil
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
