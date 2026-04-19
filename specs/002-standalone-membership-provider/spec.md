# Feature Specification: Standalone Tailscale Membership Provider

**Feature Branch**: `002-standalone-membership-provider`  
**Created**: 2026-04-18  
**Status**: Draft  
**Input**: User description: "Build a standalone Terraform provider, pmpaulino/tailscale-membership, derived from the upstream terraform-provider-tailscale (MIT-licensed, attributed). The provider exposes only Tailscale tailnet membership management."

## Clarifications

### Session 2026-04-18

- Q: What is the final Go module path for this fork? → A: `github.com/pmpaulino/terraform-provider-tailscale-membership` (matches the `terraform-provider-<NAME>` repo convention used by the Terraform community and by upstream; aligns with GoReleaser's standard provider release tooling).
- Q: What local provider type and resource type prefix MUST the provider use in HCL? → A *(amended 2026-04-18 during /speckit.implement Phase 7)*: Terraform local name **`tailscale-membership`** (with a dash); resource type prefix `tailscale_membership_`; the membership resource type is therefore `tailscale_membership_tailnet_membership`. Because the source address `pmpaulino/tailscale-membership` contains a dash, every consumer's `required_providers` block MUST alias it explicitly (e.g. `tailscale-membership = { source = "pmpaulino/tailscale-membership" }`); this alias MUST appear in every documentation example and in the migration guide. Additionally, every membership resource block MUST carry `provider = tailscale-membership` to override Terraform's implicit binding of `tailscale_*` resource types to the upstream `tailscale/tailscale` provider.

  **Original answer (rolled back)**: local name was originally proposed as `tailscale_membership` (with an underscore). Discovered during `/speckit.implement` that Terraform's CLI rejects underscores in provider local names (`must contain only letters, digits, and dashes, and may not use leading or trailing dashes`). The dashed form is the only Terraform-valid spelling. The resource-type prefix `tailscale_membership_` is unchanged because HCL resource identifiers must use underscores, not dashes — which is precisely the source of the dash↔underscore mismatch and the reason every resource block needs an explicit `provider = tailscale-membership` attribute.
- Q: Which OS/arch platforms MUST every tagged release build for? → A: The conventional Terraform Registry "first-class" matrix: `linux/amd64`, `linux/arm64`, `linux/386`, `linux/arm`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/386`, `freebsd/amd64`, `freebsd/arm`, `freebsd/386` (matches GoReleaser's default `terraform-provider` template and the Registry's expectations for a complete provider release).
- Q: Which Git tag patterns trigger a release, and how are pre-releases handled? → A: Production releases match `v<MAJOR>.<MINOR>.<PATCH>` (e.g. `v0.1.0`); pre-releases match `v<MAJOR>.<MINOR>.<PATCH>-(alpha|beta|rc).<N>` (e.g. `v0.1.0-rc.1`). Both patterns trigger the release pipeline and publish a GitHub Release; pre-release tags MUST be marked as "pre-release" in the GitHub Release UI and MUST NOT be treated as the latest release. Tags not matching either pattern MUST NOT trigger a release.
- Q: Where MUST the feature-001 remediation backlog (last-admin pre-flight, error-surfacing in downgrade-on-destroy, test assertion strength, pending-update behavior, tailnet fallback removal) be recorded? → A: A dedicated `specs/002-standalone-membership-provider/backlog.md` file. Each entry MUST contain: a one-line summary, a link to its origin in the feature 001 `/speckit.analyze` output (or to the relevant section of `specs/001-tailscale-user-management/`), and a `Status:` field (e.g. `Deferred to v0.2+`). The backlog MUST NOT be merged into `tasks.md` (so v0.1 task execution does not accidentally pick up backlog items) and MUST NOT live solely in an external issue tracker (so reviewers see the full backlog in PRs).

## Context

This feature delivers v0.1 of a hard-forked, single-purpose Terraform provider. It carries forward the membership resource designed in feature `001-tailscale-user-management` and packages it as an independently releasable provider distributed from this repository. The feature is intentionally about *packaging, scoping, and distribution* of the membership capability — the membership behavior itself is defined by feature 001 and is not redefined here.

Operators using Terraform are the primary audience. They want a small, focused provider that does exactly one job (manage tailnet memberships) without inheriting the surface area, dependencies, or release cadence of the full upstream provider.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Use the standalone provider to manage tailnet memberships (Priority: P1)

A Terraform operator declares the `pmpaulino/tailscale-membership` provider in their configuration, configures it for their tailnet, and manages tailnet memberships (create, read, update, destroy) using a single resource type exposed by the provider. No unrelated Tailscale resources (DNS, ACLs, devices, keys, webhooks, posture, contacts, settings, OAuth clients, etc.) appear in the provider's schema or documentation.

**Why this priority**: This is the entire purpose of v0.1. Without it, the provider has no value. It is also the smallest end-to-end slice that proves the fork is viable: a user can install it, configure it, and manage at least one membership.

**Independent Test**: Install the provider locally (e.g. via a dev override or a tagged release artifact), declare a single membership resource in HCL, run `terraform plan` and `terraform apply` against a test tailnet, and observe that the membership is created and visible in the tailnet. Confirm via `terraform providers schema` that no resources other than the membership resource (and the provider configuration block) are exposed.

**Acceptance Scenarios**:

1. **Given** an operator has installed the provider and configured it with valid tailnet credentials, **When** they declare a membership resource for a new identity and run apply, **Then** the membership is created in the tailnet (sending an invite when the identity is not yet a member) and the resource appears in state with the correct attributes.
2. **Given** an operator inspects the provider's published schema, **When** they list resources and data sources exposed by the provider, **Then** only the membership resource and the provider configuration block are present; no DNS, ACL, device, key, webhook, posture, contact, settings, AWS-external-ID, or OAuth-client resources are present.
3. **Given** an operator authors HCL that references any resource type that existed in the upstream provider but is out of scope here (e.g. `tailscale_membership_dns_nameservers`), **When** they run `terraform validate` or `terraform plan`, **Then** Terraform reports a clear "unknown resource type" error originating from the provider's schema, not a runtime crash.

---

### User Story 2 - Authenticate via OAuth, API key, or Federated Identity (Priority: P1)

An operator can configure the provider to authenticate against the Tailscale Control API using any of the three auth modes the upstream provider supports: OAuth (client ID/secret with scopes), an API key, or Federated Identity. The membership operations (create, read, update, destroy) work correctly under each of the three modes.

**Why this priority**: P1 alongside US1 because the membership resource is unusable without working auth, and this story carries a known correctness fix from feature 001: the membership API helper in the upstream codebase routed only through API-key Basic auth, so OAuth and Federated Identity were silently broken for membership-specific calls. Shipping v0.1 without fixing this would deliver a non-functional product to two of the three documented auth modes.

**Independent Test**: For each auth mode, configure the provider with credentials valid for that mode against a test tailnet, run a full create/read/update/destroy cycle on a single membership, and assert that every step succeeds with no auth-related errors and that the request is observed by the backend as authenticated through the expected transport.

**Acceptance Scenarios**:

1. **Given** the provider is configured with API key credentials, **When** the operator performs each of create, read, update, and destroy on a membership, **Then** every operation completes successfully and the backend records the call as API-key-authenticated.
2. **Given** the provider is configured with OAuth client credentials and the appropriate scope(s), **When** the operator performs each of create, read, update, and destroy on a membership, **Then** every operation completes successfully and the backend records the call as OAuth-authenticated. The operation MUST NOT fall back to or require API-key Basic auth.
3. **Given** the provider is configured for Federated Identity, **When** the operator performs each of create, read, update, and destroy on a membership, **Then** every operation completes successfully and the backend records the call as Federated-Identity-authenticated. The operation MUST NOT fall back to or require API-key Basic auth.
4. **Given** the operator supplies more than one auth mode in the provider block (e.g. both an API key and OAuth credentials), **When** the provider initializes, **Then** the operator is presented with a clear, actionable diagnostic explaining which combinations are valid and which one was actually selected (or that the configuration is rejected), consistent with upstream behavior.

---

### User Story 3 - Migrate from the upstream-shaped membership resource with minimal config change (Priority: P2)

An operator who is currently managing tailnet memberships using the prototype membership resource from the upstream-derived codebase (i.e. the resource designed in feature 001) can switch to `pmpaulino/tailscale-membership` by changing only the provider source address and the resource-type prefix in their HCL, plus running a documented state-migration step. Their existing schema (resource attributes, identity values, role values) and existing tailnet state (memberships, invites) are preserved.

**Why this priority**: P2 because there are not yet large numbers of users on the prototype, but the migration path must exist on day one or early adopters are stranded. Documented at v0.1 release; no in-provider tooling required beyond what `terraform state` already provides.

**Independent Test**: Take an HCL configuration written against the upstream-shaped membership resource and a corresponding Terraform state. Apply the documented migration steps (change provider source, change resource type, run the documented state move). Run `terraform plan` and confirm the result is "no changes" — i.e. state is preserved and the membership in the tailnet is unaffected.

**Acceptance Scenarios**:

1. **Given** an operator has an HCL config and state managing memberships via the upstream-derived prototype resource, **When** they follow the migration steps documented in the README and the docs site, **Then** after migration `terraform plan` shows no diff and the underlying memberships in the tailnet are unchanged.
2. **Given** an operator follows the migration documentation, **When** they reach the end of the procedure, **Then** they have a single, runnable example showing the before-config, the after-config, and the exact state-migration commands.
3. **Given** the migration documentation, **When** read by a Terraform-literate operator who has not seen this provider before, **Then** they can complete the migration without needing to read the provider source code.

---

### User Story 4 - Release signed, Registry-shaped artifacts on a Git tag (Priority: P2)

A maintainer pushes a Git tag (e.g. `v0.1.0`) to the GitHub repository. CI automatically builds the provider for the platforms required by the Terraform Registry, signs the release artifacts with a configured GPG key, and publishes the signed artifacts plus a checksums file to a GitHub Release. The artifacts are shaped (file naming, manifest, SHA256SUMS, signature file) such that they could be submitted to the Terraform Registry without further repackaging — Registry submission itself is deferred to a later version.

**Why this priority**: P2 because operators in US1 only need a working binary; they can install via a dev override or `unzip` against any GitHub Release. Signed Registry-shaped artifacts make the eventual Registry submission a non-event and let early adopters install with normal Terraform plumbing today.

**Independent Test**: Tag a release on the repository's main branch. Confirm that within the CI run a GitHub Release is published containing per-platform zip artifacts, a SHA256SUMS file, a signature file for the SHA256SUMS, and a manifest file. Verify the GPG signature against the published public key and verify each artifact's SHA256 matches the SHA256SUMS file.

**Acceptance Scenarios**:

1. **Given** a maintainer pushes a Git tag matching the release pattern, **When** CI completes, **Then** a GitHub Release exists at that tag containing platform-specific zip archives for the platforms required by the Terraform Registry, a `SHA256SUMS` file, and a detached GPG signature file for `SHA256SUMS`.
2. **Given** a published release, **When** an external party verifies the GPG signature using the project's published public key, **Then** verification succeeds.
3. **Given** a published release, **When** an external party recomputes the SHA256 of each zip, **Then** every value matches the corresponding entry in `SHA256SUMS`.
4. **Given** a published release, **When** the artifact set is inspected against the Terraform Registry's documented file-layout requirements, **Then** the layout matches and the only blocker to Registry availability is the (deferred) Registry submission step itself.

---

### User Story 5 - Discover the provider through documentation (Priority: P3)

An operator visiting the repository can find a `docs/` directory laid out the same way as the upstream provider's docs (provider configuration page + resource page) so it is immediately recognizable to anyone familiar with Terraform provider conventions. Each documented surface (provider configuration, the membership resource) has a runnable example.

**Why this priority**: P3 because the README plus example files alone are enough to evaluate the provider, but a Registry-shaped `docs/` layout is a hard prerequisite for the eventual v0.2 Registry submission and a strong ergonomics win immediately. Doing it at v0.1 avoids retrofitting later.

**Independent Test**: Open the repository and navigate `docs/`. Confirm the directory contains a provider page and a resource page for the membership resource, each with a description, argument reference, attribute reference (where relevant), and at least one runnable HCL example. Cross-check that every argument and attribute appearing in the provider's schema appears in the docs.

**Acceptance Scenarios**:

1. **Given** the repository at the v0.1 tag, **When** an operator opens `docs/`, **Then** they find a provider page and a resource page for the membership resource, organized using the same conventions as the upstream provider's `docs/` layout.
2. **Given** the docs pages, **When** an operator compares them to the provider's published schema, **Then** every argument and attribute exposed by the provider is documented, and no documented argument or attribute is absent from the schema.
3. **Given** the docs pages, **When** the example HCL in each page is copied into a fresh project and run against a test tailnet, **Then** it applies cleanly with only credential substitution required.

---

### User Story 6 - License and attribution compliance (Priority: P3)

A reviewer (or downstream consumer) inspecting the repository can immediately confirm that this fork preserves the upstream MIT license and clearly attributes the upstream project. A `LICENSE` file containing the MIT license is present at the repository root, and a `NOTICE` (or equivalent attribution file) names the upstream `terraform-provider-tailscale` project, links to it, and reproduces its copyright notice.

**Why this priority**: P3 because it is a one-time setup with low ongoing maintenance, but it is non-negotiable for any MIT-derived distribution. Doing it at v0.1 prevents license-cleanup churn at every later release.

**Independent Test**: At the v0.1 tag, open the repository root and confirm `LICENSE` is the MIT license and `NOTICE` (or equivalent) names and links to the upstream project and reproduces its copyright notice. Run any common license-scanning tool (or manual review) and confirm there are no inherited GPL/AGPL/etc. dependencies and that the MIT obligations are met.

**Acceptance Scenarios**:

1. **Given** the repository at the v0.1 tag, **When** a reviewer opens `LICENSE`, **Then** it contains the MIT license text including the upstream copyright line.
2. **Given** the repository at the v0.1 tag, **When** a reviewer opens the attribution file (e.g. `NOTICE`), **Then** the upstream `terraform-provider-tailscale` project is named, linked, and credited as the source of the derived code.
3. **Given** the repository at the v0.1 tag, **When** the dependency graph is inspected, **Then** all third-party licenses are compatible with MIT redistribution and no license-incompatible dependency is present.

---

### Edge Cases

- An operator's HCL still references an upstream resource type that was removed in this fork (e.g. `tailscale_membership_dns_nameservers`): Terraform MUST surface a clear "unknown resource type" error from the provider's schema; the provider MUST NOT panic, ignore the resource, or treat it as an empty resource.
- An operator configures both API-key and OAuth credentials in the same provider block: behavior matches upstream (clear diagnostic about which combinations are valid; deterministic selection or rejection); the provider MUST NOT silently use one and discard the other without telling the operator.
- An operator authenticated via OAuth or Federated Identity performs a membership operation: the request MUST go through the matching transport (OAuth or Federated Identity) and MUST NOT fall back to API-key Basic auth. This is the explicit correctness fix carried over from feature 001.
- The release pipeline runs without a configured GPG signing key: the release MUST fail loudly (no unsigned artifacts published) rather than silently producing unsigned artifacts.
- An operator points Terraform at an older upstream-provider source address (`tailscale/tailscale`) expecting this fork's behavior: the provider source address mismatch is a Terraform-level error, not this provider's responsibility, but the migration documentation MUST call this out so operators are not surprised.
- The repository is consumed as a Go module by another project: the module path MUST resolve to this fork, not to the upstream module path; importing both side-by-side MUST be possible without a Go module collision (covered by the rename in Constraints).
- A user installs v0.1 via dev override and later switches to a Registry-style install once available: the provider source address MUST remain `pmpaulino/tailscale-membership` so no HCL edits are required at that switch.

## Requirements *(mandatory)*

### Functional Requirements

#### Provider scope and surface

- **FR-001**: The provider MUST expose exactly one managed resource type: the tailnet membership resource as defined in feature 001. No additional managed resources MUST be exposed.
- **FR-002**: The provider MUST NOT expose any data sources in v0.1. (The memberships data source and the single-user data source are deferred to v0.2+.)
- **FR-003**: The provider's published schema MUST NOT include any of the upstream resources unrelated to membership: DNS settings, DNS nameservers, DNS preferences, DNS search paths, DNS split nameservers, ACL, device authorization, device key, device subnet routes, device tags, devices, OAuth clients, contacts, posture integration, settings, AWS external IDs, webhooks, or tailnet keys. The corresponding source files MUST be removed from the codebase rather than retained but unregistered.
- **FR-004**: The provider configuration block MUST accept the same configuration arguments as the upstream provider's configuration block for the auth modes in scope (tailnet selection, base URL, OAuth client ID/secret/scopes, API key, Federated Identity options). Argument names and semantics MUST match upstream so that operators familiar with the upstream provider can configure this provider without learning a new schema.

#### Authentication

- **FR-005**: The provider MUST support API-key authentication for all membership operations.
- **FR-006**: The provider MUST support OAuth client-credentials authentication for all membership operations. The membership API helper(s) used by the resource MUST route requests through the OAuth-authenticated HTTP transport and MUST NOT require or fall back to API-key Basic auth. (This is the correctness fix carried over from feature 001.)
- **FR-007**: The provider MUST support Federated Identity authentication for all membership operations. The membership API helper(s) used by the resource MUST route requests through the Federated-Identity-authenticated HTTP transport and MUST NOT require or fall back to API-key Basic auth.
- **FR-008**: When more than one auth mode is configured simultaneously, the provider MUST behave consistently with the upstream provider's documented precedence/rejection rules, and the resulting selection (or rejection) MUST be communicated to the operator via a clear diagnostic — no silent selection.

#### Resource behavior parity

- **FR-009**: The membership resource exposed by this provider MUST be behaviorally equivalent to the membership resource specified in feature 001 (`specs/001-tailscale-user-management/spec.md`). Any divergence MUST be explicitly listed and justified; v0.1 MUST NOT introduce silent behavioral changes relative to feature 001.
- **FR-010**: The remediation findings recorded against feature 001 (last-admin pre-flight handling, error-surfacing in downgrade-on-destroy, test assertion strength, pending-update behavior, tailnet fallback removal) MUST be captured in a dedicated file at `specs/002-standalone-membership-provider/backlog.md`. Each entry MUST contain: a one-line summary, a link to its origin in feature 001's `/speckit.analyze` output (or to the relevant section of `specs/001-tailscale-user-management/`), and a `Status:` field (e.g. `Deferred to v0.2+`). The backlog MUST NOT be merged into this feature's `tasks.md`, and MUST NOT live solely in an external issue tracker. The backlog items are NOT blockers for v0.1 release, but they MUST NOT be silently dropped.

#### Identity, naming, and module path

- **FR-011** *(amended 2026-04-18 during /speckit.implement Phase 7 — see Q3 amendment in the Clarifications session)*: The Terraform provider source address MUST be `pmpaulino/tailscale-membership` (i.e. `registry.terraform.io/pmpaulino/tailscale-membership` once Registry-published in a later version). The provider's **Terraform local name** (used inside `required_providers` and `provider "..."` blocks) MUST be `tailscale-membership` (with a dash — Terraform's CLI rejects underscores in provider local names). The provider's **resource-type prefix** (used as the first underscore-separated segment of every HCL resource type) is `tailscale_membership_` (with an underscore — HCL resource identifiers cannot contain dashes); the membership resource type is therefore `tailscale_membership_tailnet_membership`. Every consumer's `required_providers` block MUST alias the source explicitly to the local name, AND every membership resource block MUST carry `provider = tailscale-membership` to override Terraform's default mapping of the `tailscale_*` resource prefix to the upstream `tailscale/tailscale` provider:

    ```hcl
    terraform {
      required_providers {
        tailscale-membership = {
          source = "pmpaulino/tailscale-membership"
        }
      }
    }

    resource "tailscale_membership_tailnet_membership" "example" {
      provider   = tailscale-membership
      login_name = "alice@example.com"
    }
    ```

    Both the `required_providers` alias and the `provider = tailscale-membership` attribute on each resource block MUST appear in every documentation example, in every example under `examples/`, and in the migration guide, so that operators copy a working config rather than discovering the dash↔underscore mismatch themselves.
- **FR-012**: The Go module path declared in `go.mod` MUST be `github.com/pmpaulino/terraform-provider-tailscale-membership`. The corresponding GitHub repository MUST be named `terraform-provider-tailscale-membership` to match the module path and Terraform community release-tooling expectations (GoReleaser binary naming, Terraform Registry release-shape conventions). All Go import paths MUST reflect this module path, with no remaining references to the upstream `github.com/tailscale/terraform-provider-tailscale` module.
- **FR-013**: The provider MUST depend on the upstream Tailscale API client library `tailscale.com/client/tailscale/v2` only — it MUST NOT depend on the upstream `terraform-provider-tailscale` Go module, since this is a hard fork rather than a wrapper.

#### Release engineering

- **FR-014**: A GoReleaser-driven release pipeline MUST run on Git tags matching either the production release pattern `v<MAJOR>.<MINOR>.<PATCH>` (e.g. `v0.1.0`) or the pre-release pattern `v<MAJOR>.<MINOR>.<PATCH>-(alpha|beta|rc).<N>` (e.g. `v0.1.0-rc.1`). Tags not matching either pattern MUST NOT trigger a release. Per matched tag, the pipeline MUST produce: per-platform zip archives, a `SHA256SUMS` file covering those archives, a detached GPG signature file for `SHA256SUMS`, and a manifest file conforming to the Terraform Registry's documented layout. The release MUST cover, at minimum, the following OS/arch matrix: `linux/amd64`, `linux/arm64`, `linux/386`, `linux/arm`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/386`, `freebsd/amd64`, `freebsd/arm`, `freebsd/386`. Missing any of these archives from a tagged release MUST cause the release pipeline to fail (no partial releases). Pre-release tags MUST be published as GitHub Releases marked "pre-release" and MUST NOT be promoted to the GitHub "latest release" pointer.
- **FR-015**: Released archives MUST be GPG-signed with a project-controlled key, and the corresponding public key MUST be discoverable from the repository (e.g. linked from the README) so that consumers can verify signatures.
- **FR-016**: If the GPG signing key (or any other required secret) is unavailable at release time, the release pipeline MUST fail loudly. It MUST NOT publish unsigned artifacts as a fallback.
- **FR-017**: The v0.1 release MUST NOT submit the provider to the Terraform Registry. Registry submission is explicitly deferred to v0.2 once a stable release line exists.

#### Documentation

- **FR-018**: The repository MUST contain a `docs/` directory laid out using the same conventions as the upstream provider's `docs/` directory, containing at minimum: one provider configuration page and one page for the membership resource.
- **FR-019**: Each documentation page MUST include a description, an argument reference, an attribute reference (where applicable), and at least one runnable HCL example. Every argument and attribute exposed by the provider's schema MUST be documented; no argument or attribute may be present in docs without being in the schema.
- **FR-020**: The repository README MUST describe: provider purpose, supported auth modes, installation (dev override and tagged GitHub Release), and a pointer to the migration guide (FR-021).
- **FR-021**: A migration guide MUST exist (in the README or a dedicated docs page) describing how an operator currently using the upstream-derived prototype membership resource can switch to this provider with minimal HCL changes (provider source address change, resource type prefix change, and the exact `terraform state mv` (or equivalent) commands required to preserve state).

#### Licensing and attribution

- **FR-022**: The repository MUST contain a `LICENSE` file with the MIT license text, including the upstream project's original copyright line.
- **FR-023**: The repository MUST contain a `NOTICE` file (or equivalent attribution file referenced from the README) that names the upstream `terraform-provider-tailscale` project, links to its source, and credits it as the origin of the derived code.
- **FR-024**: All third-party dependencies introduced or retained MUST be license-compatible with MIT redistribution. License-incompatible dependencies MUST be removed before v0.1 release.

#### Sync and ongoing maintenance

- **FR-025**: This provider is a hard fork. There MUST be no automated upstream-sync mechanism in v0.1. Any future cherry-pick from the upstream project MUST be done manually and MUST be reviewed independently.

### Key Entities

- **Provider (`pmpaulino/tailscale-membership`)**: The Terraform plugin published from this repository. Has a configuration block (auth + tailnet selection) and exposes the membership resource. Identified to Terraform by the source address `pmpaulino/tailscale-membership` and to Go consumers by its module path (FR-012).
- **Membership resource**: The single managed resource exposed by the provider. Behavior is defined by feature 001. Resource type identifier in HCL is `tailscale_membership_tailnet_membership`. The provider's Terraform local name is `tailscale-membership` (dashed); every resource block MUST carry `provider = tailscale-membership` because the HCL resource prefix `tailscale_*` would otherwise default-bind to the upstream `tailscale/tailscale` provider. See FR-011.
- **Auth mode**: One of {API key, OAuth client credentials, Federated Identity}. Each mode is configured through provider arguments matching upstream conventions and selects the HTTP transport used by every membership API call.
- **Release artifact set**: The collection of files produced for a single Git tag — per-platform zips, `SHA256SUMS`, GPG signature, and Registry manifest — matching the Terraform Registry's documented layout but not yet submitted to the Registry.
- **Documentation page**: A markdown file under `docs/` describing either the provider configuration block or the membership resource. Every argument and attribute on the page corresponds to one in the schema, and vice versa.
- **Migration guide**: A document describing the exact HCL and `terraform state` operations required for an operator currently using the upstream-derived prototype membership resource to switch to this provider without touching the underlying tailnet state.

### Assumptions

- The membership resource's behavior, schema, and edge cases are fully specified by feature 001; this feature does not redefine them.
- "Upstream" refers to the public `terraform-provider-tailscale` repository at the commit from which this repository was forked. References to "upstream conventions" mean conventions visible in that codebase at the fork point.
- The Terraform Registry's documented file-layout requirements for provider releases are stable enough to target at v0.1 without Registry submission, and a future v0.2 submission will not require re-shaping the v0.1 artifacts beyond a Registry-side metadata step.
- Operators consuming v0.1 are willing to install via a dev override or unzip a GitHub Release; lack of Registry availability is not a v0.1 blocker.
- The GPG signing key used for releases is held by the project maintainer(s) and is configured as a CI secret before the first tagged release.
- The "remediation findings from 001's `/speckit.analyze`" referenced in the input description are recorded in the planning artifacts of feature 001 and will be lifted into this feature's backlog during planning.

### Out of Scope

- All upstream Tailscale-provider resources unrelated to membership (DNS, ACLs, devices, keys, webhooks, posture integrations, contacts, settings, AWS external IDs, OAuth clients, and any others present in upstream). These are removed from the codebase, not retained as inactive code.
- The memberships (list) data source, the single-user data source, and any bulk-invite resource. Deferred to v0.2+.
- Submission to the Terraform Registry. Deferred to v0.2 once a stable release line exists.
- Any automated upstream-sync mechanism (cherry-pick automation, rebase tooling, drift detection against upstream). This is a hard fork.
- Any dependency on the upstream `terraform-provider-tailscale` Go module. The only retained upstream Go dependency is `tailscale.com/client/tailscale/v2`.
- Resolving the remediation findings from feature 001's `/speckit.analyze`. They are tracked in this feature's backlog but are explicitly NOT v0.1 blockers.
- Any change to the membership resource's behavior beyond the auth-routing correctness fix (FR-006, FR-007). Behavioral changes are out of scope for v0.1.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator who has never used this provider before can install v0.1, configure it, and successfully apply a single membership resource against a test tailnet within 15 minutes of opening the README, using only the README and the docs pages.
- **SC-002**: For each of the three supported auth modes (API key, OAuth, Federated Identity), a complete create/read/update/destroy cycle on a membership succeeds end-to-end with zero auth-related fallbacks. (Specifically: zero API-key Basic auth fallbacks observed when OAuth or Federated Identity is the configured mode.)
- **SC-003**: The provider's published schema contains exactly one managed resource and zero data sources. Any change that adds a second resource or any data source MUST be a deliberate, documented v0.2+ change.
- **SC-004**: A Git tag matching the release pattern produces a complete, GPG-signed, Registry-shaped release artifact set on the corresponding GitHub Release with no manual post-processing required.
- **SC-005**: An operator following the migration guide can move from the upstream-derived prototype membership resource to this provider with `terraform plan` reporting zero diffs after migration, on at least one documented worked example.
- **SC-006**: The repository's license/attribution posture (MIT `LICENSE` + upstream-attributing `NOTICE`) passes a routine open-source-license review with no findings requiring code or distribution changes.
- **SC-007**: Every argument and attribute exposed by the provider's schema is documented in `docs/`, and every argument and attribute documented in `docs/` is present in the schema. Coverage is 100% in both directions.
