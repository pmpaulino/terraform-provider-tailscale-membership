terraform {
  required_providers {
    tailscale-membership = {
      source  = "pmpaulino/tailscale-membership"
      version = "~> 0.1"
    }
  }
}

# The `provider = tailscale-membership` attribute is required on every
# membership resource block. The resource type prefix `tailscale_*` would
# otherwise default-bind to the upstream `tailscale/tailscale` provider when
# both providers are loaded in the same module.
resource "tailscale_membership_tailnet_membership" "member" {
  provider   = tailscale-membership
  login_name = "alice@example.com"
  role       = "member"
}
