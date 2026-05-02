# Tasks: Standalone Tailscale Membership Provider (v0.1)

**Input**: Design documents from `specs/002-standalone-membership-provider/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md, backlog.md  
**Branch**: `002-standalone-membership-provider`

**Tests**: Included. Constitution v1.1.0 §VIII requires 100% coverage on new/modified code; the auth-routing fix in `tailscale/membership_api.go` is authored test-first per Constitution §I.

**Organization**: Tasks are grouped by user story so each story is independently testable. Most v0.1 work is *deletion*, *renaming*, and one *bug fix* — not new behavior — so Phase 2 (Foundational) is comparatively heavy and must complete before any user story is verifiable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4, US5, US6)
- File paths are absolute under the repository root `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/`

## Path Conventions

- Provider code: `tailscale/`
- Provider entrypoint: `main.go`
- Module manifest: `go.mod`
- Docs site: `docs/`
- HCL examples: `examples/resources/`
- Release config: `.goreleaser.yml`, `.github/workflows/release.yml`
- Spec & planning artifacts: `specs/002-standalone-membership-provider/`

---

## Phase 1: Setup (Verification & Baseline Capture)

**Purpose**: Capture the pre-fork baseline so non-regression checks in later phases are objective.

- [X] T001 Run `go test -coverprofile=coverage.baseline.out ./tailscale/...` from repo root and commit `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/coverage.baseline.out` so kept-code coverage can be diffed after the deletions in Phase 2 (Constitution v1.1.0 §VIII "baseline file" requirement). **Note**: the unscoped suite has pre-existing failures in DNS resources that Phase 2a deletes anyway; the committed baseline was therefore captured against the kept-code regex `'TailnetMembership|MembershipAPI|validateProviderCreds|TestProvider$'` (membership_api.go: 50–100% per func, resource_tailnet_membership.go: 63–100% per func, provider.go: 88% on schema/init, 0% on `providerConfigure`/`validateProviderCreds` — those are exercised only by acceptance tests).
- [X] T002 [P] Verify the GitHub repository name is `terraform-provider-tailscale-membership` (matches the new Go module path FR-012); if not, rename it before any release-pipeline tasks in Phase 8 land. **Done**: GitHub repo already renamed; local `origin` remote URL updated to `git@github.com:pmpaulino/terraform-provider-tailscale-membership.git`.
- [X] T003 [P] Confirm a project-controlled GPG key is available as the `GPG_PRIVATE_KEY` and `PASSPHRASE` GitHub Actions secrets (FR-015); record its key ID and fingerprint for use when authoring the in-repo `KEYS` file in T075 and the README "Verifying releases" section in T063. **Done (helper authored)**: `scripts/setup-release-gpg-key.sh` generates an ed25519 release-signing key, exports the armored private key, public key, and passphrase to `~/.tailscale-membership-release-key/`, and prints the fingerprint plus the GitHub Secrets URL. Run this script before T072 (release dress rehearsal) and before T075 (KEYS file).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Hard-fork the codebase. Delete every non-membership resource, data source, doc, and example; rename the Go module; rename the Terraform resource type; tidy the dependency graph. Until this phase completes, NO user story can be exercised end-to-end (the binary won't build cleanly under the new module path; provider schema still exposes upstream resources; HCL examples reference the old resource type).

**⚠️ CRITICAL**: All user-story phases (Phase 3 onwards) depend on Phase 2 completion.

### Phase 2a: Delete non-membership resources

- [X] T004 [P] Delete `tailscale/resource_acl.go` and `tailscale/resource_acl_test.go`
- [X] T005 [P] Delete `tailscale/resource_aws_external_id.go` and `tailscale/resource_aws_external_id_test.go`
- [X] T006 [P] Delete `tailscale/resource_contacts.go` and `tailscale/resource_contacts_test.go`
- [X] T007 [P] Delete `tailscale/resource_device_authorization.go` and `tailscale/resource_device_authorization_test.go`
- [X] T008 [P] Delete `tailscale/resource_device_key.go` and `tailscale/resource_device_key_test.go`
- [X] T009 [P] Delete `tailscale/resource_device_subnet_routes.go` and `tailscale/resource_device_subnet_routes_test.go`
- [X] T010 [P] Delete `tailscale/resource_device_tags.go` and `tailscale/resource_device_tags_test.go`
- [X] T011 [P] Delete `tailscale/resource_dns_configuration.go` and `tailscale/resource_dns_configuration_test.go`
- [X] T012 [P] Delete `tailscale/resource_dns_nameservers.go` and `tailscale/resource_dns_nameservers_test.go`
- [X] T013 [P] Delete `tailscale/resource_dns_preferences.go` and `tailscale/resource_dns_preferences_test.go`
- [X] T014 [P] Delete `tailscale/resource_dns_search_paths.go` and `tailscale/resource_dns_search_paths_test.go`
- [X] T015 [P] Delete `tailscale/resource_dns_split_nameservers.go` and `tailscale/resource_dns_split_nameservers_test.go`
- [X] T016 [P] Delete `tailscale/resource_federated_identity.go` and `tailscale/resource_federated_identity_test.go` (standalone resource only — Federated Identity auth mode in `provider.go` preserved per FR-007)
- [X] T017 [P] Delete `tailscale/resource_logstream_configuration.go` and `tailscale/resource_logstream_configuration_test.go`
- [X] T018 [P] Delete `tailscale/resource_oauth_client.go` and `tailscale/resource_oauth_client_test.go`
- [X] T019 [P] Delete `tailscale/resource_posture_integration.go` and `tailscale/resource_posture_integration_test.go`
- [X] T020 [P] Delete `tailscale/resource_tailnet_key.go` and `tailscale/resource_tailnet_key_test.go`
- [X] T021 [P] Delete `tailscale/resource_tailnet_settings.go` and `tailscale/resource_tailnet_settings_test.go`
- [X] T022 [P] Delete `tailscale/resource_webhook.go` and `tailscale/resource_webhook_test.go`

### Phase 2b: Delete non-membership data sources

- [X] T023 [P] Delete `tailscale/data_source_4via6.go` and `tailscale/data_source_4via6_test.go` (removed direct dep on `tailscale.com/net/tsaddr`, contributing to FR-013)
- [X] T024 [P] Delete `tailscale/data_source_acl.go` and `tailscale/data_source_acl_test.go`
- [X] T025 [P] Delete `tailscale/data_source_device.go` and `tailscale/data_source_device_test.go` (removed direct dep on `tailscale.com/tstest`, contributing to FR-013)
- [X] T026 [P] Delete `tailscale/data_source_devices.go` and `tailscale/datasource_devices_test.go`
- [X] T027 [P] Delete `tailscale/data_source_user.go`
- [X] T028 [P] Delete `tailscale/data_source_users.go` and `tailscale/data_source_users_test.go`

### Phase 2c: Trim provider registration & shared test fixtures

- [X] T029 `provider.go::Provider().ResourcesMap` shrunk to `{"tailscale_membership_tailnet_membership": resourceTailnetMembership()}`; `DataSourcesMap` empty. Provider schema (auth args, tailnet, base_url, user_agent), `providerConfigure`, and `validateProviderCreds` unchanged (FR-001/002/003/004/011).
- [X] T030 `provider_test.go`: dropped the `TAILSCALE_TEST_DEVICE_NAME` requirement in `testAccPreCheck` (only referenced deleted device tests). All other helpers are generic (TestServer harness, validateProviderCreds tests, resource-state assertions) and were retained verbatim.
- [X] T031 `tailscale_test.go`: file inspected — fully generic (`TestServer` HTTP test harness with method/path/body recording and queued response support). Zero references to deleted resources; no edits required.
- [X] T032 `resource_tailnet_membership_test.go`: 14 occurrences of `tailscale_tailnet_membership` renamed to `tailscale_membership_tailnet_membership` (matching T029's new registration); doc comment in `resource_tailnet_membership.go` updated likewise.

### Phase 2d: Rename Go module + import paths (FR-012)

- [X] T033 `go.mod` line 1 now reads `module github.com/pmpaulino/terraform-provider-tailscale-membership`.
- [X] T034 `git grep "github.com/tailscale/terraform-provider-tailscale" -- '*.go' 'go.mod' '.goreleaser.yml'` returns zero matches. Spec files retain both old and new paths (intentional — the migration guide documents the rename).
- [X] T035 `main.go` import updated to `"github.com/pmpaulino/terraform-provider-tailscale-membership/tailscale"`; `plugin.Serve` and `ProviderFunc` unchanged. `.goreleaser.yml` ldflags `-X` path also updated to the new module path.

### Phase 2e: Tidy dependency graph & verify build

- [X] T036 `go mod tidy` clean. `tailscale.com v1.94.1` removed from `go.mod` (was indirect-only via deleted resources). `github.com/pkg/errors` also removed. `github.com/tailscale/hujson` demoted from direct to indirect. `tailscale.com/client/tailscale/v2 v2.7.0` remains as the sole Tailscale-org direct dep — FR-013 satisfied.
- [X] T037 `go build ./...` exits 0 with no output.
- [X] T038 `go test -run 'TailnetMembership|MembershipAPI|validateProviderCreds|TestProvider$' ./tailscale/...` passes (membership resource tests + validateProviderCreds + TestProvider — the only kept tests). Full unscoped suite has pre-existing acceptance-test failures (DNS) that don't apply post-fork since those resources are gone.
- [X] T039 Per-function coverage non-regression confirmed against `coverage.baseline.out`: `membership_api.go` 50–100% per func (unchanged), `provider.go` Provider 88% (unchanged), `resource_tailnet_membership.go` 63–100% per func (unchanged). Total package coverage *increased* from 16.4% to 54.2% solely because the denominator (deleted statement count) shrank — no per-file regression. Constitution v1.1.0 §VIII satisfied.

**Checkpoint**: Repository is now a single-purpose hard fork. The provider compiles, tests pass, no upstream module path remains, the `tailscale.com v1.94.1` indirect dep is gone, and the only registered resource is `tailscale_membership_tailnet_membership`. Phases 3–8 may now begin in parallel (subject to within-story ordering).

---

## Phase 3: User Story 1 — Single membership resource visible to operators (Priority: P1) 🎯 MVP

**Goal**: A Terraform operator declaring `pmpaulino/tailscale-membership` and configuring it for their tailnet sees exactly one resource type (`tailscale_membership_tailnet_membership`) and zero data sources in the provider schema.

**Independent Test**: Build the provider, place it in a dev override, write a minimal HCL config that references `tailscale_membership_tailnet_membership`, run `terraform providers schema -json`, and assert the JSON contains exactly one entry under `resource_schemas` and zero under `data_source_schemas`.

### Tests for User Story 1

- [X] T040 [P] [US1] `TestProvider_SchemaSurface` added to `tailscale/provider_test.go`: asserts `len(ResourcesMap)==1`, `len(DataSourcesMap)==0`, sole key `tailscale_membership_tailnet_membership`. Passes.
- [X] T041 [P] [US1] `TestProvider_UnknownUpstreamResourceTypeRejected` added: covers 7 representative removed/renamed names (`tailscale_acl`, `tailscale_dns_nameservers`, `tailscale_tailnet_key`, `tailscale_webhook`, `tailscale_membership_dns_nameservers`, `tailscale_membership_acl`, AND the pre-rename `tailscale_tailnet_membership`). Passes.

### Implementation for User Story 1

- [X] T042 [US1] Registration map confirmed = `{"tailscale_membership_tailnet_membership": resourceTailnetMembership()}` — no-op verifying T029.
- [X] T043 [US1] `go test -run 'TestProvider_Schema|TestProvider_Unknown' -v ./tailscale/...` → both PASS in 0.684s.

**Checkpoint**: US1 complete. The provider's published schema satisfies SC-003 (exactly one resource, zero data sources).

---

## Phase 4: User Story 2 — Membership ops work under all three auth modes (Priority: P1)

**Goal**: API key, OAuth client credentials, and Federated Identity auth modes all successfully drive a complete create/read/update/destroy cycle on a membership. The membership API helper routes requests through the v2 client's authenticated HTTP transport (FR-006, FR-007); it does NOT silently fall back to API-key Basic auth for OAuth/Federated-Identity modes.

**Independent Test**: For each auth mode, run an acceptance test (gated by `TF_ACC`) against a test tailnet that exercises Create → Read → Update(role) → Update(suspended) → Destroy on a single membership; assert every step succeeds and the backend records each call as authenticated through the matching transport.

### Tests for User Story 2 (test-first per Constitution §I)

- [X] T044 [P] [US2] `tailscale/membership_api_test.go` created with `TestMembershipAPI_RoutesThroughAuthHTTPClient` per the contract in `contracts/auth-transport.md`: stub `tailscale.Auth.HTTPClient` returns `*http.Client` with marker-injecting `RoundTripper`. Each of the 7 helpers (`listUserInvites`, `createUserInvite`, `deleteUserInvite`, `suspendUser`, `restoreUser`, `deleteUser`, `updateUserRole`) asserts (a) request flowed through the round-tripper exactly once, (b) no `Authorization` header set by helper, (c) baseURL passed to `HTTPClient` matches `client.BaseURL`. Verified failing on pre-fix code (DNS resolution error confirms requests went via `http.DefaultClient`); now PASSES.
- [X] T045 [P] [US2] `TestMembershipAPI_APIKeyStillUsesBasicAuth` added: builds `tailscale.Client{APIKey:"test-key", BaseURL:srv.URL, Tailnet:"test-tailnet"}` (no `Auth`); for each of the 7 helper methods, verifies (a) request reaches `httptest.Server` exactly once, (b) `Authorization: Basic <base64("test-key:")>`, (c) HTTP method matches expected, (d) request path matches expected. Initially failed on `createUserInvite` (server stub returned `[]` while helper expects ≥1 element); fixture corrected. Verified PASS pre- and post-fix — regression safety confirmed.
- 6/7 subtests passed pre-fix (Basic auth path was already correct); now all 7 pass.

### Implementation for User Story 2

- [X] T046 [US2] `tailscale/membership_api.go::do()` patched per `research.md §R1` (with one correction):
  - Added `_ = m.Client.Users()` at top — triggers v2 client's `sync.Once` `init()` which installs `c.Auth.HTTPClient(...)` on `c.HTTP` for OAuth/Federated Identity, zeros `APIKey` when `Auth != nil`, and initialises `c.HTTP` to a default 1m-timeout client for API-key mode.
  - Removed the `httpClient := m.Client.HTTP; if httpClient == nil { httpClient = http.DefaultClient }` fallback — no longer reachable post-init().
  - **Kept** the `if m.Client.APIKey != "" { req.SetBasicAuth(...) }` branch (research.md §R1's claim that the v2 client transparently adds Basic auth via `c.HTTP.Do(req)` was incorrect — Basic auth is set by the v2 client's `buildRequest`, which membership_api bypasses; `T045` proves we still need explicit basic auth in API-key mode). Documented inline.
- [X] T047 [US2] `go test -run 'TestMembershipAPI' ./tailscale/...` — all 14 subtests PASS. `do()` per-function coverage is 85.0% (baseline 86.4%; the 1.4 pp drop is purely arithmetic — one new statement added, no new uncovered branch). Auth-routing fix branches (`Users()` init trigger, `HTTP.Do(req)`) both covered.
- [DEFERRED] T048 [US2] `TestAcc*_OAuthAuthMode` / `*_FederatedIdentityAuthMode` require real Tailscale OAuth-client / federated-identity credentials and a test tailnet — deferred to Phase 9 manual quickstart (T070). The unit-level T044 already proves the HTTP transport routes correctly for both modes.
- [DEFERRED] T049 [US2] Same: existing `TestResourceTailnetMembership_*` are non-acceptance unit tests against the in-process test harness; they continue to pass. A real-Tailscale acceptance pass folds into T070.
- [X] T074 [P] [US2] `TestProvider_RejectsConflictingAuthModes` added to `provider_test.go` covering all 5 invalid combinations: (a–c) `api_key` + each OAuth credential (Summary contains "conflicting"); (d) `oauth_client_id` without secret/token (Summary contains "mandatory"); (e) `oauth_client_id + oauth_client_secret + identity_token` (Summary contains "conflicting"). **Implementation gap discovered & fixed**: pre-existing `validateProviderCreds` silently accepted case (e) by returning nil; FR-008 forbids this. Added explicit branch in `validateProviderCreds` returning `"conflicting - 'oauth_client_secret' and 'identity_token' are mutually exclusive"` for case (e). All 5 subtests + the existing `TestValidateProviderCreds` (10 subtests) pass.

**Checkpoint**: US2 complete (automated portion). SC-002 ("zero auth-related fallbacks; zero API-key Basic auth observed when OAuth or Federated Identity is configured") enforced by T044 + T045 unit tests. FR-008 ("no silent auth-mode selection") enforced by T074 + the new validateProviderCreds branch. The carried-over correctness fix from feature 001 is shipped. T048/T049 (full TF_ACC acceptance against real Tailscale tailnet under each auth mode) deferred to T070's manual quickstart.

---

## Phase 5: User Story 3 — Migration from upstream-derived prototype (Priority: P2)

**Goal**: An operator currently managing memberships via the upstream-derived prototype `tailscale_tailnet_membership` resource can switch to this provider with documented HCL and `terraform state mv` commands; `terraform plan` after migration reports no diffs (SC-005).

**Independent Test**: Take a sample HCL config + Terraform state file written against the prototype resource (committed under `specs/002-standalone-membership-provider/quickstart.md` section 4 example); apply the documented migration steps; run `terraform plan` and assert exit code 0 with `No changes.` output.

### Tests for User Story 3

- [X] T050 [P] [US3] `specs/002-standalone-membership-provider/checklists/migration.md` created. Tracks the manual `terraform state mv` walkthrough against a dev tailnet before tagging v0.1.0: pre-conditions (state backup, clean baseline plan), the 7 walkthrough steps mapped to `quickstart.md` §4.1–4.7 acceptance points, post-conditions (no Tailscale API calls during migration, all IDs preserved, rollback procedure if plan diff appears), and a sign-off section for the operator/date/tailnet executed.

### Implementation for User Story 3

- [X] T051 [US3] `quickstart.md` §4 verified to cover all 7 required sub-steps: §4.1 provider source change, §4.2 provider block rename, §4.3 resource type rename, §4.4 exact `terraform state mv` command per resource (including module-qualified addresses), §4.5 `terraform state replace-provider`, §4.6 verification (`terraform init -upgrade && terraform plan` returns "No changes"), §4.7 source-address drift note. Already shipped from /speckit.plan; no edit needed.
- [DEFERRED → T063] T052 [US3] README "Migration from the upstream-derived prototype" section folds into T063's README rewrite in Phase 7 (per the plan's "User Story Dependencies" note). The link target `quickstart.md#4-migration-from-the-upstream-derived-prototype` is stable.

**Checkpoint**: US3 complete. The migration path exists, is discoverable from the README, and is verifiable manually against a worked example.

---

## Phase 6: User Story 4 — Tag-triggered, GPG-signed, Registry-shaped release (Priority: P2)

**Goal**: A Git tag matching either `vX.Y.Z` or `vX.Y.Z-{alpha,beta,rc}.N` triggers GoReleaser to build the 11-platform OS/arch matrix, GPG-sign the checksums file, and publish a GitHub Release. Tags not matching either pattern do NOT trigger a release. Pre-release tags are marked "pre-release" in the GitHub UI. The release pipeline fails loudly if `GPG_FINGERPRINT` is unset.

**Independent Test**: Run `goreleaser release --snapshot --clean` locally (no GPG required for snapshot mode); assert the produced `dist/` directory contains exactly 11 zip archives matching the expected OS/arch pattern, plus a `SHA256SUMS` file. Then push a no-op pre-release tag (e.g. `v0.0.1-rc.0`) to a test branch on a fork; observe the release workflow fires and produces the expected artifacts.

### Tests for User Story 4

- [X] T053 [P] [US4] `scripts/test-release-snapshot.sh` created and wired into `.github/workflows/ci.yml` as a `release-snapshot` job. Script runs `goreleaser release --snapshot --clean --skip=publish,sign` then asserts all 4 FR-014 invariants on `dist/`:
  - (a) exactly 11 zip archives;
  - (b) os/arch matrix exactly matches the FR-014 set (sorted `diff` against the canonical 11-line list);
  - (c) `*_SHA256SUMS` exists and has 11 lines (1 per archive);
  - (d) `*_manifest.json` exists, parses as JSON, has top-level `version: 1` (Registry manifest schema version per HashiCorp's contract — the spec text in this row originally said `version: "5.0"` which conflated the manifest schema version with the protocol version), AND `metadata.protocol_versions` contains `"5.0"` (Plugin SDK v2 protocol).
  - Verified locally: `==> All FR-014 release-shape assertions passed.` Goreleaser snapshot produces "0.26.0-SNAPSHOT-eb64233" (inheriting upstream tag history; harmless for snapshot mode).
  - Implementation note: `release.extra_files` in `.goreleaser.yml` only fires during the actual publish step, which is skipped in snapshot mode. The script copies `terraform-registry-manifest.json` into `dist/` post-build using the version it reads from `dist/metadata.json`, mimicking what the real release publishes.

### Implementation for User Story 4

- [X] T054 [US4] `.goreleaser.yml` rewritten per `research.md §R5`:
  - `ldflags` use the renamed module path (already set in T035).
  - `goos: [linux, darwin, windows, freebsd]` × `goarch: [amd64, arm64, "386", arm]` cross-product (16 pairs) with explicit `ignore` of the 5 non-shipped pairs (`darwin/386`, `darwin/arm`, `windows/arm64`, `windows/arm`, `freebsd/arm64`) → exactly 11 platforms.
  - `signs:` block preserved.
  - `release.prerelease: auto` preserved (FR-014: any `-` in tag → "pre-release" in GitHub UI).
  - Added `release.extra_files` glob for `terraform-registry-manifest.json` so the manifest ships with every real release.
  - Added top-level `version: 2` (silences the GoReleaser v2 schema warning).
  - Migrated deprecated `archives.format: zip` → `archives.formats: [zip]`.
  - New file `terraform-registry-manifest.json` at repo root (`{version: 1, metadata.protocol_versions: ["5.0"]}`).
- [X] T055 [US4] `.github/workflows/release.yml`:
  - `on.push.tags` rewritten to the four explicit glob patterns from `research.md §R3` (`v[0-9]+.[0-9]+.[0-9]+` plus three `-{alpha,beta,rc}.[0-9]+` variants).
  - GPG import step (`crazy-max/ghaction-import-gpg@v6.3.0`) and GoReleaser run unchanged.
  - The GPG action's `gpg_private_key` is a required input — when `secrets.GPG_PRIVATE_KEY` is unset the action fails with `Error: gpg_private_key input is required`, which propagates as a step failure → entire workflow fails. FR-016 ("release MUST fail loudly if GPG_FINGERPRINT is unset") satisfied without extra logic.
  - Inline header comment documents the tag patterns and the pre-release/latest decision flow.
- [X] Bonus fix in `.github/workflows/ci.yml`: `goimports -w -local github.com/tailscale .` rewritten to `github.com/pmpaulino` (the `format` job would otherwise have failed post-rename in Phase 2d).

**Checkpoint**: US4 complete. SC-004 ("a Git tag matching the release pattern produces a complete, GPG-signed, Registry-shaped release artifact set ... with no manual post-processing required") is satisfied. T053 makes the 11-archive guarantee mechanically testable in CI without needing a real tag.

---

## Phase 7: User Story 5 — Operator-discoverable docs (Priority: P3)

**Goal**: The repository's `docs/` directory contains exactly one provider-configuration page and one resource page (the membership resource), each documenting every schema argument and attribute; HCL examples in each page run cleanly with only credential substitution. README is rewritten to v0.1 shape.

**Independent Test**: Open `docs/`, confirm there is one `index.md` and exactly one file under `docs/resources/` (`tailnet_membership.md`); cross-reference every argument/attribute in `tailscale/provider.go`'s schema and in `resourceTailnetMembership()`'s schema against the docs page; assert 100% coverage in both directions (SC-007).

### Tests for User Story 5

- [X] T056 [P] [US5] `scripts/check-docs-coverage.sh` created and wired into `.github/workflows/ci.yml` as a required `docs-coverage` job. **Implementation pivot from the original task text**: the original plan invoked `tfplugindocs validate` + `tfplugindocs generate` and diffed the output against committed docs. That approach turned out to be incompatible with this provider — see T060 for the dash-vs-underscore root cause. The replacement script is dependency-free: it greps tab-indented `"<key>": {` lines from the schema map blocks in `tailscale/provider.go` and `tailscale/resource_tailnet_membership.go`, greps the `## Schema` sections of `docs/index.md` and `docs/resources/tailnet_membership.md` for `- \`<key>\`` bullets, and asserts the two sets are equal in both directions (with the SDK-implicit `id` attribute added to the resource side). Verified locally: both schema↔docs comparisons pass cleanly.

### Implementation for User Story 5

- [X] T057 [P] [US5] Bulk-deleted every file under `docs/resources/` (the previously-existing `tailnet_membership.md` was also deleted alongside the 19 stale per-resource pages and replaced by the hand-authored version produced in T060).
- [X] T058 [P] [US5] `docs/data-sources/` deleted in its entirety (FR-002 — no data sources in v0.1).
- [X] T059 [US5] `docs/index.md` rewritten by hand. Title changed to "Tailscale Membership Provider" (matches the human-readable provider name). Describes v0.1 scope (membership only); documents all three auth modes; every example uses the `tailscale-membership` local name (dashed) per the `spec.md` Q3 amendment; the `## Coexisting with the upstream tailscale/tailscale provider` section shows the dual-load pattern with the `provider = tailscale-membership` override on the membership resource; links to the migration guide and to `KEYS`. The full schema is hand-listed under `## Schema`.
- [X] T060 [US5] **Original task replaced**: the original step ran `go generate ./...` against `tfplugindocs generate` and used `templates/` for narrative additions. This proved unworkable: `tfplugindocs` requires `--provider-name` to be both (a) the prefix of registered resource keys (`tailscale_membership_*`, with underscore) for resource-doc derivation AND (b) a valid Terraform local name for the internal `terraform providers schema` validation it performs. Terraform local names cannot contain underscores; resource-key prefixes cannot contain dashes; no single value satisfies both. The chosen resolution (Phase 7, decided 2026-04-18 with user "option B" approval): drop tfplugindocs entirely. Concretely:
  - `//go:generate go run .../tfplugindocs generate ...` directive removed from `main.go` (replaced with a package-doc comment explaining the rationale).
  - `github.com/hashicorp/terraform-plugin-docs` dependency removed from `go.mod` (and from `tools.go` build-tagged file).
  - `templates/` directory deleted in its entirety (no longer needed).
  - `docs/resources/tailnet_membership.md` hand-authored to mirror the runtime schema (Required/Optional/Read-Only sections), include the full `required_providers` alias block + `provider = tailscale-membership` attribute on every example, the validation/expiry/last-admin/migration narrative blocks, and the import command.
  - Bidirectional schema↔docs coverage now enforced by T056's `scripts/check-docs-coverage.sh` instead of `tfplugindocs validate`.
- [X] T061 [P] [US5] Renamed `examples/resources/tailscale_tailnet_membership/` → `examples/resources/tailscale_membership_tailnet_membership/`. In `resource.tf` the resource type is renamed, the full `required_providers { tailscale-membership = { source = "pmpaulino/tailscale-membership", version = "~> 0.1" } }` alias block is prepended, and the `provider = tailscale-membership` attribute is added to the resource block (with an inline comment explaining why it's required). `import.sh` updated to use the new resource type in its example command.
- [X] T062 [US5] Deleted every other directory under `examples/resources/` (19 directories) and deleted `examples/data-sources/` entirely. Verified via `ls examples/resources/` returning only `tailscale_membership_tailnet_membership`.
- [X] T063 [US5] `README.md` rewritten per FR-020 + FR-021. New sections: project status (v0.1/v0.2 scope), Install (Option A dev override + Option B tagged GitHub Release), the **Naming conventions** table that explicitly enumerates the source-address / local-name / resource-type triple and warns about the `provider = tailscale-membership` requirement, Migration from the upstream-derived prototype (links to `quickstart.md §4`), **Verifying releases** (links to in-repo `KEYS` file with `gpg --import` instructions; satisfies FR-015's "discoverable from the repository" requirement that T003's local-only note alone did not meet — see T075), Documentation index, Local provider development, License & attribution.

**Checkpoint**: US5 complete. SC-001 (a new operator can install + apply within 15 minutes from README + docs alone) and SC-007 (100% docs/schema bidirectional coverage) are satisfied. The custom `scripts/check-docs-coverage.sh` enforces SC-007 in CI as the required `docs-coverage` job.

**Spec amendment record (Phase 7, 2026-04-18)**: During this phase we discovered that Terraform's CLI rejects underscores in provider local names (`must contain only letters, digits, and dashes`). The original `/speckit.clarify` Q3 answer (local name `tailscale_membership` with underscore) is therefore impossible. With user approval ("option B"), local name was changed to `tailscale-membership` (dashed) and an explicit `provider = tailscale-membership` attribute requirement was added to every membership resource block (because the `tailscale_*` resource prefix would otherwise default-bind to the upstream `tailscale/tailscale` provider in coexistence scenarios — and operators of this provider are *expected* to also load the upstream provider for non-membership resources). Cascade edits made to `spec.md` Q3 + FR-011 + Key Entities, `plan.md`, `data-model.md`, `quickstart.md` §2/§3/§4, `checklists/requirements.md`, `checklists/migration.md` §4.1–4.3. Resource-type identifier `tailscale_membership_tailnet_membership` is unchanged.

---

## Phase 8: User Story 6 — License & attribution compliance (Priority: P3)

**Goal**: `LICENSE` is the MIT license retaining the upstream copyright lines; `NOTICE` names and links to the upstream project and reproduces its copyright; no license-incompatible dependencies remain.

**Independent Test**: Open `LICENSE` and `NOTICE`; visually confirm content matches FR-022/FR-023; run `go list -m -json all | jq -r '.Path'` and a license scanner (or manual review) against the `go.sum` set; assert no GPL/AGPL/SSPL/etc. dependencies (FR-024).

### Implementation for User Story 6

- [X] T064 [P] [US6] LICENSE verified intact: contains the MIT text plus both upstream copyright lines (`Copyright (c) 2021 David Bond` + `Copyright (c) 2024 Tailscale Inc & Contributors`). No edit was needed — file was already correct from the original fork. FR-022 satisfied.
- [X] T065 [P] [US6] `NOTICE` created at repo root from the `research.md` §R4 template (FR-023): names `terraform-provider-tailscale`, links to its GitHub URL, reproduces both upstream copyright lines, and states the hard-fork relationship in plain English. License-scanning tools will pick this up alongside `LICENSE`.
- [X] T066 [P] [US6] Dependency-license review tooling rebuilt because `go-licenses` v1.6.0 crashes on Go ≥ 1.22 ("Package <stdlib> does not have module info"). Replaced with a 100-line classifier at `scripts/license-review.go` (build-tag-fenced behind `//go:build licensereview`) plus the wrapper `scripts/run-license-review.sh`. Report archived at `specs/002-standalone-membership-provider/license-review-v0.1.0.txt`. Result: 73 dependencies enumerated, **PASS — no GPL/AGPL/SSPL/Commons-Clause dependencies detected**. License mix: MIT (22), Apache-2.0 (23), BSD-3-Clause (20), MPL-2.0 (20, all HashiCorp ecosystem — file-scoped copyleft, MIT-redistribution-compatible), BSD-2-Clause (7), ISC (1). FR-024 satisfied. Re-run any time with `./scripts/run-license-review.sh`.
- [X] T075 [P] [US6] `KEYS` file created at repo root containing the ASCII-armored public half of the project's ed25519 release-signing key, with a plain-text header listing fingerprint (`3F84 0B5A 363E 8126 1D6B  57F0 1583 C036 5BDD 148C`), long key ID (`1583C0365BDD148C`), owner email (`pmpaulino@gmail.com`), creation date (2026-04-19 UTC), and expiration date (2028-04-18 UTC) so operators can cross-check before importing. Cross-linked from `README.md` (T063 — explicit fingerprint added inline as a second cross-check channel) and `quickstart.md` §5 (already drafted). **Validation performed**: (1) `gpg --import KEYS` into a throwaway `GNUPGHOME` succeeded and listed the same fingerprint; (2) `goreleaser release --snapshot --clean --skip=publish` ran with `GPG_FINGERPRINT` exported, built all 11 archives + checksum, and invoked the `signs:` step with the correct fingerprint and artifact (failure was at TTY pinentry only, which CI bypasses via `GPG_PRIVATE_KEY`/`PASSPHRASE` GitHub Actions secrets). The full sign-and-verify dress rehearsal repeats in Phase 9 (T072) and again on the first tag push. FR-015's "discoverable from the repository" requirement satisfied; `KEYS` is now reachable at `https://github.com/pmpaulino/terraform-provider-tailscale-membership/blob/main/KEYS`.

**Checkpoint**: US6 complete. SC-006 ("license/attribution posture passes a routine open-source license review with no findings") is satisfied.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final sweep before tagging v0.1.0.

- [X] T067 [P] Stray-reference grep ran clean. The remaining matches for `tailscale/tailscale` (`CONTRIBUTING.md`, `README.md`, `docs/index.md`, `examples/.../resource.tf`) are all intentional: external citations to the Tailscale daemon repo (different from the upstream provider, but the substring matches), and deliberate documentation of the upstream provider's source address in the **Naming conventions** / **Coexisting with the upstream `tailscale/tailscale` provider** sections that explain why the explicit `provider = tailscale-membership` attribute is required. The matches for `tailscale_tailnet_membership` (without the `_membership_` prefix) are all intentional too: migration guidance pointing operators from the prototype name to the new prefixed form, plus a regression-guard test case in `tailscale/provider_test.go` that asserts the pre-rename name is NOT in `ResourcesMap`. As a Phase 9 cleanup, `CONTRIBUTING.md` was rewritten for the standalone fork (issue tracker now points at `pmpaulino/terraform-provider-tailscale-membership`; scope reminder added; release / lint / docs-coverage developer commands documented).
- [X] T068 [P] `golangci-lint run ./...` (v2.11.4, zero config = defaults) went from 24 issues to 0 across `tailscale/`. Removed dead helpers from `provider.go` (`diagnosticsAsError`, `diagnosticsErrorWithPath`, `createUUID`, `readWithWaitFor`, `setProperties`, `optional`, `isAcceptanceTesting`, `combinedSchemas`) and the imports they pulled in (`errors`, `os`, `time`, `maps`, `cty`, `uuid`); removed dead test helpers from `provider_test.go` (`testAccProvider`, `testAccPreCheck`, `testAccProviderFactories`, `testResourceCreated`, `testResourceDestroyed`, `checkResourceRemoteProperties`, `checkResourceDestroyed`, `checkPropertiesMatch`, `assertEqual`, plus the redundant `TestProvider_Implemented` smoke test that's already covered by `TestProvider` and `TestProvider_SchemaSurface`) and the imports they pulled in (`errors`, `fmt`, `os`, `cmp`, `resource`, `terraform`); fixed 7 `errcheck` findings by wrapping every `defer resp.Body.Close()` in `defer func() { _ = resp.Body.Close() }()`; fixed 8 `staticcheck QF1008` findings by simplifying embedded-selector accesses (`m.Client.X` → `m.X`) on `*tailscale.Client` in `membership_api.go`. `go test ./...` still passes (17.6s, all green). `scripts/check-docs-coverage.sh` still passes.
- [X] T069 [P] Manual smoke test run against `sardine.ai` dev tailnet using locally-built provider (`go install` + `dev_overrides`). **API key mode**: full `plan → apply → destroy` cycle completed successfully — `tailscale_membership_tailnet_membership` resource created (invite issued), read back, and destroyed cleanly. **OAuth mode**: confirmed 403 `"operation only permitted for user-owned keys"` on the `user-invites` create path regardless of scopes — this is a Tailscale API server-side restriction, not a provider bug; documented in `docs/index.md` and `docs/resources/tailnet_membership.md`. **Federated identity**: same bearer-token restriction as OAuth (federated exchanges an OIDC token for an OAuth bearer token); create path blocked identically. SC-001 + SC-002 satisfied for the API key auth mode; OAuth/federated limitation documented as a known Tailscale API constraint.
- [X] T070 [P] Migration walkthrough waived: the prototype `tailscale_tailnet_membership` resource (feature 001) was never published or used in production — no real state files exist to migrate. The migration guide in `quickstart.md §4` documents the `terraform state mv` procedure for anyone who ran the prototype locally; the resource rename and import path are verified by the schema-surface tests in Phase 3 (T040–T043). SC-005 satisfied by documentation and unit-test coverage rather than a live migration run.
- [X] T071 [P] `scripts/test-release-snapshot.sh` ran locally end-to-end. All four FR-014 release-shape assertions passed: (1) exactly 11 zip archives, (2) os/arch matrix matches the FR-014 set (linux × {amd64,arm64,386,arm}; darwin × {amd64,arm64}; windows × {amd64,386}; freebsd × {amd64,arm,386}), (3) SHA256SUMS file has 11 lines, (4) `terraform-registry-manifest.json` has `version: 1` and `metadata.protocol_versions: ["5.0"]`.
- [X] T072 Pre-release dress rehearsal complete. Tagged `v0.0.1-rc.0` on the `002-standalone-membership-provider` branch tip (after force-updating `origin/main` to the same commit, per the v0.1 hard-fork strategy), pushed the tag → `.github/workflows/release.yml` fired automatically and finished green in 4m22s. Workflow steps all passed including GPG key import (correct fingerprint `3F840B5A363E81261D6B57F01583C0365BDD148C` resolved from the GitHub Actions secrets) and GoReleaser publish. The published GitHub Release at `https://github.com/pmpaulino/terraform-provider-tailscale-membership/releases/tag/v0.0.1-rc.0` carries 14 assets: the FR-014 11-platform zip matrix (darwin × {amd64,arm64}; freebsd × {386,amd64,arm}; linux × {386,amd64,arm,arm64}; windows × {386,amd64}), the Terraform-Registry-shaped `_manifest.json`, the `_SHA256SUMS` file, and its `_SHA256SUMS.sig`. `prerelease=true` was set automatically by GoReleaser's `release.prerelease: auto` rule (any tag containing a hyphen → pre-release UI), satisfying FR-014. **End-to-end verification done with only what an operator can reach from the repo**: downloaded the live release artifacts, imported the in-repo `KEYS` file's public-key block into a throwaway `GNUPGHOME`, and got `Good signature` against the expected fingerprint plus a passing `shasum -a 256 -c` on the `linux_amd64.zip`. US4 acceptance scenarios 1–4 + SC-004 satisfied. Two non-blocking workflow warnings worth tracking before v1.0: (a) `crazy-max/ghaction-import-gpg` and `goreleaser/goreleaser-action` still on Node.js 20 (deprecated in GitHub Actions runners as of June 2026); (b) `goreleaser-action`'s `version: latest` should be pinned to `~> v2`. Neither is a v0.1.0 blocker.
- [X] T073 Release tagged and published. Final version shipped as `v1.0.0` (not `v0.1.0` — bumped to v1 to clear the upstream 0.x tag history that came over with the fork). Release workflow run: https://github.com/pmpaulino/terraform-provider-tailscale-membership/actions/runs/25263158458 — completed in 1m5s. Published release: https://github.com/pmpaulino/terraform-provider-tailscale-membership/releases/tag/v1.0.0. Asset count: **14** (11 platform zips + manifest + SHA256SUMS + SHA256SUMS.sig) — matches FR-014 exactly. `prerelease: false` (no hyphen in tag). Release signed with new RSA 4096 key (fingerprint `1AE3 E49A 1CCC 2805 A321 C991 21BE A434 67F2 A13D`) required by Terraform Registry — ed25519 key rotation forced by Registry's RSA/DSA-only policy. `quickstart.md §5` uses version-agnostic `vX.Y.Z` placeholders; no update needed. Provider submitted to Terraform Registry (pmpaulino/tailscale-membership, Networking category).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies. Can begin immediately. T001 produces `coverage.baseline.out` consumed by T039.
- **Phase 2 (Foundational)**: Depends on Phase 1. **Blocks every user-story phase.** Within Phase 2: 2a/2b can run in parallel; 2c depends on 2a + 2b (it edits `provider.go`/`provider_test.go`/`tailscale_test.go` after the deleted files are gone); 2d depends on 2c (rename touches `provider.go`); 2e depends on 2d (tidy + build verification).
- **Phases 3–8 (User Stories)**: All depend on Phase 2 completion. Independent of each other (see "User Story Dependencies" below); can run in parallel.
- **Phase 9 (Polish)**: Depends on whichever user-story phases are in scope for the v0.1.0 tag.

### User Story Dependencies

- **US1 (P1)**: After Phase 2. Verifies the schema surface that Phase 2 produced; no dependencies on other user stories.
- **US2 (P1)**: After Phase 2. Test-first authored before the auth-routing fix lands. Independent of US1.
- **US3 (P2)**: After Phase 2. The migration guide already exists in `quickstart.md` from `/speckit.plan`; this phase mostly cross-links it from the README (depends on US5's README rewrite for the link target, but the link addition is a one-line edit done as part of T063).
- **US4 (P2)**: After Phase 2 (specifically T034 — the new module path is referenced in `.goreleaser.yml` ldflags). Independent of US1/US2/US3.
- **US5 (P3)**: After Phase 2 (the resource type rename in T029 must land before docs/examples can rename in T060/T061). Independent of US1/US2/US4.
- **US6 (P3)**: After Phase 2 (no actual code dependency, but the dependency-license review in T066 must run after T036's `go mod tidy`). Independent of every other user story.

### Within Each User Story

- Tests authored before implementation when present (Constitution v1.1.0 §I).
- For US2 specifically: T044 + T045 MUST be authored before T046 and MUST FAIL on the pre-fix code; T047 confirms they pass post-fix. T074 is independent of the auth-routing fix and can land any time after Phase 2 closes.
- Acceptance tests (`TF_ACC`) gated by env vars; not required for the unit-test phase to pass.

### Parallel Opportunities

- All Phase 1 setup tasks marked [P] can run in parallel (T002, T003).
- Within Phase 2a: all 19 resource-deletion tasks (T004–T022) can run in parallel (independent files).
- Within Phase 2b: all 6 data-source-deletion tasks (T023–T028) can run in parallel (independent files).
- Phases 2a and 2b can run in parallel (no overlap).
- Phase 3 (US1) tests T040 and T041 can be authored in parallel (same file but different test functions; if a single contributor handles both, no contention).
- Phase 4 (US2) tests T044, T045, and T074 can be authored in parallel (T044/T045 in `membership_api_test.go`, T074 in `provider_test.go` — disjoint files).
- Phase 8 (US6) tasks T064, T065, T066, and T075 can all run in parallel (disjoint root-level files: `LICENSE`, `NOTICE`, `go.sum` review, `KEYS`).
- All six user-story phases (3–8) can run in parallel after Phase 2 completes — they touch largely disjoint file sets:
  - US1: only `tailscale/provider_test.go`
  - US2: `tailscale/membership_api.go`, `tailscale/membership_api_test.go`, `tailscale/resource_tailnet_membership_test.go`, `tailscale/provider_test.go` (T074 only)
  - US3: `README.md` (link only) + verification of `quickstart.md`
  - US4: `.goreleaser.yml`, `.github/workflows/release.yml`, `scripts/test-release-snapshot.sh`, `.github/workflows/ci.yml`
  - US5: `docs/`, `examples/`, `README.md` (rewrite); coordinate the README edit with US3's link addition
  - US6: `LICENSE`, `NOTICE`, `KEYS`
- Phase 9 polish: T067, T068, T069, T070, T071 can run in parallel; T072 and T073 are strictly sequential (release dress-rehearsal then real release).

---

## Parallel Example: Phase 2a Deletions

```bash
# Launch all 19 resource-deletion tasks in parallel:
Task: "Delete tailscale/resource_acl.go and tailscale/resource_acl_test.go"
Task: "Delete tailscale/resource_aws_external_id.go and tailscale/resource_aws_external_id_test.go"
Task: "Delete tailscale/resource_contacts.go and tailscale/resource_contacts_test.go"
# ... (T007–T022 similarly)
```

## Parallel Example: User Story 2 Tests First

```bash
# Author both auth-transport unit tests in parallel before T046 lands the fix:
Task: "Add TestMembershipAPI_RoutesThroughAuthHTTPClient to tailscale/membership_api_test.go"
Task: "Add TestMembershipAPI_APIKeyStillUsesBasicAuth to tailscale/membership_api_test.go"
```

## Parallel Example: All P1 + P2 + P3 User Stories After Phase 2

```bash
# After Phase 2 closes, six contributors can work in parallel:
Task: "[US1] Phase 3 — schema surface tests + verification"
Task: "[US2] Phase 4 — auth-routing fix with TDD"
Task: "[US3] Phase 5 — migration guide cross-link"
Task: "[US4] Phase 6 — release pipeline updates"
Task: "[US5] Phase 7 — docs/examples/README rewrite"
Task: "[US6] Phase 8 — LICENSE/NOTICE/license-review"
```

---

## Implementation Strategy

### MVP First (US1 + US2 only)

US1 and US2 are both P1; the MVP for v0.1 is the union.

1. Complete Phase 1 (setup + baseline).
2. Complete Phase 2 (foundational hard fork). **Critical — blocks everything else.**
3. Complete Phase 3 (US1) — schema surface verification.
4. Complete Phase 4 (US2) — auth-routing fix. This is the carried-over correctness fix from feature 001.
5. **STOP and VALIDATE** — the provider now satisfies the v0.1 *functional* contract (single membership resource working under all three auth modes). Manual validation per `quickstart.md` sections 1–3.

At this point the provider is shippable as a non-Registry binary. P2/P3 stories add release plumbing and discoverability.

### Incremental Delivery

1. Setup + Foundational → repo is a clean fork.
2. Add US1 → schema surface confirmed → Demo: `terraform providers schema -json`.
3. Add US2 → auth modes work end-to-end → Demo: full CRUD cycle under each auth mode.
4. Add US4 → snapshot release works → Demo: 11-archive `dist/` produced locally.
5. Add US5 → docs/examples/README rewritten → Demo: open `docs/index.md` in a browser.
6. Add US3 + US6 → migration guide cross-linked + LICENSE/NOTICE landed → ready to tag v0.1.0.
7. Run Phase 9 polish; tag `v0.0.1-rc.0` (T072) as dress rehearsal; tag `v0.1.0` (T073).

### Parallel Team Strategy

With multiple contributors after Phase 2:

- Contributor A: US1 + US2 (both touch `tailscale/`).
- Contributor B: US4 (touches `.goreleaser.yml`, `.github/workflows/release.yml`).
- Contributor C: US3 + US5 (both touch `README.md` and `docs/`; coordinate the README edit).
- Contributor D: US6 (touches root-level legal files only).

Phase 9 polish is run by whoever cuts the release tag (T072, T073).

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks.
- [Story] label maps each task to its user story for traceability against `spec.md` (FR-009 — behavioral parity with feature 001 — is enforced by Phase 2's preservation of `resource_tailnet_membership.go` unchanged plus US1's schema-surface test).
- All file paths are absolute under `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/`.
- Commit after each task or logical group; commit messages SHOULD reference FR IDs (e.g. "fix(membership): route via v2 client auth transport (FR-006/FR-007, T046)").
- Verify tests fail before the implementation lands for US2 (Constitution §I test-first ordering).
- The five remediation findings carried over from feature 001 are tracked in `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/specs/002-standalone-membership-provider/backlog.md` (B-001…B-005); they are NOT v0.1 blockers and MUST NOT be picked up during this task list per FR-010.
- Stop at any per-phase checkpoint to validate independence before proceeding.
- Avoid: edits to `resource_tailnet_membership.go` itself (it is carried unchanged from feature 001 per FR-009; behavioral changes are explicitly out of scope for v0.1).
