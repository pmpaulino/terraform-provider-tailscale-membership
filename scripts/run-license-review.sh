#!/usr/bin/env bash
#
# run-license-review.sh
#
# Phase 8 / T066 helper: produces the dependency-license report archived
# alongside the v0.1.0 release notes. Wraps `go list -m -json all` →
# scripts/license-review.go (build-tag-fenced), and writes plain-text
# output to stdout.
#
# Exit codes:
#   0 — clean: no GPL/AGPL/SSPL/CC-licensed dependencies
#   2 — at least one license-incompatible dependency detected (FR-024)
#
# go-licenses (v1.6.0) is intentionally NOT used here because it crashes
# on Go >= 1.22 when stdlib enumeration encounters the modular toolchain
# layout. See scripts/license-review.go's package-doc comment.
#
# Usage:
#   ./scripts/run-license-review.sh                # print to stdout
#   ./scripts/run-license-review.sh > LICENSE-REVIEW-v0.1.0.txt
#   ./scripts/run-license-review.sh | grep -E '(UNKNOWN|MISSING|FAIL)'
#

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

go list -m -json all | go run -tags licensereview scripts/license-review.go
