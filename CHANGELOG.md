# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project has not yet cut a tagged release; entries are grouped under
`Unreleased` until v0.1.0.

## [Unreleased]

### Added

- Initial scaffold and clean-room design notes (`CONTINUATION.md`).
- `internal/memscan`: core, unit-tested scanning/validation logic —
  `/proc/<pid>/maps` parsing, actor-shape validation (vtable/ClassPrivate/
  RootComponent/Transform offset chain), seeded int32 byte-pattern scan,
  proximity (X,Y) scan for the new near-a-known-position capability, and
  backward pointer-reference resolution.
- `cmd/dune-resource-scanner`: CLI wiring for `seed` and `proximity` scan
  modes, JSON output.
- CI: build/vet/test on every push and PR to `main`; org-shared
  `reusable-security-scan.yml` (gitleaks, semgrep, trivy).
[#18]: https://github.com/Project-Arrakis/dune-resource-scanner/issues/18
