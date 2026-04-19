# Contributing Guidelines

This document contains tips for contributing to `pmpaulino/terraform-provider-tailscale-membership`. These are suggestions and advice, not hard and fast rules.

> **Scope reminder.** This is a hard fork of [`tailscale/terraform-provider-tailscale`](https://github.com/tailscale/terraform-provider-tailscale) (MIT-licensed; see [`NOTICE`](./NOTICE)). It exposes **only** Tailscale tailnet-membership management — the membership resource, its data sources, and the provider plumbing required to authenticate the Tailscale Control API. Anything outside that surface (DNS, ACLs, devices, keys, webhooks, posture, etc.) is **out of scope** and lives upstream.
>
> If you want a feature that already exists in the upstream provider, use that provider directly — both can coexist in the same Terraform module (see the README's [Naming conventions](./README.md#naming-conventions) section).

## Raising issues

Please use the [issue tracker](https://github.com/pmpaulino/terraform-provider-tailscale-membership/issues/new/choose) on this repository.

If your issue concerns Tailscale Control API behavior rather than the Terraform provider's wiring around it, raise it upstream:

- Tailscale API limitations: [`tailscale/tailscale`](https://github.com/tailscale/tailscale) ([API reference](https://github.com/tailscale/tailscale/blob/main/api.md)).
- Provider features unrelated to membership management: [`tailscale/terraform-provider-tailscale`](https://github.com/tailscale/terraform-provider-tailscale/issues).

## Opening pull requests

Please link a PR to an issue (or open an issue first) so the scope discussion lives somewhere referenceable.

PRs that expand the surface area beyond membership management will be closed — they belong upstream. PRs that improve the membership resource, its tests, its documentation, the release pipeline, or the provider plumbing are welcome.

## Making changes

- Go toolchain: `go.mod` declares the minimum version; install something at or above that.
- Build: `make build` (or `go build ./...`).
- Unit tests: `go test ./...`.
- Acceptance tests (`TF_ACC=1`) hit the real Tailscale Control API and will mutate state on the tailnet they target. **Use a dedicated test tailnet** — do not run them against a tailnet you care about.
- Lint: `golangci-lint run ./...`.
- Docs/schema coverage: `./scripts/check-docs-coverage.sh` (CI runs this — it must stay green).

## Releases

Tagged releases (`vMAJOR.MINOR.PATCH`) trigger `.github/workflows/release.yml`, which uses GoReleaser to build the FR-014 11-platform matrix, GPG-sign the `SHA256SUMS` file, and publish a Terraform-Registry-shaped GitHub Release. The release-signing key fingerprint is published in [`KEYS`](./KEYS) and the [README's "Verifying releases" section](./README.md#verifying-releases). Pre-release tags (anything containing a hyphen, e.g. `v0.2.0-rc.1`) auto-promote to the GitHub "pre-release" UI.
