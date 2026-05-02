---
name: Bug report
about: Create a report to help us improve
title: ''
labels: bug
assignees: ''
---

> **Scope reminder:** This provider manages only Tailscale tailnet membership (`tailscale_membership_tailnet_membership`). Bugs with other Tailscale resources belong in [tailscale/terraform-provider-tailscale](https://github.com/tailscale/terraform-provider-tailscale/issues). Bugs with the Tailscale API itself belong in [tailscale/tailscale](https://github.com/tailscale/tailscale).

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Minimal Terraform configuration and steps to reproduce:

```hcl
# paste your config here (redact any secrets)
```

**Expected behaviour**
A clear and concise description of what you expected to happen.

**Actual behaviour**
What happened instead. Paste the full `terraform` output if relevant.

**Versions**
- OS: [e.g. macOS 14, Ubuntu 24.04]
- Terraform version: [e.g. 1.9.0]
- Provider version: [e.g. 1.0.0]
- Auth mode: [API key / OAuth / Federated Identity]

**Additional context**
Add any other context here.
