# Quickstart: Standalone Tailscale Membership Provider (Phase 1)

**Feature**: 002-standalone-membership-provider  
**Spec**: [spec.md](./spec.md)

This document doubles as the operator-facing quickstart and the migration guide required by FR-021. Sections 1–3 cover first-time install and use; section 4 covers migration from the upstream-derived prototype; section 5 covers verifying the GPG signature on a downloaded release.

---

## 1. Install

### Option A: Dev override (recommended for evaluation)

Build the provider locally:

```bash
git clone https://github.com/pmpaulino/terraform-provider-tailscale-membership
cd terraform-provider-tailscale-membership
go install .
```

`go install` places the binary at `$(go env GOBIN)/terraform-provider-tailscale-membership` (or `$GOPATH/bin/...` if `GOBIN` is unset).

Add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "pmpaulino/tailscale-membership" = "/absolute/path/to/$GOBIN"
  }
  direct {}
}
```

Now `terraform plan`/`apply` resolve the provider source `pmpaulino/tailscale-membership` to the locally built binary. (No `terraform init` is required for dev-overridden providers.)

### Option B: Tagged GitHub Release (recommended for production until v0.2 ships to the Registry)

1. Download the appropriate platform zip from the GitHub Releases page (e.g. `terraform-provider-tailscale-membership_1.0.0_linux_amd64.zip`).
2. Verify the GPG signature and SHA256 (see section 5).
3. Unzip into Terraform's plugin cache directory at the path Terraform expects for filesystem mirrors:

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/pmpaulino/tailscale-membership/1.0.0/linux_amd64
unzip terraform-provider-tailscale-membership_1.0.0_linux_amd64.zip -d \
  ~/.terraform.d/plugins/registry.terraform.io/pmpaulino/tailscale-membership/1.0.0/linux_amd64
```

`terraform init` will discover the plugin from this filesystem location.

---

## 2. Configure the provider

This provider involves three distinct identifiers that obey different syntactic rules:

| What | Value | Why |
|---|---|---|
| Provider source address | `pmpaulino/tailscale-membership` | Registry/source address — any character valid in a URL path. |
| Terraform local name (in `required_providers`) | `tailscale-membership` | Terraform local names allow letters/digits/dashes; underscores are forbidden. |
| Resource type (in HCL) | `tailscale_membership_tailnet_membership` | HCL resource identifiers cannot contain dashes; underscores are required. |

Because the resource type starts with `tailscale_`, Terraform would default-bind it to the upstream `tailscale/tailscale` provider when both are loaded. **Every membership resource block MUST carry `provider = tailscale-membership`** to override the implicit binding (see §3). Declare the alias in `required_providers`:

```hcl
terraform {
  required_providers {
    tailscale-membership = {
      source  = "pmpaulino/tailscale-membership"
      version = "~> 1.0"
    }
  }
}
```

> **Spec history note**: an earlier version of this spec proposed local name `tailscale_membership` (with an underscore). Terraform's CLI rejects underscores in provider local names — `must contain only letters, digits, and dashes` — so the canonical local name is `tailscale-membership` (dashed). See `spec.md` Q3 amendment.

### 2a. API-key auth

```hcl
provider "tailscale-membership" {
  api_key = "tskey-api-..."
  tailnet = "example.com"
}
```

Or via environment: `export TAILSCALE_API_KEY=tskey-api-...` and `export TAILSCALE_TAILNET=example.com`, then leave the block empty.

### 2b. OAuth client credentials (recommended)

```hcl
provider "tailscale-membership" {
  oauth_client_id     = var.tailscale_oauth_client_id
  oauth_client_secret = var.tailscale_oauth_client_secret
  scopes              = ["users:write", "user_invites:write"]
  tailnet             = "example.com"
}
```

The OAuth client MUST have the `UserInvites` and `users` scopes (see [`contracts/api-operations.md`](./contracts/api-operations.md)).

### 2c. Federated Identity (workload identity federation)

```hcl
provider "tailscale-membership" {
  oauth_client_id = var.tailscale_oauth_client_id
  identity_token  = data.aws_iam_openid_connect_provider.this.id_token  # or any other JWT source
  tailnet         = "example.com"
}
```

---

## 3. Manage memberships

Every membership resource block MUST carry `provider = tailscale-membership` because the resource type prefix `tailscale_*` would otherwise default-bind to the upstream `tailscale/tailscale` provider when both are loaded.

```hcl
resource "tailscale_membership_tailnet_membership" "alice" {
  provider   = tailscale-membership
  login_name = "alice@example.com"
  role       = "member"
}

resource "tailscale_membership_tailnet_membership" "bob_admin" {
  provider   = tailscale-membership
  login_name = "bob@example.com"
  role       = "admin"
}

# Disable temporarily without losing the membership record:
resource "tailscale_membership_tailnet_membership" "carol_paused" {
  provider   = tailscale-membership
  login_name = "carol@example.com"
  role       = "member"
  suspended  = true
}

# Downgrade to member instead of removing on destroy:
resource "tailscale_membership_tailnet_membership" "dave_admin" {
  provider             = tailscale-membership
  login_name           = "dave@example.com"
  role                 = "admin"
  downgrade_on_destroy = true
}
```

`terraform plan` then `terraform apply`. For a brand-new identity, the apply creates a Tailscale user invite (visible in the Tailscale admin console); the invitee receives an email and accepts. `terraform refresh` (or any subsequent `plan`) reflects the resulting `state` transition from `pending` to `active`.

### Import an existing membership

```bash
terraform import 'tailscale_membership_tailnet_membership.alice' 'example.com:alice@example.com'
```

Import ID format: `tailnet:login_name`.

---

## 4. Migration from the upstream-derived prototype

If you are currently managing tailnet memberships via the prototype `tailscale_tailnet_membership` resource (i.e. the resource defined in feature 001 of this repository, before the v1.0 fork), follow these steps. **Tailnet state on the Tailscale side is unaffected**; only the local Terraform state and HCL change.

### 4.1. Update `required_providers`

Change:

```hcl
terraform {
  required_providers {
    tailscale = {
      source  = "tailscale/tailscale"
      version = "..."
    }
  }
}
```

to:

```hcl
terraform {
  required_providers {
    tailscale-membership = {
      source  = "pmpaulino/tailscale-membership"
      version = "~> 1.0"
    }
  }
}
```

### 4.2. Rename the provider block

Change `provider "tailscale" { ... }` to `provider "tailscale-membership" { ... }`. Argument names and values are unchanged (the configuration schema matches upstream per FR-004).

> If you also want to keep the upstream `tailscale/tailscale` provider loaded for non-membership resources, leave its `required_providers` entry and `provider "tailscale" { ... }` block in place; the two providers coexist.

### 4.3. Rename every membership resource and add the `provider =` attribute

For each membership resource in your configuration, change:

```hcl
resource "tailscale_tailnet_membership" "alice" {
  login_name = "alice@example.com"
}
```

to:

```hcl
resource "tailscale_membership_tailnet_membership" "alice" {
  provider   = tailscale-membership
  login_name = "alice@example.com"
}
```

Two changes per block: (1) rename the resource type from `tailscale_tailnet_membership` to `tailscale_membership_tailnet_membership`, and (2) add `provider = tailscale-membership`. The resource type prefix `tailscale_*` would default-bind to the upstream `tailscale/tailscale` provider, so the explicit `provider =` override is required even if upstream is not loaded — Terraform's static validation looks for a matching `required_providers` entry. Argument values are unchanged.

### 4.4. Move state

For each renamed resource, run:

```bash
terraform state mv \
  'tailscale_tailnet_membership.alice' \
  'tailscale_membership_tailnet_membership.alice'
```

For resources nested in modules, qualify the addresses appropriately (e.g. `module.team.tailscale_tailnet_membership.alice`).

### 4.5. Remove the old provider from state

After all resources are moved, the upstream-style provider has no remaining state references. Remove it:

```bash
terraform state replace-provider \
  'registry.terraform.io/tailscale/tailscale' \
  'registry.terraform.io/pmpaulino/tailscale-membership'
```

(This is only necessary if you previously initialized with the upstream provider source. If you forked this repository fresh and never published to a registry source, you can skip step 4.5.)

### 4.6. Verify

```bash
terraform init -upgrade
terraform plan
```

Expected result: **`No changes. Your infrastructure matches the configuration.`**

If `terraform plan` shows differences, the state move did not preserve the resource ID or an attribute default has drifted; restore from your previous state file and re-attempt step-by-step. Open an issue with the `migration` label on the GitHub repository if the issue reproduces.

### 4.7. Note: source-address drift

If an operator's HCL still references `tailscale/tailscale` (the upstream Registry source), Terraform itself will report an "unknown provider" or "provider source mismatch" error. That is a Terraform-level diagnostic, not something this provider can intercept. The fix is step 4.1.

---

## 5. Verify a downloaded release

Each tagged GitHub Release (production tag `vX.Y.Z` or pre-release `vX.Y.Z-rc.N`) ships with:

- Per-platform zip archives.
- A `SHA256SUMS` file covering all archives.
- A detached GPG signature `SHA256SUMS.sig`.
- A Registry-shape manifest file.

The project's GPG public key is published in the README (`https://github.com/pmpaulino/terraform-provider-tailscale-membership#release-signing-key`).

To verify:

```bash
# Import the public key once
gpg --recv-keys <PROJECT_KEY_ID>   # or download from the README link

# Verify the signature on the checksums file
gpg --verify SHA256SUMS.sig SHA256SUMS

# Verify the SHA256 of your downloaded archive
shasum -a 256 -c SHA256SUMS --ignore-missing
```

A successful signature verification establishes that the `SHA256SUMS` file came from the project; a successful checksum match establishes that your downloaded archive is the one covered by `SHA256SUMS`. Together they satisfy SC-004's "GPG-signed, Registry-shaped" requirement end-to-end at the consumer's end.
