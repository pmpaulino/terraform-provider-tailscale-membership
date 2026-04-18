# Data Model: Tailnet Membership (Phase 1)

**Feature**: 001-tailscale-user-management  
**Spec**: [spec.md](./spec.md)

## Entity: Membership

Single entity for the Terraform resource. Represents one identity’s membership (or pending invite) in a tailnet.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | computed | Terraform resource ID: `{tailnet}:{login_name}` (e.g. `tailnet_xxx:user@example.com`) |
| login_name | string | required | Identity (email). Used to match invite or user in Tailscale. |
| role | string | optional | One of `member`, `admin`. Default `member`. |
| state | string | computed | `pending` \| `active` \| `disabled` (suspended). |
| downgrade_on_destroy | bool | optional | If true, on destroy downgrade to member or suspend instead of removing. Default false. |

### State transitions

- **pending**: Invitation created via API; not yet accepted. (Tailscale: user invite exists.)
- **active**: User has accepted and is a member. (Tailscale: user exists, not suspended.)
- **disabled**: User is suspended. (Tailscale: user suspended via suspend endpoint.)

Removing the resource (destroy) transitions to “absent”: either invite deleted or user deleted (or downgraded/suspended if downgrade_on_destroy).

### Uniqueness

- One membership per (tailnet, login_name). Terraform resource ID encodes both.

### Validation rules (from spec)

- **login_name**: MUST be a well-formed email. Validated client-side (schema `ValidateDiagFunc`) before any HTTP call. Invalid input returns a plan-time error and is **not** idempotent — repeating the call with the same invalid input MUST again error (FR-001a).
- **role**: MUST be exactly one of `{member, admin}`. Any other value (e.g. `owner`, `it-admin`, `network-admin`, `auditor`, custom roles) is rejected at plan time even if the underlying Tailscale API would accept it (FR-005a). Other roles are out of scope for this feature.
- **Last admin / account owner**: do not allow destroy/disable that would remove or disable the last admin or the account owner; return clear error (FR-009).
- **Idempotency**: create when already member or pending → no-op, success. Update role when unchanged → no-op. Delete when already absent → no-op. Idempotency does NOT apply to validation errors above.

## Terraform resource schema (tailscale_tailnet_membership)

| Schema attribute | Type | Mode | Description |
|------------------|------|------|-------------|
| login_name | string | required | Email (identity) for the membership. |
| role | string | optional, default "member" | `member` or `admin`. |
| downgrade_on_destroy | bool | optional, default false | On destroy, downgrade/suspend instead of remove. |
| state | string | computed | `pending`, `active`, or `disabled`. |
| invite_id | string | computed | Tailscale user invite ID when state is pending. Opaque. |
| user_id | string | computed | Tailscale user ID when state is active or disabled. Opaque. |

Resource ID in state: `tailnetID:login_name` (e.g. `tailnet_abc123:alice@example.com`) so that import and lifecycle are stable.

## Mapping to Tailscale API

- **Read**: List user invites for tailnet; list users for tailnet. Find by login_name (invite email or user login_name). Set state from invite vs user and user status (suspended). Per FR-008, an invitation that is still listed by the backend MUST be reported as `state = "pending"` regardless of any expiry timestamp; it transitions to absent only once the backend no longer lists it.
- **Create**: If no user and no invite for login_name → create user invite (POST user-invites) with role. Else → no-op (idempotent).
- **Update**: If state is active or disabled and role changed → PATCH user role. If state is disabled and desired is active → restore. If state is active and desired is disabled → suspend. (Spec: disable/re-enable are first-class; role update is part of ensure.)
- **Delete**: If pending → DELETE user invite. If active/disabled and not downgrade_on_destroy → DELETE user. If downgrade_on_destroy → set role to member or suspend (per option semantics).
