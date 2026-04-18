# Backlog: Carried-over Findings from Feature 001

**Feature**: 002-standalone-membership-provider  
**Spec ref**: [spec.md FR-010](./spec.md)  
**Status**: All items deferred — NOT v0.1 blockers per FR-010.

This file records the five remediation findings that surfaced during feature 001's `/speckit.analyze` review. Per FR-010 they MUST be visible in this feature's planning artifacts (and therefore in PR review) but MUST NOT be merged into `tasks.md` so that v0.1 task execution does not accidentally pick them up.

Each item carries: a one-line summary, a link to its origin in feature 001 (the relevant FR or document section), the user-visible impact, and a `Status:` field.

---

## B-001 — Last-admin / account-owner pre-flight

**Summary**: The membership resource currently relies entirely on the Tailscale Control API to refuse destroy/disable operations that would remove the last admin or the account owner. There is no client-side pre-flight that warns the operator at `terraform plan` time.

**Origin**: `specs/001-tailscale-user-management/spec.md` FR-009 ("Last administrator and account owner protection") and `data-model.md` validation rules. Feature-001 plan and tests confirm enforcement is API-side only.

**Impact**: Operators discover the failure at `terraform apply`, after the destroy is already in flight, rather than at `plan`. The diagnostic from the Control API is surfaced verbatim — informative but reactive.

**Possible v0.2+ fix**: At plan time, when `Destroy` is the planned action and the resource's role is `admin`, optionally enumerate the tailnet's admins via the v2 client and warn (not error) if this resource is the last admin. The warning, not error, preserves API-side enforcement as the source of truth (per the spec's "API-side; the provider does not perform a proactive admin count" wording in `docs/resources/tailnet_membership.md`).

**Status**: `Deferred to v0.2+`.

---

## B-002 — Error-surfacing in `downgrade_on_destroy` paths

**Summary**: When `downgrade_on_destroy = true` and the destroy path fails partway through (e.g. role PATCH succeeds but the subsequent suspend fails), the provider returns a generic error without clearly indicating which sub-step failed and what the resource's actual residual state is in the tailnet.

**Origin**: `specs/001-tailscale-user-management/data-model.md` "Mapping to Tailscale API" Delete section ("If downgrade_on_destroy → set role to member or suspend") and feature-001 `tasks.md` tests for the destroy path.

**Impact**: Operators must consult the Tailscale admin console to determine the residual state, then `terraform import` to re-acquire it. Recovery is manual.

**Possible v0.2+ fix**: Wrap each sub-step of the destroy path in a labeled error (`role_downgrade_failed`, `suspend_failed`) and surface a structured diagnostic listing which sub-steps succeeded and which did not, with explicit instructions on how to re-acquire state.

**Status**: `Deferred to v0.2+`.

---

## B-003 — Test-assertion strength

**Summary**: Several feature-001 unit tests assert success/failure of the API flow without checking specific request payloads (e.g. that the role PATCH body actually carries the correct `role` value, or that `downgrade_on_destroy` produces a PATCH then a POST in that order rather than two POSTs). Tests pass even if the helper sends syntactically valid but semantically wrong requests.

**Origin**: `specs/001-tailscale-user-management/tasks.md` Phase 3.x test scaffolding and the existing `tailscale/resource_tailnet_membership_test.go`.

**Impact**: Latent risk of regression-invisibility: a wire-format change in the helper could pass tests yet break against the live Control API.

**Possible v0.2+ fix**: Tighten each helper-method test to assert against the recorded request URL, method, and JSON body; add a single end-to-end test that drives the resource through `Create → Update(role) → Update(suspended) → Destroy` and asserts the exact request sequence.

**Status**: `Deferred to v0.2+`.

---

## B-004 — Pending-update behavior

**Summary**: The current Read mapping classifies any backend-listed invitation as `state = "pending"` regardless of expiry (FR-008 in feature 001 — intentional design choice). However, the resource's Update path silently no-ops when state is `pending` and the operator changes `role`, because role updates target the user-id which does not yet exist for a pending invitation. The no-op is undocumented; operators changing `role` on a pending membership see "no changes" and are surprised when the eventual user accepts the invite at the original role.

**Origin**: `specs/001-tailscale-user-management/spec.md` FR-008 + `data-model.md` Update section ("If state is active or disabled and role changed → PATCH user role"). The pending-state behavior is unspecified beyond FR-008.

**Impact**: A pending-state role change is silently dropped. The operator's HCL says `admin`; the eventual accepted user is `member`.

**Possible v0.2+ fix**: When the operator changes `role` on a `pending` membership, delete the pending invite and re-create it with the new role (preserving the existing invite-resend semantics), or surface a plan-time warning that role changes on pending memberships will not propagate until the invite is accepted.

**Status**: `Deferred to v0.2+`.

---

## B-005 — Tailnet fallback removal

**Summary**: The provider currently defaults `tailnet` to `"-"` (the v2 client's "use the tailnet that owns the credentials" sentinel) when neither the HCL argument nor the `TAILSCALE_TAILNET` environment variable is set. This is convenient for single-tailnet operators but masks misconfiguration in multi-tailnet environments — operators who forget to set `tailnet` operate on whichever tailnet their credentials happen to own, rather than getting a clear error.

**Origin**: `tailscale/provider.go` schema definition for `tailnet` (`DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_TAILNET", "-")`) and `providerConfigure` rejection of empty-string `tailnet`. The `"-"` default is the upstream behavior and was carried into feature 001 unchanged.

**Impact**: Cross-tailnet apply mistakes are easier than they should be. The diagnostic "tailscale provider argument 'tailnet' is empty" is unreachable as long as the default is `"-"`.

**Possible v0.2+ fix**: Drop the `"-"` default; require `tailnet` to be set explicitly (HCL or env). For single-tailnet operators, document `tailnet = "-"` as the explicit opt-in to the credential-owning-tailnet behavior. This is a breaking change and is marked v0.2 specifically because v0.1 preserves upstream argument semantics per FR-004.

**Status**: `Deferred to v0.2+` (and explicitly flagged as a breaking change for v0.2 release notes).

---

## Out-of-scope reminders

- This backlog MUST NOT be merged into `tasks.md` (FR-010). The `tasks.md` produced by `/speckit.tasks` covers v0.1 deliverables only.
- These items MUST NOT be silently dropped (FR-010). They are tracked here, in-repo, so that PR reviewers see them at every release.
- v0.2's planning artifacts will lift any item the maintainer chooses to address into that feature's `spec.md` / `tasks.md`.
