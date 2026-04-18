# Phase 0 Research: Standalone Tailscale Membership Provider

**Feature**: 002-standalone-membership-provider  
**Plan**: [plan.md](./plan.md)

This document resolves the technical unknowns surfaced in `plan.md`'s Technical Context section. Each topic ends with **Decision / Rationale / Alternatives considered**.

---

## R1. Auth-routing fix mechanics in `membership_api.go`

### Background

`tailscale.com/client/tailscale/v2.Client` exposes auth via the pluggable `Auth` interface (`client.go` lines 24–30):

```go
// Auth is a pluggable mechanism for authenticating requests.
type Auth interface {
    // HTTPClient builds an http.Client that uses orig as a starting point and
    // adds its own authentication to outgoing requests.
    HTTPClient(orig *http.Client, baseURL string) *http.Client
}
```

`Client.init()` (lines 98–127) lazily wires the auth-aware HTTP client:

```go
if c.Auth != nil {
    c.APIKey = ""
    c.HTTP = c.Auth.HTTPClient(c.HTTP, c.BaseURL.String())
}
```

`init()` runs `sync.Once`, triggered by any of the public resource accessors (`c.Users()`, `c.Devices()`, etc.). Until one of them is called, `c.HTTP` is `nil` and `c.Auth` has not been consulted.

The current `tailscale/membership_api.go` `do()` method (lines 44–74) bypasses all of this:

```go
if m.Client.APIKey != "" {
    req.SetBasicAuth(m.Client.APIKey, "")
}
httpClient := m.Client.HTTP
if httpClient == nil {
    httpClient = http.DefaultClient
}
return httpClient.Do(req)
```

Behavior under each auth mode (as configured in `provider.go`):

| Auth mode | `m.Client.APIKey` | `m.Client.Auth` | `m.Client.HTTP` (before fix) | Outgoing request |
|-----------|-------------------|-----------------|------------------------------|------------------|
| API key | non-empty | nil | nil → `http.DefaultClient` | Basic auth header set by helper. **Works.** |
| OAuth | empty | `*tailscale.OAuth` | nil → `http.DefaultClient` (auth-decorated client never installed because `init()` never ran) | No auth header. **Broken.** |
| Federated Identity | empty | `*tailscale.IdentityFederation` | nil → `http.DefaultClient` (same reason) | No auth header. **Broken.** |

This is the FR-006 / FR-007 correctness bug.

### Decision

Two changes inside `tailscale/membership_api.go`:

1. **At the top of `do()`** (before reading `m.Client.HTTP`), call `_ = m.Client.Users()` (or any other accessor; `Users()` is semantically closest to the membership domain). This triggers `Client.init()` exactly once across the helper's lifetime, installing the auth-decorated `*http.Client` on `m.Client.HTTP` for OAuth and Federated Identity, and a plain timeout-only `*http.Client` for API-key mode.

2. **Delete the `if m.Client.APIKey != "" { req.SetBasicAuth(...) }` branch.** After `init()`, the API-key path still works because the v2 client transparently sends `Authorization: Basic <base64(apiKey:)>` via the same `do()` path it uses for its own resource methods. Confirmed by reading `client.go::buildRequest` and `client.go::do`, which add the API-key header inside `c.HTTP.Do(req)` only when `c.Auth == nil` and `c.APIKey != ""` — using the standard `net/http.Request.SetBasicAuth` semantics.

After the fix, the helper's auth handling is uniform across all three modes: it just calls `m.Client.HTTP.Do(req)` and lets the v2 client's `Auth`/`APIKey` configuration do its job.

### Rationale

- **Single source of truth for auth.** All three auth modes flow through the same v2-client extension point (`Auth.HTTPClient` for OAuth/Federated Identity; built-in API-key handling for the simple case). The helper does not need to know which mode is active.
- **One-line fix per FR-006/FR-007.** No new abstraction; no new dependency; no behavioral risk to existing API-key callers.
- **Forward compatible.** If upstream adds a fourth auth mode (e.g. mutual TLS), the helper inherits it for free.

### Alternatives considered

- **Replicate `init()` logic locally inside the helper.** Rejected: brittle (would diverge if upstream changes its init logic), and gains nothing over calling a public accessor.
- **Switch the helper to use `Client`'s public methods only (no direct `do()`).** Rejected for v0.1 because the v2 client does not expose UserInvites or the suspend/restore/delete user actions used by the membership resource (this is the original reason `membership_api.go` exists; see feature 001 `research.md §7`). Re-using the client's `*http.Client` is the smallest change.
- **Have the helper accept a pre-built `*http.Client` from the caller.** Rejected: pushes the auth wiring into `provider.go`'s `ConfigureContextFunc`, duplicating the v2 client's own logic.

### Test (authored before the fix)

`tailscale/membership_api_test.go::TestMembershipAPI_RoutesThroughAuthHTTPClient`:

- Constructs a `tailscale.Client` with a `Auth: stubAuth{}` whose `HTTPClient(orig, baseURL) *http.Client` returns an `*http.Client` with a `RoundTripper` that records the outgoing request and returns a canned response.
- Calls each helper method (`listUserInvites`, `createUserInvite`, `deleteUserInvite`, `suspendUser`, `restoreUser`, `deleteUser`, `updateUserRole`).
- Asserts the recorded request carries a marker header injected by `stubAuth`'s round-tripper. (i.e. the request flowed through the auth-decorated client.)
- Asserts that no `Authorization: Basic ...` header is present (i.e. the helper did not add Basic auth itself).
- A second test variant uses `APIKey: "test-key"` (no `Auth`) and asserts the request carries `Authorization: Basic <base64("test-key:")>` (i.e. API-key mode still works after the fix).

Coverage target: 100% of branches in `do()`, including the `c.Auth != nil` and `c.Auth == nil + APIKey != ""` paths.

---

## R2. Module-rename mechanics

### Decision

Three sequenced edits, plus a verification step:

1. **`go mod edit -module github.com/pmpaulino/terraform-provider-tailscale-membership`** — rewrites the `module` line in `go.mod`.
2. **Find/replace import paths** across the codebase: `github.com/tailscale/terraform-provider-tailscale` → `github.com/pmpaulino/terraform-provider-tailscale-membership`. Affected files (post-deletion): `main.go`, every `tailscale/*.go` left after Phase A deletions, and `.goreleaser.yml`'s `ldflags` (the `-X github.com/tailscale/terraform-provider-tailscale/tailscale.providerVersion=...` flag becomes `-X github.com/pmpaulino/terraform-provider-tailscale-membership/tailscale.providerVersion=...`).
3. **`go mod tidy`** — drops indirect modules no longer reachable from the trimmed import graph. Specifically: the indirect `tailscale.com v1.94.1` module disappears once `data_source_4via6.go` (uses `tailscale.com/net/tsaddr`) and `data_source_device_test.go` (uses `tailscale.com/tstest`) are deleted in the Phase A removals.
4. **Verification**: `git grep "github.com/tailscale/terraform-provider-tailscale"` MUST return zero matches. `go build ./...` and `go test ./...` MUST pass. The membership resource type registered in `provider.go::Provider().ResourcesMap` MUST be exactly `tailscale_membership_tailnet_membership` (not `tailscale_tailnet_membership`).

### Rationale

- `go mod edit -module` is the canonical, scriptable way to rename; it preserves the rest of `go.mod`.
- `go mod tidy` is the Constitution VII enforcement mechanism: it makes "minimal dependencies" mechanically verifiable.
- Renaming the resource type at the same time as the module is intentional: both are operator-visible identifier changes from the upstream baseline, and reviewing them together makes the intent of the change set obvious.

### Alternatives considered

- **Keep the upstream resource type `tailscale_tailnet_membership` to ease migration.** Rejected: doing so would force operators to also keep the upstream local provider type `tailscale`, which conflicts with the new source address `pmpaulino/tailscale-membership` and makes the alias requirement non-obvious. The migration guide (FR-021) handles the rename cleanly via `terraform state mv`.
- **Keep upstream module path and fork only the GitHub URL.** Rejected: violates FR-012 and would cause Go module collisions for any downstream consumer importing both fork and upstream.

---

## R3. GitHub Actions tag-filter for production + pre-release patterns

### Background

FR-014 mandates two tag patterns trigger releases:

- Production: `v<MAJOR>.<MINOR>.<PATCH>` (e.g. `v0.1.0`).
- Pre-release: `v<MAJOR>.<MINOR>.<PATCH>-(alpha|beta|rc).<N>` (e.g. `v0.1.0-rc.1`).

GitHub Actions `on.push.tags` supports glob patterns, *not* full regular expressions. Two-stage filtering is required because the production form `v*.*.*` would also accidentally match `v0.1.0-rc.1` (since `*` matches dots and dashes alike inside a tag name).

### Decision

Use a single tag glob plus a job-level `if` guard that re-applies a stricter regex match on `github.ref_name`. Concrete `.github/workflows/release.yml` snippet:

```yaml
on:
  push:
    tags:
      - "v[0-9]+.[0-9]+.[0-9]+"
      - "v[0-9]+.[0-9]+.[0-9]+-alpha.[0-9]+"
      - "v[0-9]+.[0-9]+.[0-9]+-beta.[0-9]+"
      - "v[0-9]+.[0-9]+.[0-9]+-rc.[0-9]+"
```

GitHub Actions glob: `[0-9]` is supported as a character class. Each line is an independent pattern; a tag matching any one of them triggers the workflow. Tags that do not match any pattern (e.g. `release-2026-04`, `v0.1`, `v0.1.0-foo`) are silently ignored, satisfying the FR-014 "tags not matching either pattern MUST NOT trigger a release" requirement.

Pre-release marking is handled by GoReleaser's `release.prerelease: auto`, which already promotes any tag containing a `-` to GitHub's "pre-release" flag. This satisfies the FR-014 "pre-release tags MUST be marked as 'pre-release' in the GitHub Release UI" requirement without extra workflow logic.

### Rationale

- Glob filters are evaluated by GitHub Actions before the workflow is dispatched, so non-matching tags incur zero CI cost.
- Listing the four pre-release variants explicitly (`-alpha.N`, `-beta.N`, `-rc.N`) is clearer than a single `v*-*` pattern that would also match malformed tags.
- Reusing GoReleaser's `prerelease: auto` keeps the pre-release/latest distinction in one place (the release tool), rather than splitting it between Actions and GoReleaser.

### Alternatives considered

- **One blanket `v*` pattern + an `if: startsWith(github.ref, 'refs/tags/v')` guard.** Rejected: lets typo'd tags (`v0.1.0-foo`) trigger the workflow before failing inside GoReleaser; FR-014 wants non-matching tags to never trigger.
- **Use a separate workflow per pattern.** Rejected: duplicates the entire release job for no benefit.

---

## R4. NOTICE-file format for MIT-derived projects

### Background

MIT itself does not require a `NOTICE` file (unlike Apache-2.0). FR-023 requires one anyway as a clear, scannable attribution surface — the `LICENSE` file's MIT text plus the upstream copyright lines is technically sufficient under MIT, but a separate `NOTICE` makes the fork relationship obvious at a glance.

### Decision

Ship `NOTICE` at the repository root with this structure:

```text
terraform-provider-tailscale-membership
Copyright (c) 2026 Pedro Paulino

This product includes software developed by:

    terraform-provider-tailscale
    https://github.com/tailscale/terraform-provider-tailscale
    Copyright (c) 2021 David Bond
    Copyright (c) 2024 Tailscale Inc & Contributors

Licensed under the MIT License (see LICENSE).

This is a hard fork of the upstream `terraform-provider-tailscale`
project, scoped to provide only Tailscale tailnet membership
management. Substantial portions of the membership resource and
the underlying HTTP helper are derived from the upstream project.
```

`LICENSE` retains the MIT text with both upstream copyright lines (current contents are already correct; no edit needed).

### Rationale

- Names the upstream project, links to its source, reproduces its copyright lines, and states the relationship in plain English (FR-023).
- Two-line statement of derivation lets license-scanning tools and human reviewers immediately classify the relationship.
- Pattern adapted from the Apache 2.0 NOTICE convention, which is the most widely understood attribution-file format and degrades gracefully for MIT.

### Alternatives considered

- **Skip `NOTICE` and rely on `LICENSE` alone.** Rejected: technically permissible under MIT, but FR-023 explicitly requires it, and it materially helps reviewers and license scanners.
- **Inline the upstream LICENSE text inside `NOTICE`.** Rejected: duplication risks divergence from `LICENSE`; the `NOTICE` should *attribute*, not *re-license*.

---

## R5. GoReleaser platform-matrix enumeration (FR-014)

### Background

FR-014 locks in 11 OS/arch pairs:

```text
linux/amd64, linux/arm64, linux/386, linux/arm,
darwin/amd64, darwin/arm64,
windows/amd64, windows/386,
freebsd/amd64, freebsd/arm, freebsd/386
```

The current `.goreleaser.yml` declares `goos: [freebsd, windows, linux, darwin]` × `goarch: [amd64, "386", arm, arm64]` minus the single ignore `darwin/386`. That cross-product yields 15 pairs and includes `freebsd/arm64` and `windows/arm64` and `windows/arm`, which are NOT in the FR-014 set.

### Decision

Replace the cross-product with explicit `ignore` exclusions to get exactly the 11 pairs:

```yaml
builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: "{{ .CommitTimestamp }}"
    flags:
      - -trimpath
    ldflags:
      - "-s -w -X github.com/pmpaulino/terraform-provider-tailscale-membership/tailscale.providerVersion={{.Version}} -X main.commit={{.Commit}}"
    goos: [linux, darwin, windows, freebsd]
    goarch: [amd64, arm64, "386", arm]
    ignore:
      - { goos: darwin, goarch: "386" }
      - { goos: darwin, goarch: arm }
      - { goos: windows, goarch: arm64 }
      - { goos: windows, goarch: arm }
      - { goos: freebsd, goarch: arm64 }
    binary: "{{ .ProjectName }}_v{{ .Version }}"
```

Project name comes from the repository name (`terraform-provider-tailscale-membership`) by default, which is what GoReleaser uses for `binary` and `archives.name_template`. No explicit `project_name:` line is needed if the GitHub repo is renamed to match.

A test in `tasks.md` will run `goreleaser release --snapshot --clean` (no GPG required for snapshot) and assert the produced archives count is exactly 11, with names matching the expected pattern. This pins FR-014 to a verifiable build-time check rather than a paper requirement.

### Rationale

- Explicit `ignore` list is auditable: each excluded pair has a one-line entry that a reviewer can map back to FR-014.
- Using `{{ .ProjectName }}` lets GoReleaser pick up the binary name from the repo name, so the rename in R2 propagates automatically.
- Snapshot-mode build in `tasks.md` makes FR-014's "11 archives, no partial releases" mechanically testable in CI without needing a real Git tag or GPG key.

### Alternatives considered

- **Enumerate each pair as a separate `builds:` entry.** Rejected: 11× duplication for no gain; `ignore` is the GoReleaser-idiomatic way to exclude unsupported pairs.
- **Drop FreeBSD entirely.** Rejected: FR-014 explicitly includes the three FreeBSD pairs (matches the Terraform Registry "first-class" set).

---

## Summary of Phase 0 outputs

| Topic | Decision | Affects |
|---|---|---|
| R1. Auth-routing fix | Trigger `c.init()` via a v2 client accessor; drop helper-side Basic auth | `tailscale/membership_api.go` (+ new `_test.go`) |
| R2. Module rename | `go mod edit` + import sweep + `go mod tidy` | `go.mod`, every `*.go` import block, `.goreleaser.yml` ldflags |
| R3. Tag-filter pattern | List four glob patterns under `on.push.tags`; reuse GoReleaser `prerelease: auto` | `.github/workflows/release.yml`, `.goreleaser.yml` |
| R4. NOTICE file | Apache-style attribution adapted to MIT, names upstream + reproduces upstream copyright | `NOTICE` (new) |
| R5. Platform matrix | Explicit `ignore` list yielding exactly 11 pairs; `goreleaser --snapshot` test | `.goreleaser.yml`, `tasks.md` |

No NEEDS CLARIFICATION markers remain. Ready for Phase 1.
