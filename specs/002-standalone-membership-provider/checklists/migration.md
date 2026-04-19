# Migration Walkthrough Checklist

**Feature**: 002-standalone-membership-provider
**Owner**: Pre-tag dress rehearsal (T070)
**Reference**: [`quickstart.md` §4](../quickstart.md#4-migration-from-the-upstream-derived-prototype)

This checklist tracks the manual `terraform state mv` walkthrough that verifies the FR-021 migration path against a real (dev) tailnet before tagging `v0.1.0`. It exists because US3's acceptance scenario is verified manually (no automated test); see [`tasks.md` T050](../tasks.md).

## Pre-conditions

- [ ] A dev tailnet is available with at least one membership currently managed by the prototype `tailscale_tailnet_membership` resource (feature 001 of this repo, pre-fork).
- [ ] The Terraform state file for that membership is backed up (e.g. `cp terraform.tfstate terraform.tfstate.pre-migration-backup`).
- [ ] The current `terraform plan` exit code against the prototype is `0` with `No changes.` (confirms a clean baseline).
- [ ] The v0.1 provider binary is installed via either dev override or a release artifact (see [`quickstart.md` §1](../quickstart.md#1-install)).

## Walkthrough — execute in order

- [ ] **§4.1**: `required_providers` updated from `tailscale/tailscale` to `pmpaulino/tailscale-membership` with local name `tailscale-membership` (dashed — Terraform rejects underscores in local names).
- [ ] **§4.2**: Provider block renamed from `provider "tailscale" { ... }` to `provider "tailscale-membership" { ... }`. Argument values (api_key / oauth_client_id / oauth_client_secret / identity_token / tailnet / base_url / scopes) copied verbatim.
- [ ] **§4.3**: Every `resource "tailscale_tailnet_membership" ...` block renamed to `resource "tailscale_membership_tailnet_membership" ...` AND given an explicit `provider = tailscale-membership` attribute (required because the resource type prefix `tailscale_*` would otherwise default-bind to the upstream `tailscale` provider). Argument values unchanged.
- [ ] **§4.4**: For each membership resource, `terraform state mv 'tailscale_tailnet_membership.<name>' 'tailscale_membership_tailnet_membership.<name>'` executed and reported `Move successful`. Module-qualified addresses (e.g. `module.team.tailscale_tailnet_membership.alice`) handled.
- [ ] **§4.5**: `terraform state replace-provider 'registry.terraform.io/tailscale/tailscale' 'registry.terraform.io/pmpaulino/tailscale-membership'` executed (or skipped if previously dev-overridden).
- [ ] **§4.6**: `terraform init -upgrade` succeeds.
- [ ] **§4.6**: `terraform plan` reports **exactly** `No changes. Your infrastructure matches the configuration.` — no creates, no destroys, no in-place updates. SC-005 satisfied.

## Post-conditions

- [ ] No Tailscale API call was made by the provider during the migration (verified by checking the test tailnet's audit log if available).
- [ ] All membership IDs in state match their pre-migration values (verified via `terraform state show <addr> | grep '^id'` against the backup).
- [ ] If `terraform plan` reports any diff, the migration is **rolled back** (restore `terraform.tfstate.pre-migration-backup`) and an issue with the `migration` label is opened on the GitHub repo before retrying.

## Sign-off

- [ ] Date executed: _________
- [ ] Operator: _________
- [ ] Tailnet: _________
- [ ] Number of memberships migrated: _________
- [ ] All checkboxes above checked → SC-005 verified → US3 ready for v0.1.0.
