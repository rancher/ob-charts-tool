# ob-charts-tool

Internal CLI tool for the ORBS team to manage Helm charts in the `ob-team-charts` repository.

**Note:** This CLI is bespoke for ORBS team workflows. If you're looking for reusable Helm chart utilities, see the `helmtools` library below.

## helmtools Library

The `helmtools` package provides general-purpose, reusable Go libraries for working with Helm charts. Extracted from this CLI tool, it's suitable for use in any project that needs to analyze charts, extract images, or track upstream versions.

See [helmtools/README.md](helmtools/README.md) for documentation and usage examples.

```bash
go get github.com/rancher/ob-charts-tool/helmtools
```

## CLI Usage (ORBS Team)

The CLI is used within the `ob-team-charts` repository for:
- CI workflows
- Development tasks (version checks, rebase hints)
- QA automation (image extraction, dependency validation)

Build and install locally:

```bash
go install github.com/rancher/ob-charts-tool@latest
```

## Contributing

### Prerequisites

- Go 1.26 or later
- golangci-lint

### Development

This project uses Go workspaces with two modules:
- Root module: CLI tool (`github.com/rancher/ob-charts-tool`)
- Helmtools module: Reusable library (`github.com/rancher/ob-charts-tool/helmtools`)

```bash
# Run all tests
go test ./...

# Run linter
golangci-lint run ./...

# Run integration tests (requires network)
go test -tags=integration ./helmtools/git/
```

### Making Changes

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linter
5. Submit a pull request

### Releases

See [VERSIONS.md](VERSIONS.md) for the release process and versioning strategy.