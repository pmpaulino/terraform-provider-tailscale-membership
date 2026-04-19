#!/usr/bin/env bash
#
# check-docs-coverage.sh
#
# Enforces 100% bidirectional schema<->docs coverage for hand-authored docs
# (FR-019, SC-007). Replaces the tfplugindocs-based check originally proposed
# in tasks.md T056; tfplugindocs cannot run against this provider because of
# the dash-vs-underscore mismatch between Terraform's local provider name
# (`tailscale-membership`) and the resource-key prefix
# (`tailscale_membership_*`). See main.go's package-doc comment for the full
# rationale.
#
# This script is dependency-free: it parses the provider's schema by greping
# for `Schema: map[string]*schema.Schema{ "..." }` literal entries in the
# Go source. That keeps the check fast and self-contained, but it relies on
# our convention of declaring schema field keys as bare double-quoted string
# literals inside the schema map block. If we ever switch to dynamic schema
# construction this script must be reworked.
#
# Coverage assertions:
#
#   1. Every provider-schema field declared in tailscale/provider.go appears
#      under the "## Schema" section of docs/index.md.
#   2. Every field documented under docs/index.md "## Schema" exists in the
#      provider's Go schema (catches stale docs after a schema field is
#      removed).
#   3. Every resource-schema field declared in
#      tailscale/resource_tailnet_membership.go appears under
#      docs/resources/tailnet_membership.md "## Schema".
#   4. Every field documented under docs/resources/tailnet_membership.md
#      "## Schema" exists in the resource's Go schema OR is an implicit
#      Terraform-SDK-provided attribute (`id`, which is automatic for every
#      resource).
#
# Exit codes:
#   0  all assertions passed
#   1  required tooling missing
#   2  one or more coverage assertions failed
#
# Wired into .github/workflows/ci.yml as a required job.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: required command not found: $1" >&2
    exit 1
  }
}

require_cmd rg
require_cmd awk
require_cmd diff
require_cmd sort

# extract_schema_keys <go-file>
#   Prints, one per line, the bare quoted string keys appearing inside the
#   first `Schema: map[string]*schema.Schema{ ... }` block that has the
#   <field>: { ... } literal shape we use throughout this codebase.
#   We deliberately bound to the file's full text rather than a single block
#   because both provider.go and resource_tailnet_membership.go define exactly
#   one such schema map each (provider.go's `Schema:` for the provider config,
#   resource_tailnet_membership.go's `Schema:` for the resource).
extract_schema_keys() {
  local file="$1"
  # Match `\t+"<key>": {` — schema field declarations are tab-indented
  # double-quoted lowercase-with-underscores keys followed by `: {`. Both
  # provider.go and resource_tailnet_membership.go use this convention; if
  # the codebase ever switches to a different schema-construction style this
  # extraction must be reworked.
  rg --no-line-number -o '^\t+"[a-z_]+": \{' "$file" \
    | sed -E 's/^\t+"([a-z_]+)": \{$/\1/' \
    | sort -u
}

# extract_doc_schema_keys <md-file>
#   Prints, one per line, the field names listed under "## Schema" in a
#   hand-authored docs page. Field bullets follow the convention:
#     - `field_name` (Type[, Modifier]) Description sentence.
#   We grep for those bullets only between the "## Schema" heading and EOF
#   (or the next "## " heading, whichever comes first).
extract_doc_schema_keys() {
  local file="$1"
  awk '
    /^## Schema$/ { in_schema = 1; next }
    in_schema && /^## / { exit }
    in_schema && match($0, /^- `[a-z_]+`/) {
      # Strip the leading "- `" (3 chars) and the trailing "`" then anything after.
      s = substr($0, 4)
      gsub(/`.*/, "", s)
      print s
    }
  ' "$file" | sort -u
}

# diff_sets <left-name> <right-name> <left-list> <right-list>
#   Returns 0 if both lists are identical, 1 if they differ. Prints a diff
#   describing the asymmetry on failure.
diff_sets() {
  local left_name="$1" right_name="$2" left="$3" right="$4"
  if diff <(echo "$left") <(echo "$right") >/dev/null; then
    return 0
  fi
  echo "FAIL: $left_name vs $right_name mismatch:" >&2
  echo "    < only in $left_name (schema/docs missing in the other side)" >&2
  echo "    > only in $right_name" >&2
  diff <(echo "$left") <(echo "$right") | sed 's/^/    /' >&2
  return 1
}

echo "==> Provider schema (tailscale/provider.go) <-> docs/index.md"
PROVIDER_SCHEMA_KEYS=$(extract_schema_keys tailscale/provider.go)
PROVIDER_DOC_KEYS=$(extract_doc_schema_keys docs/index.md)

if [ -z "$PROVIDER_SCHEMA_KEYS" ]; then
  echo "FAIL: extracted zero schema keys from tailscale/provider.go" >&2
  echo "      (the regex in extract_schema_keys may be out of date)" >&2
  exit 2
fi
if [ -z "$PROVIDER_DOC_KEYS" ]; then
  echo "FAIL: extracted zero documented keys from docs/index.md '## Schema'" >&2
  exit 2
fi

if ! diff_sets "tailscale/provider.go" "docs/index.md" \
  "$PROVIDER_SCHEMA_KEYS" "$PROVIDER_DOC_KEYS"; then
  exit 2
fi
echo "  [OK] provider schema fully documented and no stale doc entries"

echo
echo "==> Resource schema (tailscale/resource_tailnet_membership.go) <-> docs/resources/tailnet_membership.md"
RESOURCE_SCHEMA_KEYS=$(extract_schema_keys tailscale/resource_tailnet_membership.go)
RESOURCE_DOC_KEYS=$(extract_doc_schema_keys docs/resources/tailnet_membership.md)

if [ -z "$RESOURCE_SCHEMA_KEYS" ]; then
  echo "FAIL: extracted zero schema keys from resource_tailnet_membership.go" >&2
  exit 2
fi
if [ -z "$RESOURCE_DOC_KEYS" ]; then
  echo "FAIL: extracted zero documented keys from tailnet_membership.md '## Schema'" >&2
  exit 2
fi

# `id` is implicit in the SDK (every resource has it) and is always
# documented under Read-Only in the doc page. Add it to the schema side so
# the diff balances. Any other implicit attributes added later must go here.
RESOURCE_SCHEMA_KEYS_AUGMENTED=$(printf '%s\nid\n' "$RESOURCE_SCHEMA_KEYS" | sort -u)

if ! diff_sets "resource_tailnet_membership.go (+ implicit id)" \
               "docs/resources/tailnet_membership.md" \
               "$RESOURCE_SCHEMA_KEYS_AUGMENTED" "$RESOURCE_DOC_KEYS"; then
  exit 2
fi
echo "  [OK] resource schema fully documented and no stale doc entries"

echo
echo "==> All bidirectional schema<->docs coverage assertions passed."
