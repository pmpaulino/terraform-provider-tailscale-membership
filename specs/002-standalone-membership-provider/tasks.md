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

- [ ] T040 [P] [US1] Add `TestProvider_SchemaSurface` to `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/provider_test.go`: instantiate `Provider()`, assert `len(p.ResourcesMap) == 1`, `len(p.DataSourcesMap) == 0`, and that the sole resource key is `tailscale_membership_tailnet_membership` (FR-001, FR-002, FR-003, US1 acceptance scenario 2).
- [ ] T041 [P] [US1] Add `TestProvider_UnknownUpstreamResourceTypeRejected` to `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/provider_test.go`: assert that `p.ResourcesMap["tailscale_membership_dns_nameservers"]` is nil (US1 acceptance scenario 3 — operators referencing removed upstream resources get a clean "unknown resource type" error from Terraform's schema check).

### Implementation for User Story 1

- [ ] T042 [US1] Verify the registration map in `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/provider.go` is exactly `{"tailscale_membership_tailnet_membership": resourceTailnetMembership()}`; this should be a no-op confirming T029 is correct.
- [ ] T043 [US1] Run `go test ./tailscale/... -run "TestProvider_Schema|TestProvider_Unknown"` and confirm both new tests pass.

**Checkpoint**: US1 complete. The provider's published schema satisfies SC-003 (exactly one resource, zero data sources).

---

## Phase 4: User Story 2 — Membership ops work under all three auth modes (Priority: P1)

**Goal**: API key, OAuth client credentials, and Federated Identity auth modes all successfully drive a complete create/read/update/destroy cycle on a membership. The membership API helper routes requests through the v2 client's authenticated HTTP transport (FR-006, FR-007); it does NOT silently fall back to API-key Basic auth for OAuth/Federated-Identity modes.

**Independent Test**: For each auth mode, run an acceptance test (gated by `TF_ACC`) against a test tailnet that exercises Create → Read → Update(role) → Update(suspended) → Destroy on a single membership; assert every step succeeds and the backend records each call as authenticated through the matching transport.

### Tests for User Story 2 (test-first per Constitution §I)

- [ ] T044 [P] [US2] Create `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/membership_api_test.go` with `TestMembershipAPI_RoutesThroughAuthHTTPClient`: per the contract in `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/specs/002-standalone-membership-provider/contracts/auth-transport.md`, install a stub `tailscale.Auth` whose `HTTPClient` returns an `*http.Client` with a marker-injecting `RoundTripper`; for each helper method (`listUserInvites`, `createUserInvite`, `deleteUserInvite`, `suspendUser`, `restoreUser`, `deleteUser`, `updateUserRole`) assert the recorded request carries the marker header AND has no `Authorization: Basic` header. Test MUST fail before T046 lands.
- [ ] T045 [P] [US2] Add `TestMembershipAPI_APIKeyStillUsesBasicAuth` to the same file: build a `tailscale.Client{ APIKey: "test-key", BaseURL: serverURL, ... }` (no `Auth`), drive each helper method against an `httptest.Server` that records the `Authorization` header, assert it equals `Basic <base64("test-key:")>` (regression-safety for API-key mode after the fix). Test MUST fail before T046 lands (because the current code path bypasses `c.HTTP` initialization).

### Implementation for User Story 2

- [ ] T046 [US2] Edit `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/membership_api.go`: at the top of `do()` add `_ = m.Client.Users()` to trigger the v2 client's `init()` (which installs `c.Auth.HTTPClient(...)` on `c.HTTP` for OAuth/Federated Identity), then delete the `if m.Client.APIKey != "" { req.SetBasicAuth(m.Client.APIKey, "") }` branch. Keep the `Content-Type`, `Accept`, and `User-Agent` header logic unchanged. Per `research.md §R1`.
- [ ] T047 [US2] Run `go test ./tailscale/... -run "TestMembershipAPI"`; assert T044 and T045 both now pass. Run `go test -coverprofile=coverage.out ./tailscale/...` and assert `do()` in `membership_api.go` is 100% covered (Constitution §VIII).
- [ ] T048 [US2] Add `TestAccTailscaleMembership_OAuthAuthMode` and `TestAccTailscaleMembership_FederatedIdentityAuthMode` to `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/resource_tailnet_membership_test.go`, each gated by `os.Getenv("TF_ACC") != ""` plus the relevant auth-mode env vars (`TAILSCALE_OAUTH_CLIENT_ID`/`TAILSCALE_OAUTH_CLIENT_SECRET` for OAuth; `TAILSCALE_OAUTH_CLIENT_ID`/`TAILSCALE_IDENTITY_TOKEN` for Federated Identity). Each test runs Create → Read → Update(role admin→member) → Update(suspended true) → Destroy on a unique synthetic `login_name` (e.g. `test-{uuid}@example.com`) and asserts no errors at any step (US2 acceptance scenarios 2 and 3, SC-002).
- [ ] T049 [US2] Confirm the existing API-key-mode acceptance test in `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/resource_tailnet_membership_test.go` still passes (US2 acceptance scenario 1); if absent, add `TestAccTailscaleMembership_APIKeyAuthMode` modeled on T048.
- [ ] T074 [P] [US2] Add `TestProvider_RejectsConflictingAuthModes` to `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/provider_test.go` (FR-008, US2 acceptance scenario 4): drive `providerConfigure` (or `validateProviderCreds` directly) with each invalid combination — (a) `api_key` + `oauth_client_id`, (b) `api_key` + `oauth_client_secret`, (c) `api_key` + `identity_token`, (d) `oauth_client_id` set without either secret or token, (e) `oauth_client_id` + both `oauth_client_secret` and `identity_token`. For each case, assert that the returned `diag.Diagnostics` `HasError()` is true AND that the diagnostic `Summary` contains the literal substring `"conflicting"` or `"mandatory"` (matching the existing upstream-preserved error strings in `validateProviderCreds`). This pins FR-008's "no silent selection" guarantee mechanically rather than relying solely on grandfathered upstream tests.

**Checkpoint**: US2 complete. SC-002 ("zero auth-related fallbacks; zero API-key Basic auth fallbacks observed when OAuth or Federated Identity is the configured mode") is enforced by T044 + T045 in unit tests and T048 in acceptance tests; FR-008's multi-mode-rejection guarantee is enforced by T074. The carried-over correctness fix from feature 001 is shipped.

---

## Phase 5: User Story 3 — Migration from upstream-derived prototype (Priority: P2)

**Goal**: An operator currently managing memberships via the upstream-derived prototype `tailscale_tailnet_membership` resource can switch to this provider with documented HCL and `terraform state mv` commands; `terraform plan` after migration reports no diffs (SC-005).

**Independent Test**: Take a sample HCL config + Terraform state file written against the prototype resource (committed under `specs/002-standalone-membership-provider/quickstart.md` section 4 example); apply the documented migration steps; run `terraform plan` and assert exit code 0 with `No changes.` output.

### Tests for User Story 3

- [ ] T050 [P] [US3] No automated test required for migration; verification is manual per US3 acceptance scenario 1. Add a checklist item under `specs/002-standalone-membership-provider/checklists/requirements.md` (or create `checklists/migration.md`) tracking the manual `terraform state mv` walkthrough against a dev tailnet before tagging v0.1.0.

### Implementation for User Story 3

- [ ] T051 [US3] The migration guide is already authored in `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/specs/002-standalone-membership-provider/quickstart.md` section 4 (steps 4.1–4.7). Verify it includes: provider source change, provider block rename, resource type rename, exact `terraform state mv` command per resource, `terraform state replace-provider` command, verification step, and the source-address-drift note.
- [ ] T052 [US3] Add a "Migration from the upstream-derived prototype" section to the README (see T055) that links to `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/specs/002-standalone-membership-provider/quickstart.md#4-migration-from-the-upstream-derived-prototype`, satisfying FR-021's "in the README or a dedicated docs page" requirement.

**Checkpoint**: US3 complete. The migration path exists, is discoverable from the README, and is verifiable manually against a worked example.

---

## Phase 6: User Story 4 — Tag-triggered, GPG-signed, Registry-shaped release (Priority: P2)

**Goal**: A Git tag matching either `vX.Y.Z` or `vX.Y.Z-{alpha,beta,rc}.N` triggers GoReleaser to build the 11-platform OS/arch matrix, GPG-sign the checksums file, and publish a GitHub Release. Tags not matching either pattern do NOT trigger a release. Pre-release tags are marked "pre-release" in the GitHub UI. The release pipeline fails loudly if `GPG_FINGERPRINT` is unset.

**Independent Test**: Run `goreleaser release --snapshot --clean` locally (no GPG required for snapshot mode); assert the produced `dist/` directory contains exactly 11 zip archives matching the expected OS/arch pattern, plus a `SHA256SUMS` file. Then push a no-op pre-release tag (e.g. `v0.0.1-rc.0`) to a test branch on a fork; observe the release workflow fires and produces the expected artifacts.

### Tests for User Story 4

- [ ] T053 [P] [US4] Add `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/scripts/test-release-snapshot.sh`: runs `goreleaser release --snapshot --clean --skip=publish,sign` and asserts ALL of the following via concrete shell checks on `dist/`: (a) `find dist -name '*.zip' | wc -l` returns exactly 11; (b) every archive name matches the regex `^terraform-provider-tailscale-membership_[^_]+_(darwin|linux|windows|freebsd)_(amd64|arm64|arm|386)\.zip$` covering the FR-014 OS/arch matrix; (c) `dist/terraform-provider-tailscale-membership_*_SHA256SUMS` exists and contains exactly 11 lines (one per archive); (d) `dist/terraform-provider-tailscale-membership_*_manifest.json` exists, parses as valid JSON via `jq -e .`, contains a top-level `version` field equal to `"5.0"`, and contains a `metadata.protocol_versions` array including `"5.0"` (the Terraform Plugin Protocol version emitted by Plugin SDK v2; Registry rejects manifests missing this shape). Wire this script into `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/.github/workflows/ci.yml` as a non-required job (informational only on PRs; required on `main`).

### Implementation for User Story 4

- [ ] T054 [US4] Edit `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/.goreleaser.yml` per `research.md §R5`: update `ldflags` to `-X github.com/pmpaulino/terraform-provider-tailscale-membership/tailscale.providerVersion={{.Version}}`; keep `goos: [linux, darwin, windows, freebsd]` and `goarch: [amd64, arm64, "386", arm]`; replace the single `darwin/386` ignore with the five exclusions needed to land exactly the 11 FR-014 pairs (`darwin/386`, `darwin/arm`, `windows/arm64`, `windows/arm`, `freebsd/arm64`); preserve the `signs:` block and `release.prerelease: auto` setting (the latter satisfies FR-014's "pre-release tags MUST be marked as 'pre-release'").
- [ ] T055 [US4] Edit `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/.github/workflows/release.yml`: replace the `tags: [v*]` filter with the four explicit glob patterns from `research.md §R3` (`v[0-9]+.[0-9]+.[0-9]+`, `v[0-9]+.[0-9]+.[0-9]+-alpha.[0-9]+`, `v[0-9]+.[0-9]+.[0-9]+-beta.[0-9]+`, `v[0-9]+.[0-9]+.[0-9]+-rc.[0-9]+`); leave the rest of the job (GPG import, GoReleaser run) unchanged. Verify the existing GPG import step's `gpg_private_key` requirement guarantees a hard failure when the secret is unset (FR-016).

**Checkpoint**: US4 complete. SC-004 ("a Git tag matching the release pattern produces a complete, GPG-signed, Registry-shaped release artifact set ... with no manual post-processing required") is satisfied. T053 makes the 11-archive guarantee mechanically testable in CI without needing a real tag.

---

## Phase 7: User Story 5 — Operator-discoverable docs (Priority: P3)

**Goal**: The repository's `docs/` directory contains exactly one provider-configuration page and one resource page (the membership resource), each documenting every schema argument and attribute; HCL examples in each page run cleanly with only credential substitution. README is rewritten to v0.1 shape.

**Independent Test**: Open `docs/`, confirm there is one `index.md` and exactly one file under `docs/resources/` (`tailnet_membership.md`); cross-reference every argument/attribute in `tailscale/provider.go`'s schema and in `resourceTailnetMembership()`'s schema against the docs page; assert 100% coverage in both directions (SC-007).

### Tests for User Story 5

- [ ] T056 [P] [US5] Add `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/scripts/check-docs-coverage.sh`: performs BOTH of the following to enforce 100% bidirectional schema↔docs coverage (FR-019, SC-007), since `tfplugindocs validate` alone only checks structural well-formedness, not whether every schema field is documented or whether every documented field still exists in the schema: (a) runs `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate` and asserts exit 0; (b) runs `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --rendered-provider-name="tailscale_membership"` into a temp directory, then `diff -ru docs/ <tempdir>/docs/` and asserts the diff is empty (any drift between handwritten docs and the schema-derived docs fails CI). Wire into `.github/workflows/ci.yml` as a required job.

### Implementation for User Story 5

- [ ] T057 [P] [US5] Delete every file under `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/docs/resources/` except `tailnet_membership.md` (run a directory listing first to enumerate and delete; do not remove the directory itself).
- [ ] T058 [P] [US5] Delete `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/docs/data-sources/` in its entirety (FR-002 — no data sources in v0.1).
- [ ] T059 [US5] Rewrite `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/docs/index.md`: change page title to `tailscale_membership Provider`; describe v0.1 scope (membership only); document all three auth modes; include a complete `terraform { required_providers { tailscale_membership = { source = "pmpaulino/tailscale-membership" } } }` block in every example (FR-011); link to migration guide; link to GPG signing key (T003).
- [ ] T060 [US5] Update the resource page so it matches the renamed schema and survives `go generate ./...` (the `//go:generate` directive in `main.go` runs `tfplugindocs generate`, which regenerates `docs/resources/*.md` from the schema and templates). Execute in this exact order, otherwise the bidirectional-coverage check from T056 will fail: **(step 1)** run `go generate ./...` once after Phase 2 has renamed the resource type, so `docs/resources/tailnet_membership.md` regenerates with the new `tailscale_membership_tailnet_membership` name and current schema. **(step 2)** Verify the regenerated file's frontmatter `description:` and page title now reference `tailscale_membership_tailnet_membership`; if `tfplugindocs` did not pick up the rename, fix the schema-side `Description` strings in `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/tailscale/resource_tailnet_membership.go` and re-run step 1. **(step 3)** For any handwritten narrative additions that are NOT derivable from the schema (e.g., the `required_providers` alias block per FR-011 prepended to the first example, the link to the migration guide from `quickstart.md` §4, the link to the GPG key from T003/T075), add them to `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/templates/resources/tailnet_membership.md.tmpl` (create the `templates/` tree if it does not yet exist) — NEVER edit `docs/resources/tailnet_membership.md` directly, since the next `go generate` would overwrite those edits and T056 would fail. **(step 4)** Re-run `go generate ./...` to render the templates into `docs/`, and confirm `git diff` shows the expected narrative additions in `docs/resources/tailnet_membership.md` (FR-019, SC-007).
- [ ] T061 [P] [US5] Move `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/examples/resources/tailscale_tailnet_membership/` to `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/examples/resources/tailscale_membership_tailnet_membership/`; in `resource.tf` and `import.sh`, rename every `tailscale_tailnet_membership` to `tailscale_membership_tailnet_membership` and prepend the `required_providers` alias block to `resource.tf` (FR-011).
- [ ] T062 [US5] Delete every other directory under `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/examples/resources/` (everything except the renamed `tailscale_membership_tailnet_membership/`); delete `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/examples/data-sources/` in its entirety.
- [ ] T063 [US5] Rewrite `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/README.md` per FR-020 + FR-021: provider purpose (membership-only fork), supported auth modes, install instructions (dev override + tagged GitHub Release), migration guide section linking to `quickstart.md` section 4, **a "Verifying releases" section that links to the in-repo `/KEYS` file produced by T075 with the key fingerprint inline (FR-015 — the key MUST be discoverable from the repository, not only documented in a private note)**, and link to upstream project per the NOTICE attribution.

**Checkpoint**: US5 complete. SC-001 (a new operator can install + apply within 15 minutes from README + docs alone) and SC-007 (100% docs/schema bidirectional coverage) are satisfied. T056's `tfplugindocs validate` enforces SC-007 in CI.

---

## Phase 8: User Story 6 — License & attribution compliance (Priority: P3)

**Goal**: `LICENSE` is the MIT license retaining the upstream copyright lines; `NOTICE` names and links to the upstream project and reproduces its copyright; no license-incompatible dependencies remain.

**Independent Test**: Open `LICENSE` and `NOTICE`; visually confirm content matches FR-022/FR-023; run `go list -m -json all | jq -r '.Path'` and a license scanner (or manual review) against the `go.sum` set; assert no GPL/AGPL/SSPL/etc. dependencies (FR-024).

### Implementation for User Story 6

- [ ] T064 [P] [US6] Verify `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/LICENSE` already contains the MIT text and both upstream copyright lines (`Copyright (c) 2021 David Bond` + `Copyright (c) 2024 Tailscale Inc & Contributors`); no edit needed if intact (FR-022).
- [ ] T065 [P] [US6] Create `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/NOTICE` per the template in `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/specs/002-standalone-membership-provider/research.md` §R4 (FR-023): names `terraform-provider-tailscale`, links to its GitHub URL, reproduces upstream copyright, states the hard-fork relationship in plain English.
- [ ] T066 [P] [US6] Run a manual license review of `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/go.sum` (post-`go mod tidy` from T036); produce a one-line summary in the v0.1.0 release notes confirming no license-incompatible dependencies (FR-024). Reusable tooling: `go-licenses report ./... --template <(echo '{{range .}}{{.Name}},{{.LicenseName}}\n{{end}}')`.
- [ ] T075 [P] [US6] Create `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/KEYS` containing the ASCII-armored **public** half of the project's release-signing GPG key (output of `gpg --armor --export <fingerprint>`) — never the private key, never the passphrase. Above the armored block, include a plain-text header listing the key fingerprint, owner email, and creation date so operators can cross-check before importing. Commit this file at the repository root so it is `https://github.com/pmpaulino/terraform-provider-tailscale-membership/blob/main/KEYS`-discoverable, satisfying FR-015's "discoverable from the repository" requirement that T003's local-only note alone does not meet. Cross-link the file from `README.md` (T063) and from `quickstart.md` §5 (already drafted). Validation: `gpg --import KEYS` followed by `gpg --verify dist/terraform-provider-tailscale-membership_<version>_SHA256SUMS.sig dist/terraform-provider-tailscale-membership_<version>_SHA256SUMS` succeeds against a snapshot from T053 (snapshot mode skips signing, so for this validation step run `goreleaser release --snapshot --clean` *with* signing enabled by exporting `GPG_FINGERPRINT` locally).

**Checkpoint**: US6 complete. SC-006 ("license/attribution posture passes a routine open-source license review with no findings") is satisfied.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final sweep before tagging v0.1.0.

- [ ] T067 [P] Run `git grep "tailscale/tailscale"` and `git grep "tailscale_tailnet_membership"` (without the `_membership_` prefix) across the repo (excluding `specs/001-tailscale-user-management/` and `specs/002-standalone-membership-provider/`); both MUST return zero matches outside of the `specs/` directories. Any stray match is an oversight in Phase 2 or Phase 7.
- [ ] T068 [P] Run `golangci-lint run ./...` from repo root; fix any lint findings introduced by the deletions/renames (most likely unused imports or unused helper functions in `provider.go` that were referenced only by deleted resources).
- [ ] T069 [P] Run the full quickstart scenarios from `/Users/pat/Projects/pmpaulino/terraform-provider-tailscale/specs/002-standalone-membership-provider/quickstart.md` sections 1–3 against a dev tailnet for each of the three auth modes (manual validation; SC-001 + SC-002 end-to-end check).
- [ ] T070 [P] Run the migration walkthrough from `quickstart.md` section 4 against a dev tailnet with a worked example; assert `terraform plan` post-migration reports `No changes.` (SC-005).
- [ ] T071 [P] Run T053's `scripts/test-release-snapshot.sh` locally and confirm exactly 11 archives are produced, named per the FR-014 matrix.
- [ ] T072 Tag a pre-release on a personal fork (`v0.0.1-rc.0`) and observe `.github/workflows/release.yml` runs end-to-end with GPG signing and produces a GitHub Release marked "pre-release" (US4 acceptance scenarios 1–4, SC-004). Use this as the dress-rehearsal for v0.1.0.
- [ ] T073 Tag `v0.1.0` on `main` after all of the above pass; verify the release artifacts on GitHub against `quickstart.md` section 5.

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
