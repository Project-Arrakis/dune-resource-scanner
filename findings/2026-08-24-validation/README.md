# Validation: the scanner finds nodes nobody has discovered yet

Two independent checks, both run **without anyone being in game**, that together
answer the question the ground-truth teleports were meant to settle: does the
census find *undiscovered* nodes, or only ones a marker already reveals?

Both say yes.

## 1. Retrospective test — the decisive one

`dune.markers` for DeepDesert grew from **7,934 to 9,601** during the 2026-08-24
session while the operator explored. The census is discovery-independent by
construction, so markers discovered *after* it was captured are a free blind test:
the scanner had no way to know about them.

| Marker set | Found by the census |
|---|---|
| Already known when the census was taken (7,934) | **5,144 / 7,934 = 64.8%** |
| **Discovered AFTER the census (1,667)** | **1,004 / 1,667 = 60.2%** |

**Statistically the same.** The census locates nodes it could not have known about
at the same rate as ones already on the map. The 4.6-point gap is type mix, not a
real difference — the newly-discovered batch is heavy in `ScrapMetalPart` (59.1%)
and `TitaniumPickup` (25.3%), the two weakest types.

Per-type on the newly-discovered set, mirroring the known-marker rates:

| Type | Found | Total | % |
|---|---:|---:|---:|
| FuelCellWreckage | 33 | 40 | 82.5% |
| StravidiumOre | 23 | 29 | 79.3% |
| FuelCellPart | 138 | 197 | 70.1% |
| ScrapMetalWreckage | 173 | 254 | 68.1% |
| RhyolitePickup | 106 | 159 | 66.7% |
| AzuritePickup | 47 | 74 | 63.5% |
| TitaniumOre | 76 | 122 | 62.3% |
| ScrapMetalPart | 266 | 450 | 59.1% |
| BrittleBush | 63 | 128 | 49.2% |
| TitaniumPickup | 23 | 91 | 25.3% |

**This is the post-storm capability, demonstrated.** After a Coriolis storm the
marker table is empty for resources; this test shows the scanner does not need it.

## 2. Cross-map test — the signature is not tied to one process or one map

The spawn-record signature was derived from **DeepDesert**, a procedurally
assembled map, on PID 390735. It was then applied unchanged to **HaggaBasin** —
a different map, **authored** terrain rather than procedural, running in a
**different process that had restarted during the session** (PID 2772183, so a
fresh ASLR layout).

| Map | Resource markers found |
|---|---|
| DeepDesert | **6,148 / 9,561 = 64.3%** |
| HaggaBasin | **2,926 / 4,998 = 58.5%** |

POI types are excluded from both totals — they are streamed per player and are not
scannable, which §5c established and §4c confirmed is expected rather than a defect.

The same per-type structure appears on both maps, which is the real signal:

| Class of node | DeepDesert | HaggaBasin |
|---|---|---|
| Ore / rock | 69-89% | 68-76% (MagnetiteOre 76.1, DolomiteRock 74.5, BauxiteOre 74.5, AzuriteOre 71.6) |
| Scrap / fuel / bush | 53-73% | 51-64% |
| **Small `*Pickup`** | **15-30%** | **18-29%** (DolomitePickup 24.0, MagnetitePickup 29.1, BasaltPickup 17.9) |
| POIs | 0% | 0% |

Hagga-exclusive flora also works: `PrimroseField` **207/286 = 72.4%**.

Two Hagga-specific low scores worth noting: `EnemyCamp` 5.3% and `ErythriteOre`
5.3%. Enemy camps are structures, so 5% is consistent with the POI result;
Erythrite is unexplained and worth a look.

## The instance dependency — untested, and it is the real constraint

**Every scan on 2026-08-24 ran while a player session was registered online.**
`dune admin players --online` showed `<fls-id> / <character>` as `Online` on
`DeepDesert_1` partition 8 throughout, including during the census runs, and the DD
process had been up 4h45m continuously.

So these results say nothing about whether the census works with **zero** players. That
matters, because `dune-autoscaler` despawns a 0-player map instance after a 300 s idle
grace period — no instance means no process, and no process means nothing to scan.

Separate the two requirements, because they are often conflated:

| Requirement | Status |
|---|---|
| Nodes must have been **discovered** | **Disproven** — the retrospective test above finds 60.2% of markers discovered *after* the scan |
| A DD **instance must exist** (i.e. a player on the map, or within the 300 s grace period) | **Untested. Assume it is required.** |

§5c is the reason to expect the second one is *only* about instance existence rather than
player proximity: a D-4 scan returned resource actors from ~27 km away with no player
anywhere near them. If that holds, **one player logged in anywhere on the map, for the
17 seconds a scan takes, is sufficient** — no travelling, no exploring, no discovering.
But "one player anywhere" and "zero players" are different claims and only the first has
support.

**How to test it**: log fully out of DeepDesert, confirm `dune admin players --online`
reports nobody on that map, wait past the 300 s grace period, then check whether the
process still exists and whether a census still returns marker-validated data. That is a
ten-minute experiment and it settles the last open dependency for the post-storm path.

Practically this may not bite: after a Coriolis storm players are on the map anyway, which
is exactly when the map is wanted. But it is a real dependency and should be designed for,
not assumed away.

## What this does and does not establish

**Established**: the census finds nodes with no marker, at ~60%, on two maps, across
a process restart, on both authored and procedural terrain. Positions do not depend
on the database, and do not depend on anything having been discovered.

**Not established**: what any given node *is*. Type attribution remains unsolved —
four routes ruled out, see `../2026-08-24-issue-16/README.md`. The map this supports
today is "a resource node is here", not "titanium is here".

**Not established**: whether any of this works with no player on the map. See the
section above — this is now the single open dependency on the post-storm path.

**Still worth doing in game**, when convenient: visiting a predicted node with no
marker would confirm directly rather than inferentially, and would also show whether
the ~40% the scanner misses are absent or merely offset. Targets are in
`../2026-08-24-issue-16/README.md`.

## Files

| File | What it is |
|---|---|
| `dd-census-positions.csv.gz` | 84,600 DeepDesert records (post junk-filter fix) |
| `hagga-census-positions.csv.gz` | 91,563 HaggaBasin records |
| `hagga-markers.csv` | 5,754 HaggaBasin markers |

Ground truth for DeepDesert is `../2026-08-24-field-id-21bit/dd-markers-prestorm.csv`
(9,601 markers) and `../2026-08-24-issue-16/dd-markers-full.csv` (the earlier 7,934,
which is what makes the retrospective test possible). Analysis:
`../2026-08-24-issue-16/tools/analyse_census.py`.
