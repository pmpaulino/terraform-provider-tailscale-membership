#!/bin/sh
# Import an existing membership by tailnet and login name (email).
# Replace TAILNET and LOGIN_NAME with your tailnet ID and the user's email.
#
# Example: if your tailnet is "example.com" and the user is "alice@example.com":
#   terraform import 'tailscale_membership_tailnet_membership.member' 'example.com:alice@example.com'

terraform import 'tailscale_membership_tailnet_membership.member' 'TAILNET:LOGIN_NAME'
