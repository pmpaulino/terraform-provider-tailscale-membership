---
page_title: "tailscale_membership_tailnet_membership Resource - Tailscale Membership"
subcategory: ""
description: |-
  Manages a user's membership in a Tailscale tailnet.
---

# tailscale_membership_tailnet_membership (Resource)

The `tailscale_membership_tailnet_membership` resource manages a user's membership in a tailnet. Creating the resource ensures the identity is in the tailnet (creates an invite if needed); destroying it cancels a pending invite or removes the user. Supports suspend/restore and optional downgrade on destroy.

**Authentication requirement:** The Tailscale `user-invites` API rejects OAuth client tokens and federated identity tokens with `403 "operation only permitted for user-owned keys"`, regardless of scopes. A **personal API key** (`tskey-api-...`) is required to use the `create` path of this resource. See the [provider authentication docs](../index.md#authentication) for details.

## Example Usage

The provider's local Terraform name is `tailscale-membership` (with a dash). Resource types use underscores (`tailscale_membership_*`) per HCL identifier rules, so every membership block MUST carry `provider = tailscale-membership` to override the implicit binding to the upstream `tailscale` provider. The first example below includes the full `required_providers` alias block; subsequent snippets omit it for brevity.

```terraform
terraform {
  required_providers {
    tailscale-membership = {
      source  = "pmpaulino/tailscale-membership"
      version = "~> 0.1"
    }
  }
}

resource "tailscale_membership_tailnet_membership" "alice" {
  provider   = tailscale-membership
  login_name = "alice@example.com"
  role       = "member"
}
```

Additional patterns:

```terraform
# Admin role
resource "tailscale_membership_tailnet_membership" "bob_admin" {
  provider   = tailscale-membership
  login_name = "bob@example.com"
  role       = "admin"
}

# Disable (suspend) and re-enable via suspended
resource "tailscale_membership_tailnet_membership" "carol_paused" {
  provider   = tailscale-membership
  login_name = "carol@example.com"
  role       = "member"
  suspended  = true
}

# Optional: downgrade instead of remove on destroy
resource "tailscale_membership_tailnet_membership" "dave_admin" {
  provider             = tailscale-membership
  login_name           = "dave@example.com"
  role                 = "admin"
  downgrade_on_destroy = true
}
```

## Validation

The `login_name` attribute is validated at plan time:

- It MUST be a well-formed email address (e.g., `alice@example.com`).
- Display-name forms such as `"Alice <alice@example.com>"` are rejected; this is an identity field, not an RFC 5322 mailbox header.
- Validation errors are **not idempotent**: a second plan with the same invalid input keeps erroring, and no Tailscale API call is made.

The `role` attribute is restricted to exactly `member` or `admin`. Any other value (including legacy Tailscale role names such as `owner`, `it-admin`, `network-admin`, `auditor`, `billing-admin`) is rejected at plan time. Widening the supported set is intentionally out of scope for this resource.

## Pending invitations and expiry

If the Tailscale Control API still lists an invitation for the configured `login_name` — even when its `expires` timestamp is in the past — this provider reports `state = "pending"`. An expired-but-listed invite remains pending until it is explicitly removed (either by the operator destroying the resource, or by the invite being deleted out-of-band in the Tailscale admin console). The provider does not attempt to garbage-collect expired invites.

## Last administrator and account owner protection

The Tailscale Control API rejects any attempt to remove or suspend the last administrator or the account owner. When this happens, the provider surfaces the API's refusal as a Terraform diagnostic whose message explicitly mentions **"last admin or account owner"** so the cause is unambiguous. Enforcement is API-side; the provider does not perform a proactive admin count.

## Migration from the upstream-derived prototype

If you are currently managing memberships via the prototype `tailscale_tailnet_membership` resource (feature 001 of this repository, before the v0.1 fork), follow the [migration guide](https://github.com/pmpaulino/terraform-provider-tailscale-membership/blob/main/specs/002-standalone-membership-provider/quickstart.md#4-migration-from-the-upstream-derived-prototype) to rename resource types and `terraform state mv` your existing state.

## Import

Import an existing membership by tailnet and login name (email):

```shell
terraform import 'tailscale_membership_tailnet_membership.alice' 'example.com:alice@example.com'
```

Import ID format: `tailnet:login_name` (e.g. `example.com:alice@example.com`).

## Schema

### Required

- `login_name` (String) The identity (email) for the membership. Used to match an invite or user in the tailnet. MUST be a well-formed email address (e.g., `alice@example.com`).

### Optional

- `downgrade_on_destroy` (Boolean) If true, on destroy the user is downgraded to member or suspended instead of removed. Defaults to `false`.
- `role` (String) The role to assign. Use `member` or `admin`. Defaults to `member`.
- `suspended` (Boolean) When true, the membership is disabled (user suspended). When false, the user is active. Defaults to `false`.

### Read-Only

- `id` (String) The ID of this resource (`tailnet:login_name`).
- `invite_id` (String) The Tailscale user invite ID when state is `pending`.
- `state` (String) Current state: `pending` (invite not yet accepted), `active`, or `disabled` (suspended).
- `user_id` (String) The Tailscale user ID when state is `active` or `disabled`.
