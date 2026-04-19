// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

// Documentation is hand-authored under docs/ (not generated). The standard
// `tfplugindocs generate` workflow does not work for this provider because
// the Terraform local provider name (`tailscale-membership`, with a dash —
// per Terraform's letters/digits/dashes-only rule) cannot be the same as the
// resource-key prefix (`tailscale_membership_*`, with an underscore — per
// HCL identifier rules). tfplugindocs's `--provider-name` flag is used both
// for resource-key prefix derivation AND for generating a provider.tf that
// it validates with the Terraform CLI, so no single value satisfies both
// constraints. Bidirectional schema↔docs coverage is enforced separately by
// scripts/check-docs-coverage.sh; see specs/002-standalone-membership-provider/
// tasks.md T056.
package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/pmpaulino/terraform-provider-tailscale-membership/tailscale"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return tailscale.Provider()
		},
	})
}
