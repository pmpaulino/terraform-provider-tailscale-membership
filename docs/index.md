---
page_title: "Tailscale Membership Provider"
description: |-
  pmpaulino/tailscale-membership is a single-purpose hard fork of tailscale/terraform-provider-tailscale that exposes only Tailscale tailnet membership management.
---

# Tailscale Membership Provider

`pmpaulino/tailscale-membership` is a single-purpose, hard-fork derivative of [`tailscale/terraform-provider-tailscale`](https://github.com/tailscale/terraform-provider-tailscale) that exposes **only** Tailscale tailnet membership management. If you also need to manage devices, ACLs, DNS, keys, webhooks, etc. via Terraform, use the upstream provider — the two providers are designed to coexist in the same Terraform module.

For migration from the upstream-derived prototype, see the [migration guide](https://github.com/pmpaulino/terraform-provider-tailscale-membership/blob/main/specs/002-standalone-membership-provider/quickstart.md#4-migration-from-the-upstream-derived-prototype).

For the release-signing GPG public key (verify your downloaded release), see [`KEYS`](https://github.com/pmpaulino/terraform-provider-tailscale-membership/blob/main/KEYS) in the repository root.

## Example Usage

The Terraform local name for this provider is `tailscale-membership` (with a dash — Terraform local names allow letters/digits/dashes but not underscores). Declare it via `required_providers` so the source address is unambiguous:

```terraform
terraform {
  required_providers {
    tailscale-membership = {
      source  = "pmpaulino/tailscale-membership"
      version = "~> 0.1"
    }
  }
}

provider "tailscale-membership" {
  oauth_client_id     = "my_client_id"
  oauth_client_secret = "my_client_secret"
  tailnet             = "example.com"
}
```

Resource types are registered as `tailscale_membership_*` (underscores — HCL resource identifiers cannot contain dashes). Because the `tailscale_*` prefix would default-bind to the upstream `tailscale` provider when both are loaded, every membership resource block MUST carry `provider = tailscale-membership`. See [`tailscale_membership_tailnet_membership`](./resources/tailnet_membership.md) for the resource page.

## Coexisting with the upstream `tailscale/tailscale` provider

```terraform
terraform {
  required_providers {
    tailscale = {
      source  = "tailscale/tailscale"
      version = "~> 0.16"
    }
    tailscale-membership = {
      source  = "pmpaulino/tailscale-membership"
      version = "~> 0.1"
    }
  }
}

provider "tailscale" {
  oauth_client_id     = var.tailscale_oauth_client_id
  oauth_client_secret = var.tailscale_oauth_client_secret
}

provider "tailscale-membership" {
  oauth_client_id     = var.tailscale_oauth_client_id
  oauth_client_secret = var.tailscale_oauth_client_secret
}

resource "tailscale_acl" "main" {
  acl = file("${path.module}/acl.hujson")
}

resource "tailscale_membership_tailnet_membership" "alice" {
  provider   = tailscale-membership
  login_name = "alice@example.com"
  role       = "member"
}
```

## Authentication

Three authentication modes are supported. Exactly one must be configured; combinations are rejected at provider-configuration time with a "conflicting" diagnostic. Using a [trust credential](https://tailscale.com/kb/1623/trust-credentials) (an OAuth client or federated identity) is recommended whenever possible: trust credentials can have granular access scopes applied to them, whereas API keys cannot.

### OAuth clients

[OAuth clients](https://tailscale.com/kb/1215/oauth-clients) authenticate via `oauth_client_id` + `oauth_client_secret`. The OAuth client MUST have the `UserInvites` and `users` scopes for full membership management.

```terraform
provider "tailscale-membership" {
  oauth_client_id     = "my_client_id"
  oauth_client_secret = "my_client_secret"
  scopes              = ["users:write", "user_invites:write"]
  tailnet             = "example.com"
}
```

### Federated identities

[Workload identity federation](https://tailscale.com/kb/1581/workload-identity-federation) authenticates via `oauth_client_id` + `identity_token` (a JWT from a compatible issuer such as AWS, GCP, or GitHub Actions OIDC).

```terraform
provider "tailscale-membership" {
  oauth_client_id = "my_client_id"
  identity_token  = "my_identity_token"
  tailnet         = "example.com"
}
```

### API keys

[API keys](https://tailscale.com/kb/1101/api#authentication) authenticate via `api_key`. A user-owned API key may be required to create user invites; see the [Tailscale invite docs](https://tailscale.com/kb/1371/invite-users).

```terraform
provider "tailscale-membership" {
  api_key = "my_api_key"
  tailnet = "example.com"
}
```

## Schema

### Optional

- `api_key` (String, Sensitive) The API key to use for authenticating requests to the API. Can be set via the `TAILSCALE_API_KEY` environment variable. Conflicts with `oauth_client_id` and `oauth_client_secret`.
- `base_url` (String) The base URL of the Tailscale API. Defaults to `https://api.tailscale.com`. Can be set via the `TAILSCALE_BASE_URL` environment variable.
- `identity_token` (String, Sensitive) The JWT identity token to exchange for a Tailscale API token when using a federated identity. Can be set via the `TAILSCALE_IDENTITY_TOKEN` environment variable. Conflicts with `api_key` and `oauth_client_secret`.
- `oauth_client_id` (String) The OAuth application or federated identity's ID when using OAuth client credentials or workload identity federation. Can be set via the `TAILSCALE_OAUTH_CLIENT_ID` environment variable. Either `oauth_client_secret` or `identity_token` must be set alongside `oauth_client_id`. Conflicts with `api_key`.
- `oauth_client_secret` (String, Sensitive) The OAuth application's secret when using OAuth client credentials. Can be set via the `TAILSCALE_OAUTH_CLIENT_SECRET` environment variable. Conflicts with `api_key` and `identity_token`.
- `scopes` (List of String) The OAuth 2.0 scopes to request when generating the access token using the supplied OAuth client credentials. See <https://tailscale.com/kb/1623/trust-credentials#scopes> for available scopes. Only valid when both `oauth_client_id` and `oauth_client_secret` are set.
- `tailnet` (String) The tailnet ID. Tailnets created before October 2025 can still use the legacy ID, but the Tailnet ID is the preferred identifier. Can be set via the `TAILSCALE_TAILNET` environment variable. Default is the tailnet that owns the API credentials passed to the provider.
- `user_agent` (String) `User-Agent` header for API requests.
