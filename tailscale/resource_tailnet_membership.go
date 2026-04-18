// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"tailscale.com/client/tailscale/v2"
)

const (
	membershipStatePending  = "pending"
	membershipStateActive   = "active"
	membershipStateDisabled = "disabled"
)

// membershipResolveResult holds the result of resolving a membership by login_name.
type membershipResolveResult struct {
	State    string // pending | active | disabled
	Role     string
	InviteID string
	UserID   string
	User     *tailscale.User // when active or disabled
}

// resourceTailnetMembership returns the tailscale_tailnet_membership resource schema and CRUD.
func resourceTailnetMembership() *schema.Resource {
	return &schema.Resource{
		Description:   "The tailnet_membership resource manages a user's membership in a tailnet. Creating the resource ensures the identity is in the tailnet (creates an invite if needed); destroying it cancels a pending invite or removes the user. Supports suspend/restore and optional downgrade on destroy.",
		CreateContext: resourceTailnetMembershipCreate,
		ReadContext:   resourceTailnetMembershipRead,
		UpdateContext: resourceTailnetMembershipUpdate,
		DeleteContext: resourceTailnetMembershipDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceTailnetMembershipImport,
		},
		Schema: map[string]*schema.Schema{
			"login_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "The identity (email) for the membership. Used to match an invite or user in the tailnet. MUST be a well-formed email address (e.g., alice@example.com).",
				ValidateDiagFunc: validateLoginNameEmail,
			},
			"role": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "member",
				Description:  "The role to assign. Use `member` or `admin`.",
				ValidateFunc: validation.StringInSlice([]string{"member", "admin"}, false),
			},
			"downgrade_on_destroy": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, on destroy the user is downgraded to member or suspended instead of removed.",
			},
			"suspended": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When true, the membership is disabled (user suspended). When false, the user is active.",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current state: `pending` (invite not yet accepted), `active`, or `disabled` (suspended).",
			},
			"invite_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Tailscale user invite ID when state is pending.",
			},
			"user_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Tailscale user ID when state is active or disabled.",
			},
		},
	}
}

// validateLoginNameEmail enforces FR-001a: login_name MUST be a well-formed email
// address. The error path is non-idempotent — repeated invalid input keeps erroring
// and no HTTP call is made because Terraform fails the plan before reaching CRUD.
// Display-name forms ("Alice <alice@example.com>") are rejected: this is an identity
// field, not a mailbox header.
func validateLoginNameEmail(v interface{}, p cty.Path) diag.Diagnostics {
	s, ok := v.(string)
	if !ok {
		return diag.Diagnostics{{Severity: diag.Error, Summary: "login_name must be a string", AttributePath: p}}
	}
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "login_name must be a valid email address",
			Detail:        "login_name is empty; provide a well-formed email address (e.g., alice@example.com)",
			AttributePath: p,
		}}
	}
	if !strings.Contains(trimmed, "@") {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "login_name must be a valid email address",
			Detail:        fmt.Sprintf("login_name=%q is missing '@'; provide a well-formed email address (e.g., alice@example.com)", s),
			AttributePath: p,
		}}
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "login_name is not a valid email address",
			Detail:        fmt.Sprintf("login_name=%q: %s", s, err.Error()),
			AttributePath: p,
		}}
	}
	if addr.Address != trimmed {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "login_name must be a bare email address",
			Detail:        fmt.Sprintf("login_name=%q must contain only the email address (no display name)", s),
			AttributePath: p,
		}}
	}
	return nil
}

// membershipResolve lists user invites and users for the tailnet and finds the membership by login_name (email).
// It triggers client init by calling Users().List first. Returns nil if not found.
func membershipResolve(ctx context.Context, client *tailscale.Client, loginName string) (*membershipResolveResult, diag.Diagnostics) {
	normalized := strings.TrimSpace(strings.ToLower(loginName))
	if normalized == "" {
		return nil, diag.Errorf("login_name is empty")
	}

	// Trigger client init (required for OAuth so HTTP has auth transport).
	users, err := client.Users().List(ctx, nil, nil)
	if err != nil {
		return nil, diagnosticsError(err, "Failed to list users for membership resolve")
	}

	api := membershipAPI(client)
	invites, err := api.listUserInvites(ctx)
	if err != nil {
		return nil, diagnosticsError(err, "Failed to list user invites for membership resolve")
	}

	// Match invite by email (case-insensitive).
	//
	// FR-008 invariant: if the backend lists an invitation, this resolver MUST
	// classify it as `pending` regardless of any expiry timestamp. The userInvite
	// struct in membership_api.go intentionally omits an `Expires` field so this
	// invariant is enforced structurally — there is nothing in the local view to
	// branch on. Removing an expired-but-listed invite is the operator's job
	// (e.g., calling deleteUserInvite); until that happens, the membership state
	// is reported as pending.
	for i := range invites {
		if strings.TrimSpace(strings.ToLower(invites[i].Email)) == normalized {
			return &membershipResolveResult{
				State:    membershipStatePending,
				Role:     invites[i].Role,
				InviteID: invites[i].ID,
			}, nil
		}
	}

	// Match user by login name.
	for i := range users {
		if strings.TrimSpace(strings.ToLower(users[i].LoginName)) == normalized {
			state := membershipStateActive
			if users[i].Status == tailscale.UserStatusSuspended {
				state = membershipStateDisabled
			}
			return &membershipResolveResult{
				State:  state,
				Role:   string(users[i].Role),
				UserID: users[i].ID,
				User:   &users[i],
			}, nil
		}
	}

	return nil, nil
}

func resourceTailnetMembershipID(tailnet, loginName string) string {
	return tailnet + ":" + loginName
}

func parseMembershipID(id string) (tailnet, loginName string, err error) {
	idx := strings.Index(id, ":")
	if idx <= 0 || idx >= len(id)-1 {
		return "", "", fmt.Errorf("invalid membership id %q (expected tailnet:login_name)", id)
	}
	return id[:idx], id[idx+1:], nil
}

func resourceTailnetMembershipCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	tailnet := client.Tailnet
	if tailnet == "" {
		tailnet = "-"
	}
	loginName := d.Get("login_name").(string)
	desiredRole := d.Get("role").(string)

	resolved, diags := membershipResolve(ctx, client, loginName)
	if diags != nil && diags.HasError() {
		return diags
	}

	if resolved != nil {
		// Idempotent: already member or pending invite.
		id := resourceTailnetMembershipID(tailnet, loginName)
		d.SetId(id)
		return resourceTailnetMembershipRead(ctx, d, m)
	}

	// No user and no invite: create invite.
	api := membershipAPI(client)
	invite, err := api.createUserInvite(ctx, loginName, desiredRole)
	if err != nil {
		return diagnosticsError(err, "Failed to create user invite; ensure your token has UserInvites scope and the identity is valid")
	}

	id := resourceTailnetMembershipID(tailnet, loginName)
	d.SetId(id)
	_ = d.Set("state", membershipStatePending)
	_ = d.Set("invite_id", invite.ID)
	_ = d.Set("role", invite.Role)
	return nil
}

func resourceTailnetMembershipRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	_, loginName, err := parseMembershipID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resolved, diags := membershipResolve(ctx, client, loginName)
	if diags != nil && diags.HasError() {
		return diags
	}

	if resolved == nil {
		// Not found: remove from state for idempotent delete.
		d.SetId("")
		return nil
	}

	_ = d.Set("login_name", loginName)
	_ = d.Set("state", resolved.State)
	_ = d.Set("role", resolved.Role)
	if resolved.InviteID != "" {
		_ = d.Set("invite_id", resolved.InviteID)
		_ = d.Set("user_id", "")
	} else {
		_ = d.Set("invite_id", "")
		_ = d.Set("user_id", resolved.UserID)
	}
	if resolved.State == membershipStateDisabled {
		_ = d.Set("suspended", true)
	} else {
		_ = d.Set("suspended", false)
	}
	return nil
}

func resourceTailnetMembershipUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	_, loginName, err := parseMembershipID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resolved, diags := membershipResolve(ctx, client, loginName)
	if diags != nil && diags.HasError() {
		return diags
	}
	if resolved == nil {
		return diag.Errorf("membership not found for login_name %q", loginName)
	}

	desiredRole := d.Get("role").(string)
	desiredSuspended := d.Get("suspended").(bool)
	api := membershipAPI(client)

	if resolved.State == membershipStatePending {
		// No update for pending (role is fixed on invite); only config drift for role/suspended.
		return resourceTailnetMembershipRead(ctx, d, m)
	}

	// Active or disabled user: apply role and suspend/restore.
	if resolved.UserID != "" {
		if resolved.Role != desiredRole {
			if err := api.updateUserRole(ctx, resolved.UserID, desiredRole); err != nil {
				return diagnosticsError(err, "Failed to update user role")
			}
		}
		wantDisabled := desiredSuspended
		isDisabled := resolved.State == membershipStateDisabled
		if wantDisabled && !isDisabled {
			if err := api.suspendUser(ctx, resolved.UserID); err != nil {
				return diagnosticsError(err, "Failed to suspend user; ensure you are not the last admin or account owner")
			}
		} else if !wantDisabled && isDisabled {
			if err := api.restoreUser(ctx, resolved.UserID); err != nil {
				return diagnosticsError(err, "Failed to restore user")
			}
		}
	}

	return resourceTailnetMembershipRead(ctx, d, m)
}

func resourceTailnetMembershipDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	_, loginName, err := parseMembershipID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resolved, diags := membershipResolve(ctx, client, loginName)
	if diags != nil && diags.HasError() {
		return diags
	}

	if resolved == nil {
		// Already absent: idempotent success.
		return nil
	}

	api := membershipAPI(client)
	downgrade := d.Get("downgrade_on_destroy").(bool)

	if resolved.State == membershipStatePending {
		if err := api.deleteUserInvite(ctx, resolved.InviteID); err != nil {
			return diagnosticsError(err, "Failed to delete user invite")
		}
		return nil
	}

	if resolved.UserID != "" {
		if downgrade {
			_ = api.updateUserRole(ctx, resolved.UserID, "member")
			_ = api.suspendUser(ctx, resolved.UserID)
		} else {
			if err := api.deleteUser(ctx, resolved.UserID); err != nil {
				return diagnosticsError(err, "Failed to delete user; ensure you are not the last admin or account owner")
			}
		}
	}
	return nil
}

func resourceTailnetMembershipImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	_, _, err := parseMembershipID(d.Id())
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
