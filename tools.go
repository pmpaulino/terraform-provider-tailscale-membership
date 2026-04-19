// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

//go:build tools

package tools

import (
	// goimports is invoked by `make fmt` and the CI `format` job.
	// terraform-plugin-docs (tfplugindocs) was removed in Phase 7 / feature 002:
	// docs are hand-authored under docs/. See main.go's package-doc comment for
	// the rationale (Terraform local-name dash-vs-underscore mismatch).
	_ "golang.org/x/tools/cmd/goimports"
)
