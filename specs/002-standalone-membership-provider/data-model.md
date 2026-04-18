# Data Model: Standalone Tailscale Membership Provider (Phase 1)

**Feature**: 002-standalone-membership-provider  
**Spec**: [spec.md](./spec.md)  
**Plan**: [plan.md](./plan.md)

## Canonical reference

The data model for the membership entity itself — attributes, types, required/optional/computed flags, state machine, validation rules, idempotency rules, and the mapping to the Tailscale Control API — is fully defined by feature 001 in [`specs/001-tailscale-user-management/data-model.md`](../001-tailscale-user-management/data-model.md). This feature delivers the same entity unchanged.

Per the spec's FR-009 ("The membership resource exposed by this provider MUST be behaviorally equivalent to the membership resource specified in feature 001"), the only deltas in v0.2 of this feature are operator-visible identifier renames, listed below.

## v0.1 deltas (this feature only)

### Resource type identifier

| Field | Feature 001 (upstream-derived prototype) | Feature 002 (standalone provider v0.1) |
|---|---|---|
| Provider source address | (n/a — same as upstream) | `pmpaulino/tailscale-membership` |
| Local provider type (HCL) | `tailscale` | `tailscale_membership` (set via `required_providers` alias because the source contains a dash) |
| Resource type (HCL) | `tailscale_tailnet_membership` | `tailscale_membership_tailnet_membership` |
| Go module path of the provider | `github.com/tailscale/terraform-provider-tailscale` | `github.com/pmpaulino/terraform-provider-tailscale-membership` |

The Terraform resource ID format (`{tailnet}:{login_name}`, e.g. `tailnet_abc123:alice@example.com`) is **unchanged**. State files migrated via `terraform state mv` (see `quickstart.md` "Migration") preserve the same ID and therefore the same underlying tailnet object reference.

### Resource attributes

**Unchanged** from feature 001. Reproduced here for convenience:

| Schema attribute | Type | Mode | Notes |
|------------------|------|------|-------|
| `login_name` | string | required | Email identity. Validated as RFC 5322 mailbox at plan time (FR-001a in feature 001). |
| `role` | string | optional, default `"member"` | Allowlist `{member, admin}` only (FR-005a in feature 001). |
| `downgrade_on_destroy` | bool | optional, default `false` | If true, on destroy downgrade or suspend instead of remove. |
| `state` | string | computed | One of `pending`, `active`, `disabled`. |
| `invite_id` | string | computed | Tailscale user invite ID when state is `pending`. |
| `user_id` | string | computed | Tailscale user ID when state is `active` or `disabled`. |

### State machine

**Unchanged** from feature 001. The transitions (`pending → active`, `active ↔ disabled`, any → absent) and their idempotency rules are inherited verbatim.

### Validation rules

**Unchanged** from feature 001. `login_name` mailbox validation, `role` allowlist enforcement, last-admin/account-owner protection (API-side, surfaced as a clear diagnostic), and the non-idempotency of validation errors all carry over.

### Mapping to the Tailscale Control API

**Unchanged** from feature 001. See [`specs/001-tailscale-user-management/data-model.md` §"Mapping to Tailscale API"](../001-tailscale-user-management/data-model.md#mapping-to-tailscale-api). The HTTP transport used to invoke those API operations changes (per R1 in `research.md` and FR-006/FR-007), but the operations themselves and their request/response shapes do not.

## Provider-configuration data (this feature only)

The provider configuration block is *not* a Terraform resource and was not specified by feature 001's data model. It is documented here because v0.2 ships it as part of the provider's user-visible surface (US1 acceptance scenario 2: "the provider configuration block is present").

### Configuration arguments

All argument names, types, environment-variable defaults, and conflict rules match the upstream provider verbatim per FR-004. No semantic changes; only `tailnet` defaulting behavior is noted as a feature-001-backlog item (see `backlog.md`).

| Argument | Type | Source(s) | Default | Sensitive | Conflicts with |
|---|---|---|---|---|---|
| `api_key` | string | env `TAILSCALE_API_KEY`, HCL | `""` | yes | `oauth_client_id`, `oauth_client_secret`, `identity_token` |
| `oauth_client_id` | string | env `TAILSCALE_OAUTH_CLIENT_ID` or `OAUTH_CLIENT_ID`, HCL | `""` | no | `api_key` |
| `oauth_client_secret` | string | env `TAILSCALE_OAUTH_CLIENT_SECRET` or `OAUTH_CLIENT_SECRET`, HCL | `""` | yes | `api_key`, `identity_token` |
| `identity_token` | string | env `TAILSCALE_IDENTITY_TOKEN` or `IDENTITY_TOKEN`, HCL | `""` | yes | `api_key`, `oauth_client_secret` |
| `scopes` | list(string) | HCL | `[]` | no | only valid with OAuth or Federated Identity |
| `tailnet` | string | env `TAILSCALE_TAILNET`, HCL | `"-"` (provider rejects empty; "-" means "tailnet that owns the credentials") | no | — |
| `base_url` | string | env `TAILSCALE_BASE_URL`, HCL | `"https://api.tailscale.com"` | no | — |
| `user_agent` | string | HCL | derived (`terraform-provider-tailscale-membership/{version}`) | no | — |

### Auth-mode selection rules

The `validateProviderCreds` precedence is **unchanged** from upstream and matches the spec's FR-008. Three valid configurations:

1. **API key**: `api_key` set; OAuth/Federated-Identity fields empty.
2. **OAuth client credentials**: `oauth_client_id` + `oauth_client_secret` set; `api_key` and `identity_token` empty.
3. **Federated Identity**: `oauth_client_id` + `identity_token` set; `api_key` and `oauth_client_secret` empty.

Any other combination yields a clear diagnostic and configuration is rejected. Per FR-008, no mode is silently selected.

### Auth-transport contract

After `ConfigureContextFunc` returns, the resulting `*tailscale.Client` carries either:

- `APIKey` set, `Auth` nil — for mode 1; OR
- `Auth: *tailscale.OAuth{...}`, `APIKey` empty — for mode 2; OR
- `Auth: *tailscale.IdentityFederation{...}`, `APIKey` empty — for mode 3.

The membership helper (`tailscale/membership_api.go`) MUST trigger the v2 client's `init()` (which installs the `Auth.HTTPClient` decoration on `c.HTTP`) before issuing any request. See [`contracts/auth-transport.md`](./contracts/auth-transport.md) for the formal contract and the test that enforces it.
