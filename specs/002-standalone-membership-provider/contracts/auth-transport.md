# Contract: Membership Helper ↔ v2 Client Auth Transport (Phase 1)

**Feature**: 002-standalone-membership-provider  
**Spec**: [spec.md](../spec.md) (FR-006, FR-007)  
**Research**: [research.md §R1](../research.md)

## Purpose

Pin down, in a testable form, how the membership API helper (`tailscale/membership_api.go`) MUST interact with the v2 client's pluggable `Auth` interface so that all three documented auth modes (API key, OAuth, Federated Identity) work end-to-end without the helper itself knowing which mode is active.

## Contract

### Inputs

- `*tailscale.Client` produced by the provider's `ConfigureContextFunc` (`tailscale/provider.go::providerConfigure`). One of three valid shapes:
  1. `APIKey != ""`, `Auth == nil`.
  2. `Auth: *tailscale.OAuth{...}`, `APIKey == ""`.
  3. `Auth: *tailscale.IdentityFederation{...}`, `APIKey == ""`.

### Pre-conditions

Before the helper performs any HTTP request:

- The v2 client's `init()` MUST have run. The helper triggers this by calling any of the v2 client's resource accessors — concretely `_ = m.Client.Users()` at the top of `do()`. After this call:
  - For inputs (2) and (3), `m.Client.HTTP` is the `*http.Client` returned by `m.Client.Auth.HTTPClient(originalHTTP, baseURL)`. Its `Transport` is auth-decorated.
  - For input (1), `m.Client.HTTP` is a plain `*http.Client` with the v2 client's default 1-minute timeout, and `m.Client.APIKey` is preserved for the v2 client's own API-key handling.

### Behavior

For every helper method (`listUserInvites`, `createUserInvite`, `deleteUserInvite`, `suspendUser`, `restoreUser`, `deleteUser`, `updateUserRole`):

- The outgoing `*http.Request` MUST be dispatched via `m.Client.HTTP.Do(req)`.
- The helper MUST NOT call `req.SetBasicAuth(...)` itself. (The v2 client's own request pipeline injects API-key Basic auth when `Auth == nil`; the helper relying on that is FR-013-compatible because `Auth.HTTPClient`'s decoration runs at the round-tripper layer for OAuth / Federated Identity, while API-key mode is handled separately by the v2 client's request-build path.)
- The helper MUST set `Content-Type: application/json`, `Accept: application/json`, and `User-Agent: m.Client.UserAgent` headers (only `User-Agent` is conditional on non-empty value).

### Post-conditions

- For input (2) (OAuth): the request that reaches the Tailscale Control API carries an `Authorization: Bearer <oauth-access-token>` header injected by the OAuth `HTTPClient`'s `RoundTripper`. No `Authorization: Basic` header.
- For input (3) (Federated Identity): the request carries the `Authorization` header constructed by the Identity Federation flow's round-tripper. No `Authorization: Basic` header.
- For input (1) (API key): the request carries `Authorization: Basic <base64(apikey:)>` set by the v2 client's request pipeline. No duplicate Basic auth set by the helper.

### Failure modes

- If `m.Client.Auth.HTTPClient(...)` returns a client whose `RoundTripper` returns an error (e.g. token-exchange failure), `m.Client.HTTP.Do(req)` propagates that error to the helper's caller. The helper MUST surface it as-is in the returned `error`; the resource layer translates it into a Terraform diagnostic per FR-012 (feature 001).

## Test (must exist; authored before the fix lands)

Located at `tailscale/membership_api_test.go`.

### `TestMembershipAPI_RoutesThroughAuthHTTPClient`

For each helper method:

1. Construct a stub `tailscale.Auth` whose `HTTPClient(orig *http.Client, baseURL string) *http.Client` returns an `*http.Client` with a custom `RoundTripper` that:
   - Records the inbound `*http.Request`.
   - Adds a marker header `X-Test-Auth-Mode: stub`.
   - Returns a canned `*http.Response` with status `200 OK` and a JSON body matching the helper method's expected response shape.
2. Build `&tailscale.Client{ BaseURL: u, Tailnet: "test", Auth: stub, ... }` and wrap it with `membershipAPI(...)`.
3. Invoke the helper method.
4. Assert: the recorded request contains `X-Test-Auth-Mode: stub` (i.e. the stub round-tripper was used) AND has no `Authorization: Basic` header (i.e. the helper did not bypass the auth transport).

### `TestMembershipAPI_APIKeyStillUsesBasicAuth`

For each helper method:

1. Build `&tailscale.Client{ BaseURL: u, Tailnet: "test", APIKey: "test-key", HTTP: <test-server-backed http.Client> }` (no `Auth`).
2. Invoke the helper method against an `httptest.Server` that records the `Authorization` header.
3. Assert: the recorded `Authorization` header equals `Basic <base64("test-key:")>` (verifying the regression-safety: API-key mode still works after the fix).

Coverage requirement: 100% of branches in the modified `do()` function (Constitution v1.1.0 §VIII). Both tests MUST exist and MUST fail before the fix lands; both MUST pass after.
