# Research: Tailscale User Management (Phase 0)

**Feature**: 001-tailscale-user-management  
**Spec**: [spec.md](./spec.md)

## 1. Tailscale API ↔ Spec Mapping

| Spec concept | Tailscale API | Notes |
|--------------|----------------|------|
| Ensure membership (invite when needed) | `POST /tailnet/{tailnet}/user-invites` (create invite with email, role) | If user already exists, API may return conflict; we treat as idempotent. If invite exists for email, list invites and no-op create. |
| Pending membership | `GET /tailnet/{tailnet}/user-invites` → find by email | UserInvite has email (or similar); list and match. |
| Active membership | `GET /tailnet/{tailnet}/users` (or `GET /users/{userId}`) | Existing data_source_users / data_source_user; user has role, status. |
| Disable user | `POST /users/{userId}/suspend` | Enterprise/Personal; OAuth scope `users`. |
| Re-enable user | `POST /users/{userId}/restore` | Same. |
| Remove membership | Pending: `DELETE /user-invites/{userInviteId}`. Active: `POST /users/{userId}/delete` | Idempotent: 404 on delete is success. |
| Role (member/admin) | Create invite: role in body. Update: `PATCH /users/{userId}/role` | API supports member/admin and others (owner, it-admin, etc.); spec uses member/admin. |
| Last admin / account owner | API may reject suspend/delete for owner or last admin; return clear error → map to FR-012 message. | No separate API; rely on API error and surface as clear message. |
| Downgrade on destroy | Optional: instead of delete user, call `PATCH /users/{userId}/role` to "member" or suspend. | Align with GitHub downgrade_on_destroy. |

**Decision**: Use a single Terraform resource `tailscale_tailnet_membership` keyed by a stable identifier. Resource ID in state: use **tailnet + login_name (email)** as the Terraform resource ID (e.g. `tailnet_xxx:user@example.com`) so that the same identity is always the same resource whether it is pending invite or active user.

**Rationale**: Matches GitHub membership (org + username). Tailscale does not expose a single “membership” ID that spans invite vs user; we derive membership by listing invites and users and matching on email/login_name.

**Alternatives considered**: (1) Two resources (invite + user) — rejected to match spec’s single-membership concept. (2) Use invite ID or user ID as Terraform ID — rejected because when an invite is accepted, the invite disappears and we get a user ID; switching ID in state would force replace; keying by email keeps one resource across lifecycle.

---

## 2. Tailscale Go Client (tailscale.com/client/tailscale/v2)

**Decision**: Use existing client `tailscale.com/client/tailscale/v2`. Provider already uses `client.Users().List`, `client.Users().Get`. For user invites and user suspend/restore/delete/role we must use the same client’s API surface.

**Rationale**: No new dependencies (constitution). If the current client does not expose UserInvites or user suspend/restore/delete/role, add calls via the same HTTP client the provider uses (or extend the client dependency within the same repo/module if the client is vendored or a direct dependency).

**Alternatives considered**: (1) Separate HTTP client for Control API — rejected to avoid duplication. (2) Fork tailscale client — only if upstream lacks methods; prefer contributing or using internal HTTP calls.

**Action**: Verify in implementation that `tailscale.com/client/tailscale/v2` exposes (or can be extended for): ListUserInvites, CreateUserInvite, GetUserInvite, DeleteUserInvite; User Suspend, Restore, Delete, UpdateRole. If not, document in tasks and implement via provider’s existing HTTP base (e.g. client’s low-level request API).

---

## 7. Client API Verification (T001 – Implementation Finding)

**Verified**: 2025-02-07 (implementation phase).

| Capability | tailscale.com/client/tailscale/v2 v2.7.0 | Notes |
|------------|------------------------------------------|--------|
| Users().List(ctx, userType, role) | ✅ Exposed | Used by data_source_users, data_source_user. |
| Users().Get(ctx, id) | ✅ Exposed | Used by data_source_user. |
| UserInvites (list, create, get, delete) | ❌ Not exposed | No `Client.UserInvites()`; no type `UserInvite` in package. |
| User suspend | ❌ Not exposed | UsersResource has only Get and List. |
| User restore | ❌ Not exposed | Same. |
| User delete | ❌ Not exposed | Same. |
| User update role (PATCH) | ❌ Not exposed | Same. |

**Conclusion**: The v2 client does **not** expose UserInvites or user suspend/restore/delete/role. The provider must implement these operations via direct HTTP calls to the Tailscale Control API, using the same `*tailscale.Client` instance: its exported fields `BaseURL`, `Tailnet`, `APIKey`, `Auth`, and `HTTP` are sufficient to build and send authenticated requests to `/tailnet/{tailnet}/user-invites`, `/user-invites/{id}`, and `/users/{userId}/suspend`, `/users/{userId}/restore`, `/users/{userId}/delete`, `PATCH /users/{userId}/role`. Implement an internal API helper (e.g. in `tailscale/` or a small `api/` helper) that constructs requests, applies the client’s auth, and uses the client’s HTTP client (or default).

---

## 3. GitHub Membership Flow Alignment

**Decision**: Implement Create/Update as a single “ensure” path (like GitHub’s EditOrgMembership): if no user and no pending invite for the email → create invite; if pending invite exists → no-op; if user exists → update role if needed. Delete: if pending invite → delete invite; if user → delete user (or downgrade/suspend if downgrade_on_destroy is set).

**Rationale**: Spec explicitly requires this pattern (reference model: GitHub provider membership). Idempotency and single-resource semantics are preserved.

**Alternatives considered**: Separate “invite” vs “member” resources — rejected per spec.

---

## 4. Identity Key: login_name (Email)

**Decision**: Use `login_name` (email) as the user-facing identity for the resource (e.g. `login_name` attribute and part of resource ID). Tailscale user invites are created with an email; users have `login_name`. Normalize to lowercase for comparison if needed.

**Rationale**: Spec says “identity (e.g. email)”; Tailscale API uses email for invites and login_name for users. One canonical field avoids ambiguity.

**Alternatives considered**: Use user ID only — rejected because before acceptance we only have an invite, not a user ID.

---

## 5. OAuth Scopes and API Keys

**Decision**: Document that full membership management requires OAuth scopes `UserInvites` and `users` (and that creating invites may require user-owned API keys per Tailscale docs). Provider does not enforce scopes; API errors are surfaced as clear, actionable messages (FR-012).

**Rationale**: Matches spec assumption that authorization is defined by the tailnet/API; we only surface errors clearly.

---

## 6. Error Handling and FR-012

**Decision**: On API errors (4xx/5xx, not found, rate limit), return Terraform diag errors with a clear, actionable message (e.g. “Failed to create user invite: …; check that your token has UserInvites scope and that the identity is valid.”). Use existing `diagnosticsError`-style helpers used elsewhere in the provider.

**Rationale**: Spec FR-012 requires clear, actionable failure messages.

---

## 8. Access revocation on disable/remove (FR-007)

**Decision**: FR-007 ("revoke or disconnect a user's access when they are disabled or removed") is satisfied by the underlying Tailscale Control API. The provider does NOT implement any access-revocation mechanism of its own:

- **Disable** → `POST /users/{userId}/suspend` causes Tailscale to invalidate the user's auth keys and disconnect their devices.
- **Remove** (active user) → `POST /users/{userId}/delete` removes the user record; all tokens become invalid.
- **Remove** (pending invite) → `DELETE /user-invites/{userInviteId}` invalidates the invite link.

**Rationale**: Tailscale's Control API is authoritative for session/device lifecycle; duplicating that in the provider would be both wrong (we can't revoke server-side sessions from a client) and a source of drift.

**Implication for tests**: FR-007 is tested implicitly by asserting the correct API endpoint is invoked in T011/T015 (suspend), T012/T015 (restore), T016/T019 (delete user), and T007/T019 (delete invite). No additional FR-007-specific test is needed.

---

## 9. Concurrent administrator changes (FR-010)

**Decision**: FR-010 ("last write wins on concurrent ops; no conflict error") is satisfied for free by two layered guarantees:

1. **Terraform serializes Create/Update/Delete** for the same resource within a single workspace (state lock + sequential `apply`).
2. **The Tailscale Control REST API is non-locking** for user-state mutations: each `PATCH role`, `POST suspend`, `POST restore`, `POST delete` accepts the request unconditionally and the most recent successful call wins.

**Rationale**: No provider-side locking, no optimistic-concurrency tokens (e.g., `If-Match`), and no conflict-detection logic are required. Cross-workspace or cross-operator races resolve at the API layer with last-write-wins semantics.

**Implication for tests**: FR-010 has no provider-side behavior to test in isolation. A documented decision in this section is the artifact that satisfies the spec MUST.
