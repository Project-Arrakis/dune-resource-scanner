# dune-resource-scanner

Locating raw-resource nodes on a self-hosted **Dune: Awakening** server by reading the
game server's own process memory, so a live map can show where resources are **immediately
after a Coriolis storm regenerates Deep Desert** — when the game's own marker table is
empty and no database can answer the question.

📊 **[Findings — what is established, disproved, and still open](https://project-arrakis.github.io/dune-resource-scanner/)**
· [same index as Markdown](findings/README.md)

> Read the rendered findings page for the results. This README covers what the tool is and
> how to run it.

## Where this stands

| Capability | Status |
|---|---|
| Node **positions**, including ones nobody has discovered | ✅ ~60–64% recall, whole map, 17 s |
| Node **types** — which node is Titanium | ❌ Unsolved. Four attribution routes ruled out |
| Spice and Flour Sand from the database | ⚠️ Exact, but only the inner ~87% of the map |
| Named points of interest from the database | ✅ Complete without exploration |

The headline result is that the scanner finds nodes **before anyone has discovered them** —
60.2% of markers that appeared *after* a scan was taken, against 64.8% for markers already
known. That is the capability the whole project exists for, and it is measured rather than
assumed. What is still missing is a *name* for each node, not a position.

## What it does

Reads `/proc/<pid>/mem` on a running game server, **read-only**, and matches the memory
layout of resource spawn records against known world coordinates. It never writes to the
target process and is designed to run as an occasional host job, never a privileged
sidecar.

Resource nodes turned out **not** to be engine actors — they are 384-byte spawn records
that nothing in memory points at, which is why the original actor-walking approach
recovered only 2.8% of them. See the findings for how that was established.

## Scope and intent

Built for an operator's **own self-hosted, non-production development server**, to power a
live map for that server's players. It is read-only, single-purpose, and not a game client
modification.

## Repository layout

| Path | What it is |
|---|---|
| `cmd/dune-resource-scanner/` | The CLI |
| `internal/memscan/` | Memory scanning and actor validation, unit tested |
| `findings/` | Evidence, analysis and re-runnable tooling, one directory per investigation |
| `docs/` | The published findings page — **generated**, never edited by hand |
| `sessions/` | Dated working records and the handoff prompt |
| `tools/` | Repository guards and the page generator |
| `CONTINUATION.md` | The living technical account. Long, and the source of truth |

## Building

```sh
go build ./...
go test -race -cover ./...
```

Cross-compile for the server host:

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/dune-resource-scanner
```

## Contributing to the docs

`findings/README.md` is the single source for the findings page. After editing it:

```sh
python3 tools/build-findings-page.py     # regenerate docs/index.html
./tools/check-public-safe.sh             # no PII or internal addresses
```

CI enforces both, plus `gofmt`, `go vet`, `go test -race`, shellcheck, and
gitleaks/semgrep/trivy.
