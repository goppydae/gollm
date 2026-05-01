---
title: Build & Release
weight: 60
description: Mage build targets, Nix dev shell, and CI/CD pipelines
categories: [internals]
---

`gollm` uses a combination of **Mage** and **GitHub Actions** for CI/CD.

---

## Versioning

The project version is maintained in a `VERSION` file in the repository root. During build, `Magefile.go` reads this file and injects it into the binary using linker flags (`-ldflags "-X main.version=..."`).

---

## Mage Targets

| Target | Description |
|---|---|
| `Build` | Compile `glm` for the current platform with version injection |
| `Test` | Run all unit tests with coverage |
| `Vet` | Static analysis with `go vet` |
| `Lint` | Run `golangci-lint` |
| `Vuln` | Vulnerability scan with `govulncheck` |
| `All` | Run generate, build, test, vet, lint, and vuln in sequence |
| `Release` | Cross-compile for Linux, macOS, and Windows (AMD64/ARM64), package into `dist/` |
| `Generate` | Run `buf` to regenerate protobuf stubs |
| `Docs` | Generate API reference (gomarkdoc) and build the Hugo site |
| `DocsServe` | Run Hugo dev server at `localhost:1313` with live reload |
| `PkgSite` | Run `pkgsite` for local full API browsing including internals |

---

## CI/CD Pipelines

### Continuous Integration (`ci.yml`)

Triggered on every push to `main` and all pull requests. Runs `mage all` within a Nix environment on both `ubuntu-latest` and `macos-latest`, then uploads per-platform binaries as build artifacts. Coverage is collected and summarised via `go tool cover`.

### Automated Release (`release.yml`)

Triggered by pushing a version tag (e.g., `v1.2.3`). Runs `mage release` to build cross-platform assets and uses `softprops/action-gh-release` to publish them to a new GitHub Release.

### Docs Deploy (`docs.yml`)

Triggered on push to `main` and on published releases. Runs `mage docs` (gomarkdoc + Hugo build) and deploys `docs/public/` to the `gh-pages` branch via `peaceiris/actions-gh-pages`.
