# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project has not yet cut a tagged release; entries are grouped under
`Unreleased` until v0.1.0.

## [Unreleased]

### Fixed

- `ValidTransform` now rejects denormal / sub-micrometre non-zero coordinates,
  `Inf` on any axis, and out-of-world `Z` (previously only X and Y were
  bounded, so any finite garbage `Z` passed). Uninitialized memory reading as
  `6.8e-310` was being reported as a real actor ([#14]).
- `FindNearbyXY` no longer accepts NaN pairs. Both range guards were written as
  "skip if outside tolerance", and every comparison involving NaN is false, so
  NaN fell through both and matched every target at every tolerance. 16 bytes of
  `0xFF` (two int64 `-1`s) is the common source. A single live scan produced
  ~17.4M spurious hits from this — almost all of its runtime and its 12.6 GB
  peak RSS ([#18]).

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

[#14]: https://github.com/Project-Arrakis/dune-resource-scanner/issues/14
[#18]: https://github.com/Project-Arrakis/dune-resource-scanner/issues/18
