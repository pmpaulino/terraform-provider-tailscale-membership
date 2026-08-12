# Maintenance: CI, security dependencies, and upstream review

## Task

Restore required CI, remove reachable dependency/toolchain vulnerabilities, refresh the hard fork's retained build/release plumbing, and review upstream changes without reintroducing deleted provider features.

## Proof

The maintenance batch is ready for review when all of these pass from a clean checkout:

```bash
go fmt ./...
go run golang.org/x/tools/cmd/goimports -w -local github.com/pmpaulino .
go mod tidy
git diff --exit-code
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
./scripts/check-docs-coverage.sh
./scripts/check_license_headers.sh .
./scripts/run-license-review.sh
./scripts/test-release-snapshot.sh
```

CI must report green for `test`, `docs-coverage`, `format`, `licenses`, and `release-snapshot`. The release snapshot must contain 11 platform archives, one Registry manifest, and a checksum entry for all 12 files.

## Upstream scope

Port only changes relevant to this single-resource fork: supported Go and direct dependency versions, security fixes, and maintained GitHub Action pins. Do not merge upstream wholesale or adopt its Plugin Framework migration unless this provider's retained membership resource is deliberately migrated in a separate feature.

## Live boundary

No GitHub settings changes, authenticated Tailscale API calls, tags, releases, commits, or pushes are part of the offline batch. Before release, validate the built provider with API-key membership CRUD on a dedicated test tailnet, then run reviewer and release gates.
