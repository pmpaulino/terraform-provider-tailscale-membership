# Implementation Plan: Standalone Tailscale Membership Provider

**Branch**: `002-standalone-membership-provider` | **Date**: 2026-04-18 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `specs/002-standalone-membership-provider/spec.md`

## Summary

This v0.1 transforms the upstream-derived repository into a single-purpose hard fork that exposes only the tailnet membership resource designed in feature `001-tailscale-user-management`. The plan has six concrete strands:

1. **Delete** every non-membership resource and data source (DNS, ACLs, devices, keys, webhooks, posture, contacts, settings, AWS external IDs, OAuth clients, federated-identity *resource*, logstream, plus the `tailscale_user`/`tailscale_users`/`tailscale_device`/`tailscale_devices`/`tailscale_4via6`/`tailscale_acl` data sources) along with their tests, docs pages, and examples directories. The auth-mode named "Federated Identity" stays in `provider.go`; only the standalone `tailscale_federated_identity` resource is removed.
2. **Rename** the Go module from `github.com/tailscale/terraform-provider-tailscale` to `github.com/pmpaulino/terraform-provider-tailscale-membership` (FR-012) and the Terraform local provider type from `tailscale` to `tailscale_membership`, making the membership resource type `tailscale_membership_tailnet_membership` (FR-011). Source address stays `pmpaulino/tailscale-membership` and is aliased in every `required_providers` example.
3. **Fix** the auth-routing correctness bug in `tailscale/membership_api.go` (FR-006/FR-007). The membership helper currently sets API-key Basic auth on the request itself and uses `m.Client.HTTP` (which is `nil` until `init()` runs, so OAuth/Federated-Identity authenticated transports are never installed and falls back to `http.DefaultClient`). The fix is to trigger `c.init()` by touching any of the v2 client's resource accessors (e.g. `m.Client.Users()`) before reading `m.Client.HTTP`, then drop the Basic-auth shortcut and let the v2 client's `Auth.HTTPClient` decoration handle credentials uniformly across all three modes.
4. **Refresh** the operator-facing surfaces: rewrite `docs/index.md` for the new provider name and required `required_providers` alias; keep the resource page (`docs/resources/tailnet_membership.md`) but rename headings to `tailscale_membership_tailnet_membership`; replace the README with a v0.1-shaped one (purpose, install, auth modes, migration guide pointer); ship `examples/resources/tailscale_membership_tailnet_membership/{resource.tf,import.sh}`; add a `NOTICE` attributing the upstream project (FR-022/FR-023).
5. **Tighten** the release pipeline. Update `.goreleaser.yml` to: change `ldflags` import path; switch `ProjectName` to `terraform-provider-tailscale-membership`; explicitly enumerate the 11 OS/arch pairs from FR-014 and ignore unsupported combinations; keep the existing GPG signing block (which already fails the build when `GPG_FINGERPRINT` is unset, satisfying FR-016); update `.github/workflows/release.yml`'s `tags` filter from the permissive `v*` to two patterns covering production (`v[0-9]+.[0-9]+.[0-9]+`) and pre-release (`v[0-9]+.[0-9]+.[0-9]+-(alpha|beta|rc).[0-9]+`) tags so non-matching tags are silently ignored as required by FR-014.
6. **Track** the five remediation findings carried over from feature 001 as a dedicated `specs/002-standalone-membership-provider/backlog.md`. They are explicit non-goals for v0.1 (FR-010) but must remain visible in PR review.

This plan is intentionally about *deletion + renaming + one targeted bug fix* rather than new feature work. The membership resource itself is delivered unchanged from feature 001; FR-009 enforces behavioral parity. Per the Constitution v1.1.0 §I scoping rule, mechanical refactors (module-path renames, import rewrites, file deletions) are exempt from test-first ordering as long as the existing tests stay green; only the auth-routing fix in `membership_api.go` must be authored test-first.

## Technical Context

**Language/Version**: Go 1.25.x (per existing `go.mod`; will be preserved across the module rename).  
**Primary Dependencies**: `github.com/hashicorp/terraform-plugin-sdk/v2 v2.38.2`, `tailscale.com/client/tailscale/v2 v2.7.0`. After deletions, the indirect `tailscale.com v1.94.1` module drops out (only used by the `data_source_4via6` and `data_source_device` files, both deleted), bringing the runtime dep tree closer to FR-013's stated minimum.  
**Storage**: N/A (Tailscale Control API is the source of truth).  
**Testing**: `go test ./...` with the existing fake-client patterns under `tailscale/`; 100% coverage required for new + modified code per Constitution v1.1.0 §VIII. Acceptance tests (`TF_ACC`) are gated by env var per the same section's provider-acceptance allowance.  
**Target Platform**: Terraform 1.x runtime (provider invoked via the Plugin SDK v2 over plugin protocol 5). Release artifacts cover the 11-platform OS/arch matrix locked in FR-014.  
**Project Type**: Single Go module — Terraform provider.  
**Performance Goals**: Normal Terraform provider latency; no targets beyond what the Tailscale Control API supports.  
**Constraints**:

- Module path MUST be `github.com/pmpaulino/terraform-provider-tailscale-membership` and every `import` line in the codebase MUST reflect that path; no leftover `github.com/tailscale/terraform-provider-tailscale` strings allowed (FR-012).
- The membership API helper MUST route through the v2 client's authenticated `*http.Client` (the one installed by `c.Auth.HTTPClient(...)` inside `c.init()`); it MUST NOT add API-key Basic auth itself when the auth mode is OAuth or Federated Identity (FR-006/FR-007).
- Local provider type registered with the Plugin SDK MUST be `tailscale_membership` so the resource type identifier matches the documented `tailscale_membership_tailnet_membership` (FR-011).
- Release pipeline MUST gate on the production and pre-release tag patterns and MUST fail without `GPG_FINGERPRINT`; non-matching tags MUST NOT trigger a release (FR-014, FR-016).
- Only retained upstream Go dependency from the Tailscale ecosystem is `tailscale.com/client/tailscale/v2` (FR-013).

**Scale/Scope**: One Terraform resource (`tailscale_membership_tailnet_membership`), zero data sources, one provider configuration block. Six P1–P3 user stories, 25 functional requirements, 7 success criteria. Net code change is dominated by file deletions (≈ 30 `_test.go` + resource files removed, ≈ 18 docs pages removed, ≈ 16 example dirs removed); the only substantive logic change is one bug fix in `membership_api.go`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design. Per Constitution v1.1.0 "Constitution Check Discipline" each row cites a concrete artifact or behavior (no rubber-stamping).*

| Principle | Status | Concrete artifact / behavior |
|-----------|--------|------------------------------|
| I. Build Small, Test-Driven Steps | Pass | The only behavioral change is the auth-routing fix in `tailscale/membership_api.go`. A failing test (`TestMembershipAPI_RoutesThroughAuthHTTPClient` in `tailscale/membership_api_test.go`) is authored first per `tasks.md` Phase 3.x; the test installs a stub `tailscale.Auth` whose `HTTPClient` wraps a `RoundTripper` that records each request and asserts every membership operation's request flows through it. File deletions and the module-path rename are pure refactors with no behavior change and are exempt from test-first ordering per Constitution v1.1.0 §I "Scope for extensions"; existing membership tests remain the safety net (must stay green after the refactor). |
| II. Keep Code Simple and Clear | Pass | Auth fix is two changes in `membership_api.go`: (a) call `m.Client.Users()` (or any other resource accessor) once at the top of `do()` to trigger `c.init()`, then read `m.Client.HTTP` (which is now the auth-decorated client); (b) delete the `if m.Client.APIKey != ""` Basic-auth branch — API-key auth is already wrapped into the v2 client's behavior when `Auth == nil`. No new abstraction layers; no new packages. Module-rename edits are mechanical (`go mod edit -module ...` + a single sed-equivalent `gopls rename` on the import path). |
| III. Embrace Feedback and Iteration | Pass | Provider feedback surfaces (per Constitution III): docs at `docs/index.md` and `docs/resources/tailnet_membership.md`; runnable example at `examples/resources/tailscale_membership_tailnet_membership/{resource.tf,import.sh}`; release notes via `goreleaser` (`changelog: use: github`); GitHub Issues as the canonical bug-report channel; iteration signal = GitHub label `auth/transport` on issues — any operator report that OAuth or Federated Identity auth misroutes triggers a follow-up release. README MUST link to the issue tracker (FR-020). **Deprecation path — N/A justification**: Constitution III mandates that user-affecting behavior changes ship as a deprecation warning in at least one minor release before removal. The resource-type rename `tailscale_tailnet_membership` → `tailscale_membership_tailnet_membership` (FR-011) and the module-path rename (FR-012) qualify as such changes, but v0.1.0 is the **first published release** of this fork — there is no prior version of `pmpaulino/tailscale-membership` in which to land a deprecation warning. The migration guide (FR-021, materialized in `quickstart.md` §4 and cross-linked from the README) is the substitute communication mechanism. From v0.2 onward, any user-affecting change MUST follow the standard one-minor-release deprecation path. |
| IV. Automate Relentlessly | Pass | CI workflow `.github/workflows/ci.yml` (unchanged) keeps running `go test ./...` on PRs. Release workflow `.github/workflows/release.yml` is updated to gate on the two tag patterns from FR-014 and to fail loudly without `GPG_FINGERPRINT` (the existing `crazy-max/ghaction-import-gpg` step already fails on missing key, satisfying FR-016). GoReleaser's `signs:` block requires `GPG_FINGERPRINT` to be set or the run aborts. Acceptance tests gated by `TF_ACC` per Constitution VIII allowance. |
| V. Design for Change | Pass | The auth fix is the *removal* of a hard-coded Basic-auth branch in favor of the v2 client's documented `Auth.HTTPClient(orig, baseURL) *http.Client` extension point (see `tailscale.com/client/tailscale/v2/client.go` lines 24–30, 112–115). Any future change to upstream auth modes flows through that interface without touching the membership helper. The `.goreleaser.yml` platform matrix is enumerated as a list of `goos`+`goarch` pairs so adding/removing a platform is a one-line edit. |
| VI. Optimize for Communication and Learning | Pass | `data-model.md` references feature 001's `specs/001-tailscale-user-management/data-model.md` as canonical and lists only v0.2 deltas (resource type identifier change). `research.md` documents (a) the v2 client's `Auth` extension contract and the exact hook used by the fix, (b) the GoReleaser ldflags/project-name implications of the module rename, (c) the GitHub Actions tag-filter syntax used for FR-014. `quickstart.md` doubles as the migration guide (FR-021). All commit messages MUST reference FR IDs (e.g. "fix(membership): route via v2 client auth transport (FR-006/FR-007)"). |
| VII. Minimal Dependencies | Pass | **Net dependency reduction.** Zero new Go modules. The deletions of `data_source_4via6.go` (uses `tailscale.com/net/tsaddr`) and `data_source_device_test.go` (uses `tailscale.com/tstest`) eliminate the only direct usages of the indirect `tailscale.com v1.94.1` module; `go mod tidy` after deletions MUST drop it from `go.mod`. After this feature, the only retained upstream Tailscale Go dep is `tailscale.com/client/tailscale/v2 v2.7.0`, exactly as FR-013 mandates. |
| VIII. Complete Test Coverage | Pass | New test (`tailscale/membership_api_test.go::TestMembershipAPI_RoutesThroughAuthHTTPClient`) covers 100% of new branches in the modified `do()` function, including the path where `c.Auth != nil`. Modified function (`membership_api.go::do`) brought to 100% coverage in the same change set. Removed files take their tests with them — no orphaned coverage to maintain. The kept-code coverage baseline (membership resource + helper + provider `ConfigureContextFunc`) MUST NOT regress; CI's `go test -coverprofile=coverage.out ./tailscale/...` plus the existing `coverage.out` baseline at HEAD provide the non-regression check (Constitution VIII "baseline file"). Acceptance test `TestAccTailscaleTailnetMembership_*` (already in `resource_tailnet_membership_test.go`) continues to gate on `TF_ACC`. |

**No violations. Complexity tracking table left empty.**

### Post-Design Re-check (after Phase 1 artifacts)

Re-checked 2026-04-18 against `research.md`, `data-model.md`, `contracts/`, `quickstart.md`, and `backlog.md`. No new packages introduced; no new dependencies; the auth-routing fix is exactly the one-line `init()`-trigger documented in `research.md §1`; the data model is unchanged from feature 001; contracts are inherited unchanged. The platform matrix and tag-filter changes are localized to `.goreleaser.yml` and `.github/workflows/release.yml`. **No principle status changes.**

## Project Structure

### Documentation (this feature)

```text
specs/002-standalone-membership-provider/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 (auth fix mechanics, GoReleaser/Actions config, NOTICE format)
├── data-model.md        # Phase 1 (cites feature 001 as canonical; lists v0.2 deltas only)
├── quickstart.md        # Phase 1 (install + auth + migration guide; doubles as FR-021 source)
├── contracts/           # Phase 1 (Tailscale Control API operations used; cites feature 001)
├── backlog.md           # NEW (FR-010 — five remediation findings carried over from feature 001)
├── checklists/
│   └── requirements.md  # /speckit.specify quality checklist (already complete)
└── tasks.md             # Phase 2 (created by /speckit.tasks — NOT created here)
```

### Source Code (repository root, post-feature)

```text
tailscale/
├── provider.go                              # UPDATE: ResourcesMap shrinks to one entry; DataSourcesMap empties; provider local type stays inferred from ProviderFunc registration
├── provider_test.go                         # UPDATE: keep configure-credentials tests; drop tests for removed resources
├── resource_tailnet_membership.go           # KEEP unchanged (carried from feature 001)
├── resource_tailnet_membership_test.go      # KEEP unchanged
├── membership_api.go                        # FIX: route through v2 client's authenticated HTTP (FR-006/FR-007)
├── membership_api_test.go                   # NEW: TestMembershipAPI_RoutesThroughAuthHTTPClient + path-by-path test for each helper method
└── tailscale_test.go                        # UPDATE: trim to fixtures still referenced by membership tests

# DELETED in this feature (resources):
#   resource_acl.go, resource_acl_test.go
#   resource_aws_external_id.go, resource_aws_external_id_test.go
#   resource_contacts.go, resource_contacts_test.go
#   resource_device_authorization.go, resource_device_authorization_test.go
#   resource_device_key.go, resource_device_key_test.go
#   resource_device_subnet_routes.go, resource_device_subnet_routes_test.go
#   resource_device_tags.go, resource_device_tags_test.go
#   resource_dns_configuration.go, resource_dns_configuration_test.go
#   resource_dns_nameservers.go, resource_dns_nameservers_test.go
#   resource_dns_preferences.go, resource_dns_preferences_test.go
#   resource_dns_search_paths.go, resource_dns_search_paths_test.go
#   resource_dns_split_nameservers.go, resource_dns_split_nameservers_test.go
#   resource_federated_identity.go, resource_federated_identity_test.go     # NOTE: this is the "Federated Identity" *resource* only; the auth mode of the same name stays in provider.go
#   resource_logstream_configuration.go, resource_logstream_configuration_test.go
#   resource_oauth_client.go, resource_oauth_client_test.go
#   resource_posture_integration.go, resource_posture_integration_test.go
#   resource_tailnet_key.go, resource_tailnet_key_test.go
#   resource_tailnet_settings.go, resource_tailnet_settings_test.go
#   resource_webhook.go, resource_webhook_test.go
#
# DELETED in this feature (data sources):
#   data_source_4via6.go, data_source_4via6_test.go
#   data_source_acl.go, data_source_acl_test.go
#   data_source_device.go, data_source_device_test.go
#   data_source_devices.go, datasource_devices_test.go
#   data_source_user.go
#   data_source_users.go, data_source_users_test.go

docs/
├── index.md                                 # REWRITE: new provider name, alias note, link to migration guide and signing key
└── resources/
    └── tailnet_membership.md                # KEEP, rename resource type in headings/examples to tailscale_membership_tailnet_membership; add the required_providers alias to every example
# DELETED docs/resources/* and docs/data-sources/* for everything other than the membership resource page.

examples/
└── resources/
    └── tailscale_membership_tailnet_membership/
        ├── resource.tf                      # NEW path (renamed from examples/resources/tailscale_tailnet_membership/); content updated to use the required_providers alias and the new resource type
        └── import.sh                        # NEW path; same import format, updated resource address
# DELETED examples/resources/* for every other resource and examples/data-sources/* in full.

main.go                                      # UPDATE: import path → github.com/pmpaulino/terraform-provider-tailscale-membership/tailscale; ProviderFunc registration unchanged in shape
go.mod                                       # UPDATE: module path; run `go mod tidy` after deletions to drop indirect tailscale.com v1.94.1
.goreleaser.yml                              # UPDATE: ldflags var path → github.com/pmpaulino/terraform-provider-tailscale-membership/tailscale.providerVersion; explicit goos/goarch matrix per FR-014; ignores: include freebsd/arm64, linux/arm64 special cases as needed
.github/workflows/release.yml                # UPDATE: tags filter from "v*" to a list with two glob patterns matching FR-014 production + pre-release forms (use GitHub Actions push.tags glob; the release-pattern test is a tasks.md item)
LICENSE                                      # KEEP (MIT, retain upstream copyright lines per FR-022)
NOTICE                                       # NEW (FR-023): names upstream `terraform-provider-tailscale` repo, links to https://github.com/tailscale/terraform-provider-tailscale, reproduces upstream copyright
README.md                                    # REWRITE per FR-020 + FR-021 (purpose, install, auth modes, migration guide w/ before-/after-HCL + `terraform state mv` commands)
```

**Structure Decision**: The repository remains a single Go module — Terraform provider. The shape of the codebase doesn't change; the *count* of files does. No new packages, no new directories under `tailscale/`. The membership-only surface area means a new operator opening the repo sees a tiny `tailscale/` directory (≈ 6 files) instead of the upstream's ≈ 50, which is itself a documentation win and aligned with Constitution II ("Keep Code Simple and Clear").

## Phases

### Phase 0: Outline & Research

Output: `specs/002-standalone-membership-provider/research.md` resolving every Technical Context unknown. Topics:

1. **Auth-routing fix mechanics**: how `tailscale.com/client/tailscale/v2.Client.init()` installs an authenticated `*http.Client` via `Auth.HTTPClient(orig, baseURL)`; why the membership helper currently bypasses it; the minimal change to route through it without breaking API-key auth.
2. **Module rename mechanics**: `go mod edit -module`, `find/sed` of import paths, GoReleaser `ldflags` var-path implications, `ProjectName` derivation from the binary name, GitHub Actions cache key invalidation.
3. **GitHub Actions tag-filter for two release patterns**: how to express both `v[0-9]+.[0-9]+.[0-9]+` and `v[0-9]+.[0-9]+.[0-9]+-(alpha|beta|rc).[0-9]+` under `on: push: tags:` (Actions uses glob patterns, not regex; the production form is `v*.*.*` plus a `!v*-*` exclusion paired with a separate pre-release filter).
4. **NOTICE-file format for MIT-derived projects**: standard layout (project name, fork origin URL, retained upstream copyright lines, statement of derivation) drawn from common practice (Apache NOTICE conventions adapted to MIT).
5. **GoReleaser platform-matrix enumeration**: how to express the 11-pair set with `goos`/`goarch` lists and `ignore` exclusions; verify each entry actually builds (already covered by upstream config; we just lock the set explicitly).

Each topic resolves with: **Decision / Rationale / Alternatives considered**.

**Output**: `research.md` with all NEEDS CLARIFICATION resolved (this plan has none).

### Phase 1: Design & Contracts

**Prerequisites**: `research.md` complete.

1. **`data-model.md`**: cites `specs/001-tailscale-user-management/data-model.md` as canonical for the membership entity, schema, state transitions, validation rules, and Tailscale-API mapping. Lists v0.2 deltas: only the resource type identifier changes (`tailscale_tailnet_membership` → `tailscale_membership_tailnet_membership`); attribute names, types, computed/required/optional flags, and resource-ID format are unchanged.

2. **`contracts/`**: cites `specs/001-tailscale-user-management/contracts/api-operations.md` as canonical for the Tailscale Control API operations consumed by the membership helper (UserInvites and Users tags). No new operations are added in this feature; the helper's HTTP transport is changed but the operations themselves are unchanged. A short `contracts/auth-transport.md` documents the contract between the membership helper and the v2 client's `Auth` interface (input: `*tailscale.Client` after `init()` has run; output: every helper method uses `c.HTTP` to perform requests).

3. **`quickstart.md`**: doubles as the operator-facing migration guide required by FR-021. Sections: (a) install via dev override; (b) install via tagged GitHub Release; (c) `required_providers` alias and a complete first example for each auth mode (API key, OAuth, Federated Identity); (d) migration from the upstream-derived prototype (provider source/local-name swap, resource-type rename, exact `terraform state mv` commands); (e) verifying the GPG signature on a downloaded release.

4. **`backlog.md`**: records the five remediation findings carried over from feature 001 per FR-010. One row per finding with summary, link to the relevant section of `specs/001-tailscale-user-management/`, and `Status: Deferred to v0.2+`.

5. **Agent context update**: run `.specify/scripts/bash/update-agent-context.sh cursor-agent` to refresh the agent-specific context file with the locked-in technical context (Go 1.25.x, Plugin SDK v2, v2 Tailscale client, new module path).

**Output**: `data-model.md`, `contracts/api-operations.md` (cite-only), `contracts/auth-transport.md`, `quickstart.md`, `backlog.md`, updated agent context file.

## Complexity Tracking

*No constitution violations. Table not used.*
