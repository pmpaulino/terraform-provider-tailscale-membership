# Security Policy

## Supported Versions

Only the latest release is actively supported with security fixes.

| Version | Supported |
|---------|-----------|
| 1.x (latest) | ✅ |
| < 1.0 | ❌ |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Report vulnerabilities privately via [GitHub Security Advisories](https://github.com/pmpaulino/terraform-provider-tailscale-membership/security/advisories/new).

Include:
- A description of the vulnerability and its impact
- Steps to reproduce
- Any suggested mitigations

You can expect an acknowledgement within 7 days and a fix or disclosure timeline within 30 days.

## Scope

This provider is a thin Terraform wrapper around the [Tailscale Control API](https://tailscale.com/kb/1101/api). Vulnerabilities in the Tailscale API itself or the Tailscale client library (`tailscale.com/client/tailscale/v2`) should be reported to [Tailscale's security team](https://tailscale.com/security) directly.
