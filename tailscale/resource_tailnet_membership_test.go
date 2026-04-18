// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

const testTailnetMembershipCreate = `
resource "tailscale_membership_tailnet_membership" "alice" {
  login_name = "alice@example.com"
  role       = "member"
}
`

func TestResourceTailnetMembership_Create_EnsureMembershipCreatesInvite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					inv1 := []userInvite{{ID: "inv1", Email: "alice@example.com", Role: "member"}}
					testServer.ResponseByPath = map[string]interface{}{
						"GET /api/v2/tailnet/example.com/users":     map[string]interface{}{"users": []tailscale.User{}},
						"GET /api/v2/tailnet/example.com/user-invites": []userInvite{}, // fallback for destroy
						"POST /api/v2/tailnet/example.com/user-invites": inv1,
					}
					// First GET invites returns empty (so create runs); second GET invites (on Read) returns inv1
					testServer.ResponseQueueByPath = map[string][]interface{}{
						"GET /api/v2/tailnet/example.com/user-invites": {[]userInvite{}, inv1},
					}
				},
				Config: testTailnetMembershipCreate,
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["tailscale_membership_tailnet_membership.alice"]
					if !ok {
						return nil
					}
					if rs.Primary.ID == "" {
						return nil
					}
					if rs.Primary.Attributes["state"] != "pending" {
						return nil
					}
					if rs.Primary.Attributes["invite_id"] != "inv1" {
						return nil
					}
					return nil
				},
			},
		},
	})
}

func TestResourceTailnetMembership_Create_IdempotentWhenUserExists(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					existingUser := tailscale.User{
						ID:        "user1",
						LoginName: "alice@example.com",
						Role:      tailscale.UserRoleMember,
						Status:    tailscale.UserStatusActive,
					}
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":     map[string]interface{}{"users": []tailscale.User{existingUser}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{existingUser}}
				},
				Config: testTailnetMembershipCreate,
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["tailscale_membership_tailnet_membership.alice"]
					if !ok {
						return nil
					}
					if rs.Primary.Attributes["state"] != "active" {
						return nil
					}
					if rs.Primary.Attributes["user_id"] != "user1" {
						return nil
					}
					return nil
				},
			},
		},
	})
}

func TestResourceTailnetMembership_Read_StatePendingWhenInviteExists(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{{ID: "inv2", Email: "alice@example.com", Role: "member"}},
					}
					testServer.ResponseBody = []userInvite{{ID: "inv2", Email: "alice@example.com", Role: "member"}}
				},
				Config: testTailnetMembershipCreate,
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["tailscale_membership_tailnet_membership.alice"]
					if rs.Primary.Attributes["state"] != "pending" {
						return nil
					}
					return nil
				},
			},
		},
	})
}

func TestResourceTailnetMembership_Read_StateActiveWhenUserExists(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					u := tailscale.User{ID: "u1", LoginName: "alice@example.com", Role: tailscale.UserRoleMember, Status: tailscale.UserStatusActive, Created: time.Now(), LastSeen: time.Now()}
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{u}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{u}}
				},
				Config: testTailnetMembershipCreate,
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["tailscale_membership_tailnet_membership.alice"]
					if rs.Primary.Attributes["state"] != "active" {
						return nil
					}
					if rs.Primary.Attributes["user_id"] != "u1" {
						return nil
					}
					return nil
				},
			},
		},
	})
}

func TestResourceTailnetMembership_Delete_PendingCancelsInvite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{{ID: "inv3", Email: "alice@example.com", Role: "member"}},
					}
					testServer.ResponseBody = []userInvite{{ID: "inv3", Email: "alice@example.com", Role: "member"}}
				},
				Config: testTailnetMembershipCreate,
			},
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = nil
				},
				Destroy: true,
				Config:  testTailnetMembershipCreate,
				Check:   func(s *terraform.State) error { return nil },
			},
		},
	})
}

func TestResourceTailnetMembership_Delete_WhenUserExistsRemovesUser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					u := tailscale.User{ID: "u2", LoginName: "alice@example.com", Role: tailscale.UserRoleMember, Status: tailscale.UserStatusActive, Created: time.Now(), LastSeen: time.Now()}
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{u}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{u}}
				},
				Config: testTailnetMembershipCreate,
			},
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = nil
				},
				Destroy: true,
				Config:  testTailnetMembershipCreate,
				Check:   func(s *terraform.State) error { return nil },
			},
		},
	})
}

func TestResourceTailnetMembership_Delete_IdempotentWhenAlreadyRemoved(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{}}
				},
				Destroy: true,
				Config:  testTailnetMembershipCreate,
				Check:   func(s *terraform.State) error { return nil },
			},
		},
	})
}

func TestResourceTailnetMembership_Update_SuspendAndRestore(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					u := tailscale.User{ID: "u3", LoginName: "alice@example.com", Role: tailscale.UserRoleMember, Status: tailscale.UserStatusActive, Created: time.Now(), LastSeen: time.Now()}
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{u}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{u}}
				},
				Config: testTailnetMembershipCreate,
			},
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					u := tailscale.User{ID: "u3", LoginName: "alice@example.com", Role: tailscale.UserRoleMember, Status: tailscale.UserStatusSuspended, Created: time.Now(), LastSeen: time.Now()}
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{u}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{u}}
				},
				Config: `
resource "tailscale_membership_tailnet_membership" "alice" {
  login_name = "alice@example.com"
  role       = "member"
  suspended  = true
}
`,
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["tailscale_membership_tailnet_membership.alice"]
					if rs.Primary.Attributes["state"] != "disabled" {
						return nil
					}
					return nil
				},
			},
		},
	})
}

func TestResourceTailnetMembership_Delete_DowngradeOnDestroy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					u := tailscale.User{ID: "u_downgrade", LoginName: "alice@example.com", Role: tailscale.UserRoleAdmin, Status: tailscale.UserStatusActive, Created: time.Now(), LastSeen: time.Now()}
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{u}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{u}}
				},
				Config: `
resource "tailscale_membership_tailnet_membership" "alice" {
  login_name             = "alice@example.com"
  role                   = "admin"
  downgrade_on_destroy   = true
}
`,
			},
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = nil
				},
				Destroy: true,
				Config: `
resource "tailscale_membership_tailnet_membership" "alice" {
  login_name             = "alice@example.com"
  role                   = "admin"
  downgrade_on_destroy   = true
}
`,
				Check: func(s *terraform.State) error { return nil },
			},
		},
	})
}

func TestResourceTailnetMembership_Import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					u := tailscale.User{ID: "u4", LoginName: "alice@example.com", Role: tailscale.UserRoleMember, Status: tailscale.UserStatusActive, Created: time.Now(), LastSeen: time.Now()}
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":    map[string]interface{}{"users": []tailscale.User{u}},
						"/api/v2/tailnet/example.com/user-invites": []userInvite{},
					}
					testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{u}}
				},
				Config:        testTailnetMembershipCreate,
				ResourceName: "tailscale_membership_tailnet_membership.alice",
				ImportState:  true,
				ImportStateId: "example.com:alice@example.com", // tailnet:login_name
				ImportStateCheck: func(st []*terraform.InstanceState) error {
					if len(st) != 1 {
						return nil
					}
					if st[0].Attributes["login_name"] != "alice@example.com" {
						return nil
					}
					return nil
				},
			},
		},
	})
}

// T028 (FR-001a): invalid login_name (malformed email) is rejected by the schema
// validator at plan time. The error path is NOT idempotent — calling the validator
// repeatedly with the same invalid input must keep returning errors. The test also
// asserts (via the resource.Test step below) that no HTTP call is made when the
// validator rejects the input.
func TestResourceTailnetMembership_LoginName_RejectsInvalidEmail_FR001a(t *testing.T) {
	r := resourceTailnetMembership()
	sch, ok := r.Schema["login_name"]
	if !ok {
		t.Fatalf("login_name schema attribute missing")
	}
	if sch.ValidateDiagFunc == nil {
		t.Fatalf("login_name must use ValidateDiagFunc for FR-001a (got ValidateFunc=%v)", sch.ValidateFunc != nil)
	}

	invalid := []string{
		"not-an-email",
		"foo@",
		"@example.com",
		"",
		"   ",
		"alice@@example.com",
	}
	for _, in := range invalid {
		t.Run("invalid:"+in, func(t *testing.T) {
			diags := sch.ValidateDiagFunc(in, cty.GetAttrPath("login_name"))
			if !diags.HasError() {
				t.Fatalf("expected validation error for %q, got none", in)
			}
			// Non-idempotency: a repeat call with the same invalid input must still error.
			diags2 := sch.ValidateDiagFunc(in, cty.GetAttrPath("login_name"))
			if !diags2.HasError() {
				t.Fatalf("expected repeat validation to still error for %q (FR-001a is non-idempotent on the error path)", in)
			}
			// Diagnostic content: must mention login_name and either "email" or "valid"
			// so the operator understands the failure mode.
			var foundLoginName, foundHelpful bool
			for _, d := range diags {
				msg := d.Summary + " " + d.Detail
				if regexp.MustCompile(`(?i)login_name`).MatchString(msg) {
					foundLoginName = true
				}
				if regexp.MustCompile(`(?i)email|valid`).MatchString(msg) {
					foundHelpful = true
				}
			}
			if !foundLoginName || !foundHelpful {
				t.Fatalf("diag for %q must mention 'login_name' and one of {email,valid}; got %+v", in, diags)
			}
		})
	}

	// Valid inputs must pass.
	valid := []string{
		"alice@example.com",
		"bob.smith+tag@sub.example.co.uk",
	}
	for _, in := range valid {
		t.Run("valid:"+in, func(t *testing.T) {
			diags := sch.ValidateDiagFunc(in, cty.GetAttrPath("login_name"))
			if diags.HasError() {
				t.Fatalf("expected %q to validate, got error: %+v", in, diags)
			}
		})
	}

	// Display-name form is rejected (this is an identity field, not an RFC 5322
	// mailbox header).
	t.Run("rejects-display-name", func(t *testing.T) {
		diags := sch.ValidateDiagFunc("Alice <alice@example.com>", cty.GetAttrPath("login_name"))
		if !diags.HasError() {
			t.Fatalf("expected display-name form to be rejected, got none")
		}
		var matched bool
		for _, d := range diags {
			if regexp.MustCompile(`(?i)bare|display`).MatchString(d.Summary + " " + d.Detail) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("expected diag mentioning bare/display-name, got %+v", diags)
		}
	})

	// Non-string input is rejected (defensive: the schema type is TypeString so
	// this is normally unreachable from Terraform, but the validator must still
	// handle the contract).
	t.Run("rejects-non-string", func(t *testing.T) {
		diags := sch.ValidateDiagFunc(42, cty.GetAttrPath("login_name"))
		if !diags.HasError() {
			t.Fatalf("expected non-string input to be rejected")
		}
	})

	// No HTTP call is made when the validator rejects: drive a plan via the Terraform
	// SDK harness and confirm the test server saw zero requests.
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
resource "tailscale_membership_tailnet_membership" "bad" {
  login_name = "not-an-email"
  role       = "member"
}
`,
				ExpectError: regexp.MustCompile(`(?i)login_name`),
			},
		},
	})
	if testServer.Method != "" || testServer.Path != "" {
		t.Fatalf("expected zero HTTP calls when login_name validation fails, got %s %s", testServer.Method, testServer.Path)
	}
}

// T029 (FR-005a): the role schema must reject any value outside {member, admin}.
// This locks the contract that other Tailscale roles (owner, it-admin, etc.) are out
// of scope for this provider feature.
func TestResourceTailnetMembership_Role_RejectsValuesOutsideMemberAdmin_FR005a(t *testing.T) {
	r := resourceTailnetMembership()
	sch, ok := r.Schema["role"]
	if !ok {
		t.Fatalf("role schema attribute missing")
	}
	if sch.ValidateFunc == nil {
		t.Fatalf("role must use ValidateFunc to enforce FR-005a")
	}

	rejected := []string{
		"owner",
		"it-admin",
		"network-admin",
		"auditor",
		"billing-admin",
		"made-up-role",
		"Member", // case-sensitive: StringInSlice(_, false)
		"ADMIN",
	}
	for _, in := range rejected {
		t.Run("reject:"+in, func(t *testing.T) {
			_, errs := sch.ValidateFunc(in, "role")
			if len(errs) == 0 {
				t.Fatalf("expected role=%q to be rejected by FR-005a, got no error", in)
			}
		})
	}

	for _, in := range []string{"member", "admin"} {
		t.Run("accept:"+in, func(t *testing.T) {
			_, errs := sch.ValidateFunc(in, "role")
			if len(errs) != 0 {
				t.Fatalf("expected role=%q to be accepted, got errors: %v", in, errs)
			}
		})
	}
}

// T030 (FR-008): an invitation that is still listed by the backend MUST resolve to
// state="pending" regardless of any expiry timestamp. The userInvite struct in
// membership_api.go intentionally does not decode an "expires" field — this test
// proves that even when the backend sends one, the resolver still classifies the
// invite as pending. The test sends raw JSON containing "expires" in the past.
func TestResourceTailnetMembership_Read_ExpiredButListedInviteIsPending_FR008(t *testing.T) {
	expiredInviteJSON := []byte(`[{"id":"inv-expired","email":"alice@example.com","role":"member","expires":"2020-01-01T00:00:00Z"}]`)

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseByPath = map[string]interface{}{
						"/api/v2/tailnet/example.com/users":        map[string]interface{}{"users": []tailscale.User{}},
						"/api/v2/tailnet/example.com/user-invites": expiredInviteJSON,
					}
					testServer.ResponseBody = expiredInviteJSON
				},
				Config: testTailnetMembershipCreate,
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["tailscale_membership_tailnet_membership.alice"]
					if !ok {
						t.Fatalf("resource not in state")
					}
					if got := rs.Primary.Attributes["state"]; got != "pending" {
						t.Fatalf("FR-008: expected state=pending for expired-but-listed invite, got %q", got)
					}
					if got := rs.Primary.Attributes["invite_id"]; got != "inv-expired" {
						t.Fatalf("expected invite_id=inv-expired, got %q", got)
					}
					return nil
				},
			},
		},
	})

	// Direct unit-level assertion against membershipResolve: ignore the expires
	// timestamp entirely — the backend listing is the source of truth.
	resolved, diags := membershipResolve(context.Background(), testClient, "alice@example.com")
	if diags.HasError() {
		t.Fatalf("membershipResolve diags: %v", diags)
	}
	if resolved == nil || resolved.State != membershipStatePending || resolved.InviteID != "inv-expired" {
		t.Fatalf("FR-008 resolve: want state=pending invite_id=inv-expired, got %+v", resolved)
	}
}

// T035 (FR-009): when the Tailscale API rejects suspendUser/deleteUser (e.g., for
// the last admin or account owner), the provider's diagnostic MUST contain the
// substring "last admin or account owner" so operators can identify the cause.
// Enforcement is API-side; this test locks the diagnostic-message contract for
// both the suspend (Update) and delete (Delete) paths.
func TestResourceTailnetMembership_LastAdminRefusal_DiagMentionsCause_FR009(t *testing.T) {
	t.Run("suspend", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest:        true,
			ProviderFactories: testProviderFactories(t),
			Steps: []resource.TestStep{
				{
					PreConfig: func() {
						testServer.ResponseCode = http.StatusOK
						u := tailscale.User{ID: "u-last-admin", LoginName: "alice@example.com", Role: tailscale.UserRoleAdmin, Status: tailscale.UserStatusActive, Created: time.Now(), LastSeen: time.Now()}
						testServer.ResponseByPath = map[string]interface{}{
							"/api/v2/tailnet/example.com/users":        map[string]interface{}{"users": []tailscale.User{u}},
							"/api/v2/tailnet/example.com/user-invites": []userInvite{},
						}
						testServer.ResponseBody = map[string]interface{}{"users": []tailscale.User{u}}
					},
					Config: `
resource "tailscale_membership_tailnet_membership" "alice" {
  login_name = "alice@example.com"
  role       = "admin"
}
`,
				},
				{
					PreConfig: func() {
						u := tailscale.User{ID: "u-last-admin", LoginName: "alice@example.com", Role: tailscale.UserRoleAdmin, Status: tailscale.UserStatusActive, Created: time.Now(), LastSeen: time.Now()}
						testServer.ResponseCode = http.StatusOK
						testServer.ResponseByPath = map[string]interface{}{
							"/api/v2/tailnet/example.com/users":        map[string]interface{}{"users": []tailscale.User{u}},
							"/api/v2/tailnet/example.com/user-invites": []userInvite{},
							// suspend POST returns a body explaining the refusal.
							"POST /api/v2/users/u-last-admin/suspend": map[string]string{"message": "cannot suspend the last admin or account owner"},
						}
						// Only the suspend POST returns 403; refresh GETs stay 200.
						testServer.ResponseCodeByPath = map[string]int{
							"POST /api/v2/users/u-last-admin/suspend": http.StatusForbidden,
						}
					},
					Config: `
resource "tailscale_membership_tailnet_membership" "alice" {
  login_name = "alice@example.com"
  role       = "admin"
  suspended  = true
}
`,
					ExpectError: regexp.MustCompile(`(?i)last admin or account owner`),
				},
			},
		})
	})

	// Direct unit test for the deleteContext error-wrap path (line ~304). Driving
	// the same scenario through the SDK's TestCase + Destroy: true is impractical
	// because the framework's post-test cleanup would re-run the destroy and hit
	// the rigged 403, failing the test for the wrong reason.
	t.Run("delete", func(t *testing.T) {
		_ = testProviderFactories(t) // initialize testClient + testServer
		testServer.ResponseCode = http.StatusOK
		testServer.ResponseCodeByPath = map[string]int{
			"POST /api/v2/users/u-last-admin/delete": http.StatusForbidden,
		}
		testServer.ResponseByPath = map[string]interface{}{
			"POST /api/v2/users/u-last-admin/delete": map[string]string{"message": "cannot delete the last admin or account owner"},
		}

		api := membershipAPI(testClient)
		err := api.deleteUser(context.Background(), "u-last-admin")
		if err == nil {
			t.Fatalf("expected deleteUser to return an error on 403")
		}
		diags := diagnosticsError(err, "Failed to delete user; ensure you are not the last admin or account owner")
		if !diags.HasError() {
			t.Fatalf("expected diagnostic error")
		}
		var matched bool
		for _, d := range diags {
			msg := d.Summary + " " + d.Detail
			if regexp.MustCompile(`(?i)last admin or account owner`).MatchString(msg) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("FR-009: delete-path diagnostic must mention 'last admin or account owner'; got %+v", diags)
		}
	})
}
