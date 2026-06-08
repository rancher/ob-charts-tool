# Versioning and Releases

This repository contains two independently versioned modules:

1. **CLI Tool** (`github.com/rancher/ob-charts-tool`) - Internal ORBS team tooling
2. **helmtools Library** (`github.com/rancher/ob-charts-tool/helmtools`) - Public, reusable library

## Tag Formats

- **CLI:** `vX.Y.Z` (e.g., `v2.0.0`) → triggers GoReleaser, builds binaries
- **helmtools:** `helmtools/vX.Y.Z` (e.g., `helmtools/v0.1.0`) → Git tag only, consumed as Go module

Both follow [semver](https://semver.org/). helmtools is currently v0.x (pre-stable) and will be promoted to v1 once the API stabilizes.

## Release Strategy

### helmtools changes
```bash
git tag helmtools/v0.1.0
git push origin helmtools/v0.1.0    # Triggers CI: tests, linter, GitHub release
git tag v2.0.0                       # CLI release including new helmtools
git push origin v2.0.0               # Triggers GoReleaser: binaries
```

### CLI-only changes
```bash
git tag v2.0.1              # helmtools version unchanged
git push origin v2.0.1      # Triggers GoReleaser: binaries
```

## Development

The CLI **always** uses local helmtools code via the `replace` directive in `go.mod`. This applies to development, CI, and releases.
Helmtools version tags are only for external library consumers - they don't affect the CLI at all.

```bash
# go.mod always has:
# replace github.com/rancher/ob-charts-tool/helmtools => ./helmtools

# Edit helmtools, CLI sees changes immediately
vim helmtools/chart/client.go
go run . monitoring branch-verify-check
```

## Notes

- `go.work` is gitignored (local dev only)
- **CLI releases** (`v*` tags): GitHub Actions → GoReleaser → binaries
- **helmtools releases** (`helmtools/v*` tags): GitHub Actions → tests, linter, integration tests → GitHub release with changelog
- External helmtools users get only library deps (no CLI bloat: cobra, viper, k8s.io)
- Changelogs are filtered to relevant changes: helmtools releases only show `helmtools/` directory commits
