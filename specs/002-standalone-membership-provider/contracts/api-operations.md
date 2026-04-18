# API Operations Used: Tailscale Control API (Phase 1)

**Feature**: 002-standalone-membership-provider  
**Status**: Cite-only — inherited from feature 001 unchanged.

## Canonical reference

This feature uses the same set of Tailscale Control API operations as feature 001. The operations, methods, paths, scopes, and request/response bodies are documented in [`specs/001-tailscale-user-management/contracts/api-operations.md`](../../001-tailscale-user-management/contracts/api-operations.md).

## v0.1 deltas

**None** for the API surface itself.

The only change in v0.1 is the **HTTP transport** that carries these requests: per FR-006/FR-007 (and `research.md §R1`), the membership helper MUST route requests through the v2 client's `Auth.HTTPClient`-decorated `*http.Client` rather than constructing its own `http.DefaultClient`-backed request with helper-side Basic auth. The contract for that transport is documented in [`auth-transport.md`](./auth-transport.md). The wire format of every request and response is identical to feature 001.

## Scopes

Unchanged from feature 001:

- `UserInvites` — required for `POST /tailnet/{tailnet}/user-invites`, `GET /tailnet/{tailnet}/user-invites`, `DELETE /user-invites/{id}`.
- `users` — required for `GET /tailnet/{tailnet}/users`, `POST /users/{id}/suspend`, `POST /users/{id}/restore`, `POST /users/{id}/delete`, `POST /users/{id}/role`.

The provider documentation (`docs/index.md` and `docs/resources/tailnet_membership.md`) MUST list both scopes as required for OAuth and Federated Identity modes.
