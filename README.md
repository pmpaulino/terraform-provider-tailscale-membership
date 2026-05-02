# terraform-provider-tailscale-membership

A single-purpose Terraform provider, **`pmpaulino/tailscale-membership`**, that exposes only Tailscale tailnet membership management.

This is a hard-fork derivative of [`tailscale/terraform-provider-tailscale`](https://github.com/tailscale/terraform-provider-tailscale) (MIT-licensed; see [`LICENSE`](./LICENSE) and [`NOTICE`](./NOTICE)). All upstream resources unrelated to membership (DNS, ACLs, devices, keys, webhooks, posture integrations, contacts, settings, AWS external IDs, OAuth clients, etc.) have been removed; the v1.0 surface is one resource, [`tailscale_membership_tailnet_membership`](./docs/resources/tailnet_membership.md).

If you also need to manage devices, ACLs, DNS, etc., use the upstream provider — the two are designed to coexist in the same Terraform module.

## Status

- **v1.0 (current)**: GitHub-Releases-only; not yet on the Terraform Registry. Install via dev override or filesystem mirror (see [Install](#install) below).
- **v1.1 (planned)**: Terraform Registry submission and additional read-only data sources.

## Install

### Option A — dev override (recommended for evaluation)

```bash
git clone https://github.com/pmpaulino/terraform-provider-tailscale-membership
cd terraform-provider-tailscale-membership
go install .
```

Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "pmpaulino/tailscale-membership" = "/absolute/path/to/$GOBIN"
  }
  direct {}
}
```

### Option B — tagged GitHub Release (recommended for production until v1.1 ships to the Registry)

1. Download the appropriate platform zip from the [GitHub Releases](https://github.com/pmpaulino/terraform-provider-tailscale-membership/releases) page.
2. **Verify the GPG signature and SHA256** (see [Verifying releases](#verifying-releases) below).
3. Unzip into Terraform's filesystem-mirror plugin path:

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/pmpaulino/tailscale-membership/1.0.0/linux_amd64
unzip terraform-provider-tailscale-membership_1.0.0_linux_amd64.zip -d \
  ~/.terraform.d/plugins/registry.terraform.io/pmpaulino/tailscale-membership/1.0.0/linux_amd64
```

`terraform init` will discover the plugin from this path.

The full operator quickstart, including all three authentication modes and worked examples, is in [`specs/002-standalone-membership-provider/quickstart.md`](./specs/002-standalone-membership-provider/quickstart.md).

## Naming conventions (important)

This provider's source address contains a dash, but Terraform's local-name and HCL identifier rules disagree about which separator to use. Operators need to know the three identifiers involved:

| What | Value | Why |
|---|---|---|
| Provider source address | `pmpaulino/tailscale-membership` | Registry/source address — any character valid in a URL path. |
| Terraform local name (in `required_providers`) | `tailscale-membership` | Terraform local names allow letters/digits/dashes; underscores are forbidden. |
| Resource type (in HCL) | `tailscale_membership_tailnet_membership` | HCL resource identifiers cannot contain dashes; underscores are required. |

Because the resource type starts with `tailscale_`, Terraform would default-bind it to the upstream `tailscale/tailscale` provider when both are loaded in the same module. **Every membership resource block MUST carry `provider = tailscale-membership`** to override that implicit binding:

```hcl
terraform {
  required_providers {
    tailscale = {
      source  = "tailscale/tailscale"
      version = "~> 0.16"
    }
    tailscale-membership = {
      source  = "pmpaulino/tailscale-membership"
      version = "~> 1.0"
    }
  }
}

provider "tailscale" { ... }
provider "tailscale-membership" { ... }

resource "tailscale_acl" "main" { ... }                          # bound to upstream implicitly

resource "tailscale_membership_tailnet_membership" "alice" {
  provider   = tailscale-membership                              # required override
  login_name = "alice@example.com"
  role       = "member"
}
```

See [`docs/index.md`](./docs/index.md) for the full provider page.

## Migration from the upstream-derived prototype

If you previously managed memberships via the prototype `tailscale_tailnet_membership` resource (feature 001 of this repository, before the v0.1 fork), follow the [migration guide in `quickstart.md` §4](./specs/002-standalone-membership-provider/quickstart.md#4-migration-from-the-upstream-derived-prototype). It walks through:

1. Updating `required_providers`.
2. Renaming the provider block.
3. Renaming every membership resource type (`tailscale_tailnet_membership` → `tailscale_membership_tailnet_membership`) and adding the `provider = tailscale-membership` attribute.
4. `terraform state mv` for each resource.
5. `terraform state replace-provider` to drop the old source address.
6. Verification: `terraform plan` should report `No changes.`

## Verifying releases

Each tagged release is signed with the project's GPG key. The public half of that key is published in [`KEYS`](./KEYS) at the repository root.

**Expected fingerprint** (cross-check this against the header inside `KEYS` before importing):

```text
1AE3 E49A 1CCC 2805 A321 C991 21BE A434 67F2 A13D
```

```bash
# Import the project's release-signing key once
curl -fsSL https://raw.githubusercontent.com/pmpaulino/terraform-provider-tailscale-membership/main/KEYS \
  | gpg --import

# Verify the SHA256SUMS signature on a downloaded release
gpg --verify terraform-provider-tailscale-membership_1.0.0_SHA256SUMS.sig \
             terraform-provider-tailscale-membership_1.0.0_SHA256SUMS

# Verify the SHA256 of your downloaded archive
shasum -a 256 -c terraform-provider-tailscale-membership_1.0.0_SHA256SUMS --ignore-missing
```

A successful signature verification establishes the `SHA256SUMS` file came from the project; a successful checksum match establishes your downloaded archive matches what was signed.

## Documentation

- Provider page: [`docs/index.md`](./docs/index.md)
- Resource page: [`docs/resources/tailnet_membership.md`](./docs/resources/tailnet_membership.md)
- Operator quickstart and migration guide: [`specs/002-standalone-membership-provider/quickstart.md`](./specs/002-standalone-membership-provider/quickstart.md)
- v0.1 specification, design, and tasks: [`specs/002-standalone-membership-provider/`](./specs/002-standalone-membership-provider/)

## Local provider development

For changes to the provider itself, see the [Terraform Plugin debugging docs](https://developer.hashicorp.com/terraform/plugin/debugging) and use the dev override flow described in [Install — Option A](#option-a--dev-override-recommended-for-evaluation) above. Run `make build` to build, `go test ./...` to run unit tests, and `make testacc` to run acceptance tests against a real Tailscale tailnet (requires `TF_ACC=1` plus `TAILSCALE_*` credentials).

## License & attribution

This project is MIT-licensed; see [`LICENSE`](./LICENSE). It is a hard fork of [`tailscale/terraform-provider-tailscale`](https://github.com/tailscale/terraform-provider-tailscale); see [`NOTICE`](./NOTICE) for the upstream attribution. There is no ongoing upstream sync.
