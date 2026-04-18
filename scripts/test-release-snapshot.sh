#!/usr/bin/env bash
#
# test-release-snapshot.sh
#
# Runs goreleaser in snapshot mode (no GPG, no publish, no real tag) and
# asserts that the produced dist/ matches the FR-014 release shape:
#
#   1. Exactly 11 zip archives, named per the FR-014 OS/arch matrix
#      (linux × {amd64,arm64,386,arm}; darwin × {amd64,arm64};
#      windows × {amd64,386}; freebsd × {amd64,arm,386}).
#   2. A SHA256SUMS file containing one line per archive (= 11 lines).
#   3. A Registry-shape manifest file (terraform-registry-manifest.json) that
#      parses as valid JSON, has top-level version: 1 (manifest schema
#      version per HashiCorp's Terraform Registry contract), and contains a
#      metadata.protocol_versions array including "5.0" (Plugin SDK v2's
#      protocol -- Registry rejects manifests missing this shape).
#
# This pins FR-014 to a verifiable build-time check rather than a paper
# requirement. Wired into .github/workflows/ci.yml as an informational job.
#
# Note on T053's spec text: T053 calls for "top-level version field equal to
# '5.0'", but the Terraform Registry's manifest schema uses version: 1 to
# describe the manifest format itself, with the actual Plugin Protocol
# versions inside metadata.protocol_versions. We use the Registry-correct
# shape (version: 1, metadata.protocol_versions: ["5.0"]); the spec text is
# corrected in tasks.md alongside this script.
#
# Exit codes:
#   0  all assertions passed
#   1  goreleaser failed
#   2  archive count, naming, or count of SHA256SUMS lines is wrong
#   3  manifest file is missing, malformed, or has wrong shape
#
# Requirements: goreleaser (>= v2.0), jq.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

PROJECT_NAME="terraform-provider-tailscale-membership"
EXPECTED_ARCHIVE_COUNT=11

# FR-014 OS/arch matrix as a sorted, newline-delimited list. Each line is
# "<os>_<arch>". We compare this set against the produced archives below.
EXPECTED_PLATFORMS=$(cat <<'EOF'
darwin_amd64
darwin_arm64
freebsd_386
freebsd_amd64
freebsd_arm
linux_386
linux_amd64
linux_arm
linux_arm64
windows_386
windows_amd64
EOF
)

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: required command not found: $1" >&2
    exit 1
  }
}

require_cmd goreleaser
require_cmd jq

echo "==> Running goreleaser snapshot (no publish, no sign)..."
rm -rf dist/
# --skip publish,sign keeps this runnable in CI without GPG / GitHub token.
goreleaser release --snapshot --clean --skip=publish,sign

# release.extra_files in .goreleaser.yml is only applied during the actual
# release-publish step (skipped in snapshot mode). Mimic what the real release
# would do by copying terraform-registry-manifest.json into dist/ with the
# Registry-expected filename. This lets the same script verify the manifest
# shape that consumers will actually see.
SNAPSHOT_VERSION=$(jq -r '.version' dist/metadata.json)
if [ -z "$SNAPSHOT_VERSION" ] || [ "$SNAPSHOT_VERSION" = "null" ]; then
  echo "ERROR: could not read .version from dist/metadata.json" >&2
  exit 1
fi
cp terraform-registry-manifest.json \
   "dist/${PROJECT_NAME}_${SNAPSHOT_VERSION}_manifest.json"

echo
echo "==> Verifying dist/ shape..."

# (1) Archive count.
ARCHIVES=$(find dist -maxdepth 1 -name '*.zip' | sort)
ARCHIVE_COUNT=$(echo "$ARCHIVES" | grep -c '.' || true)
if [ "$ARCHIVE_COUNT" -ne "$EXPECTED_ARCHIVE_COUNT" ]; then
  echo "FAIL: expected $EXPECTED_ARCHIVE_COUNT zip archives, got $ARCHIVE_COUNT" >&2
  echo "Archives found:" >&2
  echo "$ARCHIVES" >&2
  exit 2
fi
echo "  [OK] archive count = $EXPECTED_ARCHIVE_COUNT"

# (2) Archive naming: extract <os>_<arch> from each filename and diff against
#     EXPECTED_PLATFORMS.
ACTUAL_PLATFORMS=$(echo "$ARCHIVES" \
  | sed -E "s|^dist/${PROJECT_NAME}_[^_]+_([a-z0-9]+_[a-z0-9]+)\.zip\$|\1|" \
  | sort)

if ! diff <(echo "$EXPECTED_PLATFORMS") <(echo "$ACTUAL_PLATFORMS") >/dev/null; then
  echo "FAIL: archive os/arch set does not match FR-014 matrix:" >&2
  diff <(echo "$EXPECTED_PLATFORMS") <(echo "$ACTUAL_PLATFORMS") >&2 || true
  exit 2
fi
echo "  [OK] archive os/arch matrix matches FR-014"

# (3) SHA256SUMS file exists and has one line per archive.
SHASUMS=$(find dist -maxdepth 1 -name '*_SHA256SUMS' -type f)
if [ -z "$SHASUMS" ]; then
  echo "FAIL: no SHA256SUMS file found in dist/" >&2
  exit 2
fi
SHASUMS_COUNT=$(echo "$SHASUMS" | wc -l | tr -d ' ')
if [ "$SHASUMS_COUNT" -ne 1 ]; then
  echo "FAIL: expected exactly 1 SHA256SUMS file, got $SHASUMS_COUNT:" >&2
  echo "$SHASUMS" >&2
  exit 2
fi
SHASUMS_LINES=$(wc -l < "$SHASUMS" | tr -d ' ')
if [ "$SHASUMS_LINES" -ne "$EXPECTED_ARCHIVE_COUNT" ]; then
  echo "FAIL: SHA256SUMS has $SHASUMS_LINES lines, expected $EXPECTED_ARCHIVE_COUNT" >&2
  cat "$SHASUMS" >&2
  exit 2
fi
echo "  [OK] $SHASUMS = $EXPECTED_ARCHIVE_COUNT lines"

# (4) Registry-shape manifest exists and is well-formed.
MANIFEST=$(find dist -maxdepth 1 -name '*_manifest.json' -type f)
if [ -z "$MANIFEST" ]; then
  echo "FAIL: no Registry manifest (*_manifest.json) found in dist/" >&2
  echo "      (check release.extra_files glob in .goreleaser.yml)" >&2
  exit 3
fi
MANIFEST_COUNT=$(echo "$MANIFEST" | wc -l | tr -d ' ')
if [ "$MANIFEST_COUNT" -ne 1 ]; then
  echo "FAIL: expected exactly 1 manifest file, got $MANIFEST_COUNT:" >&2
  echo "$MANIFEST" >&2
  exit 3
fi

if ! jq -e . "$MANIFEST" >/dev/null; then
  echo "FAIL: $MANIFEST is not valid JSON" >&2
  exit 3
fi

VERSION=$(jq -r '.version' "$MANIFEST")
if [ "$VERSION" != "1" ]; then
  echo "FAIL: $MANIFEST .version = '$VERSION', want '1' (Registry manifest schema version)" >&2
  exit 3
fi

PROTO=$(jq -r '.metadata.protocol_versions | join(",")' "$MANIFEST")
case ",$PROTO," in
  *,5.0,*)
    ;;
  *)
    echo "FAIL: $MANIFEST .metadata.protocol_versions does not include '5.0' (have: $PROTO)" >&2
    echo "       Plugin SDK v2 advertises protocol 5.0; Registry rejects manifests missing this." >&2
    exit 3
    ;;
esac
echo "  [OK] $MANIFEST = {version:1, metadata.protocol_versions:[\"$PROTO\"]}"

echo
echo "==> All FR-014 release-shape assertions passed."
