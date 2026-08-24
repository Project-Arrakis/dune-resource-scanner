# Continuation prompt — Dune Awakening dynamic Live Map

Lives at `~/projects/repos/dune-resource-scanner/sessions/CONTINUATION-PROMPT.md`.
Paste it into a fresh session, and **overwrite it in place before that session ends**.

Last rewritten: **2026-08-24**.

---

Continue the Dune Awakening Live Map work. **Read
`~/projects/repos/dune-resource-scanner/CONTINUATION.md` first — the Goal, then §10,
§11, and the §4a–§4d block from session 4.** Earlier sections predate the real product
requirement and reach conclusions it overturns; the document flags those inline.

## The goal

Build a **dynamic live map of both Deep Desert and Hagga Basin** showing **all resources,
POIs, player buildings, and storm/worm activity** — and critically, one that works
**immediately after a Coriolis storm regenerates Deep Desert**, so players know where to
go without re-exploring.

`dune.markers` is discovery-driven for resources and empty right after a wipe, so a
database-only map can only report what someone already found. Reading the live server's
process memory is the only way to see undiscovered ore nodes.

## Where this actually stands (read before planning anything)

| Capability | State |
|---|---|
| **Node positions, including undiscovered ones** | **Working.** ~60–64% recall, whole map, 17 s, 136 MB RSS |
| **Naming a class** (this class is TitaniumOre) | **Works** — CONTINUATION §6, 1-3 confirmations per class |
| **Typing what the census finds** | **Blocked.** Naming works via actors, but only 2.8% of markers resolve to an actor while the census reaches 72%; overlap is 0.7% |
| Spice by tier, Flour Sand | Exact from `resourcefield_state` + `field_id` decode — **but only the inner ~87% of the map** (21-bit packing limit) |
| Named POIs | **Complete** from `dune.markers` where `long_range=true`; scanner not needed |
| Bases, vehicles, storage, players | Already shipped in Core |
| Zero-player operation | **Proven 2026-08-24** — 64.0% coverage with nobody logged in, identical to a with-player scan |

**The honest summary**: post-storm you could ship spice, flour sand, bases, POIs and an
*unlabelled* "a resource node is here" layer. You could not say which node is Titanium.

## Two experiments left over from 2026-08-24, both cheap, both need the operator

Do these first — they are short and they change what is worth building.

1. ~~Zero-player scan~~ — **done 2026-08-24, and it works.** Zero online players, process
   alive past T+540 s, census returned 84,569 records at 64.0% coverage, identical to the
   with-player run. No player dependency on the post-storm path.

2. **Walk-to validation — `dune admin teleport` does NOT work.**
   Re-tested three ways on 2026-08-24 (two long-range, one 3,000 uu hop inside loaded
   terrain): all `publish=ok`, exit 0, **zero movement**. §6c is re-corrected accordingly.
   **No validation plan may depend on teleporting the operator.** Targets, controls and
   the exact before-state are in `findings/2026-08-24-issue-16/README.md`: four
   predictions ~100 m from the operator's DD base in terrain with no marker within ±6 km,
   plus three already-marked controls at 3–15 m. A positive result would also yield the
   first *labelled* records in unexplored terrain, which is exactly what type attribution
   has never had.

## Track A — the scanner

### #16 is a redesign, not a bug fix

Root cause, established 2026-08-24 and evidenced in `findings/2026-08-24-issue-16/`:

- Pass 1 locates **211/211** known markers in a test box (100%). Pass 2 resolves **6/211**
  (2.8%). The loss is entirely in actor resolution.
- The hypothesis this issue was opened with is **disproved**. Offset-agnostically,
  **nothing in memory points into the 2 KB preceding these nodes' transforms**, so
  widening the region set cannot help.
- Ore/scrap/pickup nodes are **384-byte spawn records in an array**, not UObject actors.
  Spice and flour sand are unaffected — seed mode reaches them via `BaseValue` inside a
  genuine actor, which is why that path always worked.
- Filtering on the spawn-record signature gives **64.3% on DeepDesert, 58.5% on
  HaggaBasin** — map-independent and process-independent, surviving a restart with fresh
  ASLR.

### Type attribution — four routes ruled out, do not re-derive

| Route | Result |
|---|---|
| Actor chain (`actor → RootComponent`) | No back-references exist at all |
| All 48 record offsets, 0..376 | No per-type value. `+0` is unique per record (zero collisions) — a per-instance handle |
| The object `+0` points at, first 256 bytes | Not a UObject — holds float64 coordinate pairs (a bounding box) |
| Memory-address clustering | 30.9% same-type adjacency vs a 12.9% chance baseline — real, far too weak |

Still unexplored: the handle-like `0x498000xx` value at the followed object's qword[1];
whether render/mesh data carries identity; and the **small-`*Pickup` gap** (14–30% recall
against 66–89% for ore, with RhyolitePickup and AzuritePickup the odd ones out at 67%) —
a second record shape is the likely cause and closing it pushes coverage past 65%.

### Smaller items

- **Emit `[]` not `null`** on an empty scan (Go nil-slice marshalling; breaks naive
  parsers). Still open.
- **Class-anchor mode** — reconsider. Anchors were meant to re-derive `ClassPrivate` after
  a restart, but ore nodes have no reachable `ClassPrivate`.
- **Fast-capture** — dump regions to disk, analyse offline. A slow inline scan cannot
  catch a sandworm or a storm.
- ~~#14 `ValidTransform`~~ and ~~#18 NaN pairs~~ — **done**, PRs #17 and #19.

## Track B — data sources

| Layer | Source | State |
|---|---|---|
| Spice (3 tiers), Flour Sand | `resourcefield_state` + `field_id` decode | **Ready for the inner ~87% of the map** — the 21-bit packing cannot represent \|x\| or \|y\| beyond 1,048,575, where 12.9% of real DD markers sit |
| Ore / stone / flora nodes | scanner + type attribution | **Positions work; types blocked** |
| Named POIs | `dune.markers` where `long_range=true` | **Ready and complete** — revealed at range, not by visiting; scanner not needed |
| Bases, storage, vehicles, players | existing `liveMap*` in Core | **Already shipped** |
| Dropped loot | `dune.actors` `BP_LootContainer` | **Ready**, unused |
| Hagga region labels | marker `DisplayName` | **Ready** — normalise spelling |
| DD grid labels | 9×9 @ 270,000 uu, origin (−52656, −52066) | **Ready** |
| Map regeneration signal | `debug_get_coriolis_seeds()` | **Ready** |
| Sandstorm per-structure | `game_events` type 13 | **Partial** |
| Active storm position, sandworms, hidden treasure | — | **Open**, memory-only |
| Primrose water | — | **Unreachable** — not an item, not persisted |

Remaining identification work: **Jasmium** (Hagga only, last of the 15); `Dolomite*`
(Carbon Ore) item name; **Impure Fuel** (unidentified — `FuelCellPart` yields `Oil`, which
displays as "Fuel Cell").

## Track C — build the Live Map

Only after Track A. No Core code exists yet. Constraints from the eight-hats audit stand:
no continuous privileged sidecar, scoped read-only DB role, console API reads a read-only
bind-mounted scanner output file rather than talking to the scanner. Core's Live Map
already exists (`console/web/src/features/liveMap/LiveMapPanel.tsx`,
`console/api/src/duneDb.js` `liveMap*`) and uses `LIVE_MAP_CONFIGS` min/max normalisation
on 4096×4096 images — **reuse it; do not introduce a second calibration scheme.** Known
bug to fix while there: `liveMapBases()` filters `coalesce(a.partition_id,0) > 0`, so a
base whose instance despawned silently vanishes.

## Working practices that matter

- **`dune admin teleport` does not work.** Pick targets near where the operator already is.
- **Operator must be stationary before any *proximity* scan** — it reads `-near` at launch
  and runs minutes. The map-wide census has no `-near`, so movement is irrelevant there.
- **Bound a diagnostic's working set inside the tool**, not just with a wall-clock timeout.
  Two runs this session had to be killed manually — one reached 10.7 GB RSS on a host with
  24 GB free that also runs the live game server. `census.go` streams to disk and takes
  `-maxrecords` for exactly this reason.
- **Check what a metric rewards before believing its ranking.** A type-discriminator search
  reported a confident `pure = 15` that was pure artifact (`0x0` satisfies "all markers of
  this type share a value").
- **Verify a filter against real data before shipping it.** A per-axis origin filter looked
  obviously right and would have discarded up to 98 real nodes, because DD crosses the
  origin.
- **Match marker-centric, not actor-centric.** Nearest-marker-wins produced two
  confidently-wrong labels.
- **Never infer item names.** Four predictions, four wrong. Node name, item name and
  display name are three independent things — gather and diff `dune.items`, ordering by
  `acquisition_time`.
- **Verify third-party claims.** A `field_id` report this session was right about the
  21-bit limit and wrong about it causing the observed misses.
- **Use a remote-side `timeout`** — a local `timeout ssh` leaves the remote process running.
- **Never write anything a future session needs to `/tmp`.** It is `tmpfs`. Session notes go
  in `sessions/`, evidence in `findings/<date>-<topic>/`, committed. Binaries deployed to
  `dune-dev:/tmp` are fine — they are redeployable and hold no findings.

## Environment

`ssh dune-dev`, passwordless sudo. **DeepDesert = `DeepDesert_1`, Hagga = `Survival_1`.**
Find PIDs with `/tmp/gamepid.sh <map>` (matches on RSS to skip the `/bin/sh` wrapper, which
shares a truncated `comm` with the binary — plain `pgrep -f` also matches your own ssh
shell). Cross-compile locally (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`) and `scp`
over. `dune database sql "<query>"` is the read-only DB interface; for complex SQL write a
file locally and `scp` it rather than fighting shell quoting — note single quotes inside a
single-quoted `ssh '...'` will break. `dune-dev` is sanctioned for this work; `dune-prod`
is not.

Persistent scan data lives on `dune-dev:~/scan-findings/` (not `/tmp`).

Test character: FLS `<fls-id>`, name `<character>`.

## Repo state as of 2026-08-24

`main` green. **Open issues: #16 (the blocker), #1 (v1 tracking).** No open PRs.
Merged this session: #17, #19, #20, #21, #22, #23, #24. #14 and #18 closed by their fixes.

Evidence directories, all with re-runnable tooling:

- `findings/2026-08-24-issue-16/` — root cause, the funnel, the census, `census.go` and
  `analyse_census.py`
- `findings/2026-08-24-validation/` — the retrospective and cross-map validation
- `findings/2026-08-24-field-id-21bit/` — the 21-bit verification and `analyse_field_id.py`
