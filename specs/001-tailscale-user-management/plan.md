# Implementation Plan: Tailscale User Management (Membership)

**Branch**: `001-tailscale-user-management` | **Date**: 2026-02-07 (revised) | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `specs/001-tailscale-user-management/spec.md`. Extend the Terraform Tailscale provider with user management following the same flow as the GitHub provider membership resource.

## Summary

Deliver a single Terraform resource that represents **tailnet membership** (one membership per identity). Creating the resource ensures the identity is in the tailnet: if not present, the provider creates a user invite (Tailscale API sends the invitation); if already a member or pending invite, the operation is idempotent. The same resource supports state *pending*, *active*, and *disabled* (Tailscale: suspended). Destroy cancels a pending invite or removes the user (with optional downgrade-on-destroy).

Per the Session 2026-02-07 clarifications, the resource MUST:

- **Validate identity before any API call** (FR-001a): malformed/unsupported `login_name` returns a clear validation error and is **not** idempotent — repeating the call with the same invalid input MUST again error.
- **Constrain `role` to `{member, admin}`** (FR-005a): any other value (e.g. `owner`, `it-admin`, `network-admin`, `auditor`, custom) is rejected at plan time, even if the underlying Tailscale API accepts it.
- **Treat expired-but-listed invites as `pending`** (FR-008): while the backend still lists an expired invite, Read MUST report `state = "pending"`; a subsequent ensure on the same identity is a no-op until the invite drops off the backend listing (then a new invite is created).

Implementation uses the existing Tailscale API (user invites, users list/get, suspend, restore, delete, role update) and the existing `tailscale.com/client/tailscale/v2` client, adding a new resource `tailscale_tailnet_membership` and reusing existing data sources where appropriate. Operations not exposed by the v2 client (UserInvites, suspend/restore/delete, role PATCH) are issued via direct HTTP using the client's existing auth and HTTP fields (see [research.md §7](./research.md)).

## Technical Context

**Language/Version**: Go 1.25.x (per `go.mod`)  
**Primary Dependencies**: `hashicorp/terraform-plugin-sdk/v2`, `tailscale.com/client/tailscale/v2` (v2.7.0)  
**Storage**: N/A (Tailscale Control API is source of truth)  
**Testing**: Go testing + terraform-plugin-sdk; 100% coverage required for new/modified code (constitution v1.1.0 §VIII)  
**Target Platform**: Terraform 1.x; provider runs in Terraform CLI environment  
**Project Type**: Terraform provider (single Go module)  
**Performance Goals**: Normal provider apply/read latency; no special targets beyond API responsiveness  
**Constraints**:
- OAuth scopes MUST include `UserInvites` and `users` for full membership management; user-owned API keys required for creating invites (per Tailscale API notes).
- `login_name` MUST be a well-formed email; validated client-side before any HTTP call (FR-001a).
- `role` schema MUST use `ValidateFunc` (or equivalent) restricting to `{member, admin}` (FR-005a).
- Read MUST classify invitations returned by the backend as `state = "pending"` regardless of expiry timestamp (FR-008).

**Scale/Scope**: One resource type; tens to hundreds of memberships per tailnet typical.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design. Per constitution v1.1.0 "Constitution Check Discipline" each row cites a concrete artifact or behavior (no rubber-stamping).*

| Principle | Status | Concrete artifact / behavior |
|-----------|--------|------------------------------|
| I. Build Small, Test-Driven Steps | Pass | New code lives in `tailscale/resource_tailnet_membership.go` (+ `_test.go`); test-first ordering enforced per `tasks.md` Phase 3.1–3.3. Modified function = `provider.go` resource registration map (one-line edit, covered by existing acceptance test). |
| II. Keep Code Simple and Clear | Pass | Single resource file `tailscale/resource_tailnet_membership.go`; CRUD delegated to a thin HTTP helper at `tailscale/membership_api.go` (built via `membershipAPI(client)`) reusing the existing `*tailscale.Client` auth/HTTP fields (see `research.md §7`). No new abstraction layers; identity validation is a single `ValidateDiagFunc` on the `login_name` schema (FR-001a). |
| III. Embrace Feedback and Iteration | Pass | Provider feedback surfaces (per constitution III): docs at `docs/resources/tailnet_membership.md`, examples at `examples/resources/tailscale_tailnet_membership/{resource.tf,import.sh}`, GitHub Issues for user-reported drift. (No CHANGELOG.md exists in this repo — release notes are produced via `.github/workflows/release.yml`; the row no longer claims a CHANGELOG artifact.) |
| IV. Automate Relentlessly | Pass | CI workflow `.github/workflows/ci.yml` runs `go test ./...` and lint; release workflow `.github/workflows/release.yml` cuts releases. New resource picks these up without new pipeline steps. Acceptance tests gated by `TF_ACC` per constitution VIII. |
| V. Design for Change | Pass | Role allowlist is the inline literal `[]string{"member", "admin"}` at `tailscale/resource_tailnet_membership.go:57` (passed to `validation.StringInSlice`); widening the set is a one-line change. Tailscale API calls funnel through `tailscale/membership_api.go` so an API change is localized to that file. |
| VI. Optimize for Communication and Learning | Pass | `data-model.md` documents the membership ↔ Tailscale API mapping; `research.md §1, §7` explain decisions and v2-client gaps; `quickstart.md` is the user-facing guide; commit message will reference FR IDs (FR-001…FR-012). |
| VII. Minimal Dependencies | Pass | Zero new module dependencies; `go.mod` unchanged. HTTP helper uses `net/http` (stdlib) and the existing `tailscale.com/client/tailscale/v2` for auth headers. |
| VIII. Complete Test Coverage | Pass | New code (`resource_tailnet_membership.go`, helper) at 100% line/branch via unit tests using the existing fake client pattern (see `tailscale/acl_test.go` for prior art). Acceptance test (`TestAccTailscaleTailnetMembership_*`) gated by `TF_ACC`, runs in scheduled CI per constitution VIII. Untouched files keep current coverage baseline. |

No violations. Complexity tracking table left empty.

### Post-Design Re-check (after Phase 1 artifacts)

Re-checked 2026-02-07 against `data-model.md`, `research.md`, `quickstart.md`, and the Session 2026-02-07 clarifications. The added validation requirements (FR-001a, FR-005a) are satisfied by schema-level `ValidateDiagFunc`s — no new package, no new dependency, no behavioral coupling. FR-008 (expired-but-listed = pending) is satisfied entirely inside the Read mapping in `data-model.md §"Mapping to Tailscale API"`. **No principle status changes.**

## Project Structure

### Documentation (this feature)

```text
specs/001-tailscale-user-management/
├── plan.md              # This file
├── research.md          # Phase 0 (Tailscale API ↔ spec mapping; client capabilities)
├── data-model.md        # Phase 1 (Membership entity, states, Terraform schema)
├── quickstart.md        # Phase 1 (Usage examples)
├── contracts/           # Phase 1 (API operations used)
└── tasks.md             # Phase 2 (/speckit.tasks output)
```

### Source Code (repository root)

```text
tailscale/
├── resource_tailnet_membership.go      # NEW: membership resource (Create/Read/Update/Delete)
├── resource_tailnet_membership_test.go # NEW: tests (100% coverage on new/modified code)
├── provider.go                         # UPDATE: register tailscale_tailnet_membership
├── data_source_user.go                 # existing
├── data_source_users.go                # existing
├── resource_tailnet_key.go             # existing (reference pattern)
└── ... (other existing resources/data sources)

docs/
└── resources/
    └── tailnet_membership.md           # NEW: provider docs for tailscale_tailnet_membership

examples/
└── resources/
    └── tailscale_tailnet_membership/   # NEW: HCL example + import.sh
```

**Structure Decision**: The provider is a single Go module under `tailscale/`. New code is one resource file and one test file; provider registration adds one entry. Documentation follows the existing `docs/resources/` pattern. No new packages or services.

## Complexity Tracking

*No constitution violations. Table not used.*
