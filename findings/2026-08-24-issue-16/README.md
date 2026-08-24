# Issue #16 root-cause investigation — 2026-08-24

Live evidence for the [#16](https://github.com/Project-Arrakis/dune-resource-scanner/issues/16)
recall investigation, captured against `dune-dev`, DeepDesert process
(`DeepDesert_1`, PID 390735 at the time). Coriolis seed was still `2`, unchanged
since session 3, so the map had **not** regenerated and `dune.markers` was still
valid ground truth for these coordinates.

**Ground truth**: the WindPass box, `-near=-4368,-198837 -tolerance 15000`
(a 30,000 x 30,000 uu box — note `tolerance` is a per-axis half-width, not a
radius). `dune.markers` lists **211** markers inside it.

## The headline result

| Stage | Coverage of the 211 markers |
|---|---|
| Pass 1 — raw (X,Y) transform located in memory | **211 / 211 (100%)** |
| Pass 2 — resolved to a validated actor | **6 / 211 (2.8%)** |

Positions are not the problem. **Every** node's position is present in scanned
memory, to within 1 m. The loss is entirely in pass 2's actor resolution, and
what is actually missing from the output is a resource *type*, not a position.

## Files

| File | What it is |
|---|---|
| `windpass-markers-ground-truth.csv` | The 211 markers (`type,x,y`), as fed to the diagnostics |
| `windpass-markers-raw.txt` | Same query with `z` and `long_range`, raw psql output |
| `baseline-windpass-scan.json` | Unmodified `-mode proximity` output: 96 actors, only **42 distinct positions** |
| `funnel-before-nan-fix.txt` | Instrumented funnel, before #18: 17,385,580 raw hits, 12.6 GB peak RSS |
| `funnel-after-nan-fix.txt` | Same, after #18: 623,083 raw hits, 165 MB peak RSS, coverage still 211/211 |
| `offset-agnostic-probes.txt` | Three nodes probed without assuming `Transform=384`/`RootComponent=576` |
| `memory-layout-dump-titanium.txt` | Annotated qword dump around a real TitaniumOre transform |
| `array-walk-stride384.txt` | Walking the 384-byte record array: best 85/211 (40.3%) |
| `type-discriminator-search-v1.txt` | First discriminator search — scoring was too weak, see below |
| `type-discriminator-search-v2-exclusive.txt` | Same with exclusive purity: best 11/17 types, 81 collisions |
| `type-discriminator-search-v3-recsig.txt` | Restricted to hits matching the spawn-record signature |
| `tools/` | The four throwaway diagnostics, `//go:build ignore` so they stay out of the build |

## Root cause

The scanner resolves a node by walking `actor -> RootComponent -> transform` and,
in pass 2, by searching for a pointer to `hit - Transform`. For ore/scrap/pickup
nodes **that chain does not exist in memory.**

An offset-agnostic probe searched every 8-byte-aligned address in
`[hit - 2048, hit]` for *any* pointer anywhere in memory, then walked back up to
2048 bytes from each such pointer looking for a valid vtable/ClassPrivate chain —
assuming neither offset:

| Probe | pass-1 hits | resolutions |
|---|---:|---:|
| TitaniumOre `-3814,-198877` (baseline "finds" it) | 13 | **0** |
| TitaniumOre `-7923,-207410` (baseline misses it) | 13 | **0** |
| ScrapMetalPart `-17893,-208453` (baseline misses it) | 14 | 2, at `tOff=1936, rcOff=1104` |

**Nothing points into the 2 KB preceding these transforms.** Widening the region
set — the hypothesis this issue was opened with — cannot help; there is no
back-reference to find.

The first row matters on its own: the node the baseline *does* report resolves to
zero actors when probed precisely. Its wide-scan "hit" was a neighbouring actor
8.7 uu away, so several of the 6/211 successes are coincidental neighbours rather
than the nodes themselves.

## What these objects are

`memory-layout-dump-titanium.txt` shows a repeating **384-byte record** (bases at
−640 and −256 relative to the hit, exactly 384 apart):

```
 +0    HEAP-PTR              (0x…2700 / 0x…2600 — sequential, 256 apart)
 +8    0x0000000100000001
 +16   2
 +232  f64 x, y, z           z ~ 1_051_450   (pre-trace sentinel Z)
 +256  f64 x, y, z           z ~ 2_844       (terrain-snapped Z)  <- hits land here
 +280  EXE-PTR
 +320  EXE-PTR
 +336  EXE-PTR
```

The record before our node holds `(23838.4, −209130.0, 2844.7)` — another node's
position. The two triples 24 bytes apart, one with `Z ~ 1.05e6` and one with a
terrain Z, read as a downward ground-snap trace. The same signature appears in the
raw hits (`0x…4c8` z=1052462.99 vs `0x…4e0` z=3856.99; the marker's own z is 3867).

These are **spawn records in a large array, not individually-referenced UObject
actors** — consistent with `CONTINUATION.md` §10d, where `dune.actor_spawners`
holds discovery-independent content-block spawner slots.

Spice and flour sand are unaffected: seed mode finds them via the `BaseValue`
field *inside a genuine actor*, which is why that path was always reliable.

## What did not work, and is recorded so it is not retried blindly

- **Fixed-stride array walk.** Walking outward at stride 384 and reading the `+256`
  triple raises coverage from 2.8% to **40.3%** at best across 11 seeded candidate
  arrays (4, 6, 74, 33, 80, 77, 78, 41, 38, 85, 52 of 211). A real jump, but the
  stride is clearly not uniform across the buffer, so this is not the answer as-is.
- **Type-discriminator search, v1.** Scored an offset by "every marker of a type
  shares some value there". `0x0` and a common vtable satisfy that for every type,
  so it reported a meaningless `pure=15`. **The metric was wrong, not the data** —
  recorded because the output looks like a result and is not one.
- **Type-discriminator search, v2.** Re-scored requiring a value to be shared by
  every marker of a type *and* by no marker of any other type. Best offset (−16)
  resolves 11 of 17 types but with 81 cross-type collisions — too noisy to build on.
  Each marker has 8–334 position copies in memory (render, physics, nav, spawn
  data), and mixing them dilutes any real signal.

## The strongest concrete result: the spawn-record signature works

`type-discriminator-search-v3-recsig.txt` filters pass-1 hits down to those whose
`hit - 256` looks like a spawn record — a heap pointer at `+0` and the constant
`0x0000000100000001` at `+8`, both taken from the layout dump.

| | Unfiltered | Signature-filtered |
|---|---:|---:|
| Plausible-Z hits | 106,467 | **274** |
| Markers covered (within 1 m) | 211 / 211 | **152 / 211 (72%)** |
| Position copies per marker | 8 - 334 | **1 - 2** |

This is the most useful single finding after the root cause. A signature derived
from **one** memory dump reduces the candidate set by 99.7% while still covering
72% of known nodes, at 1-2 copies each instead of hundreds. Compare the current
shipped behaviour: 2.8% coverage, and 96 returned actors collapsing to 42 distinct
positions.

It is a **25x recall improvement over the shipped scanner** and needs no actor
chain, no pointer back-reference, and no `ClassPrivate`.

Two honest caveats:

- **72%, not 100%.** 59 markers have a pass-1 hit but no signature-matching record.
  The signature is from a single sample and is very likely over-strict (the `+8`
  constant especially). Relaxing it is the obvious next step, measured against this
  same ground truth.
- **It still yields no type.** Scoring every offset from -2048 to +512 across the
  filtered set produced `pure = 0` — no offset holds a value shared by all markers
  of a type and no others. The ranking in that file is also misleading: with
  `pure = 0` everywhere it falls through to sorting by fewest collisions, which
  promotes all-zero offsets (`distinct = 1`). Re-rank by high `distinct` and low
  `collisions` before reading it again.

## Map-wide census — the signature validated on 37x more ground truth

The 72% figure above came from one box and 211 markers. The operator had since
explored a great deal more, so `dune.markers` for DeepDesert had grown from 440
(session 3) to **7,934** rows across **31 types**. `tools/census.go` scans the whole
world in one pass instead of a box, filters on the spawn-record signature inline,
and streams to disk.

```
raw plausible-XYZ triples: 59,254,091
strict signature records:      85,788
elapsed=17.3s  maxrss=136 MB
```

For comparison, the shipped scanner takes 2-5 minutes on a *single box* and peaked
at 12.6 GB before #18.

### Coverage: 5,144 / 7,934 markers (64.8%)

Against the shipped scanner's **2.8%** — a **23x improvement**, now measured on the
whole map rather than one box. Full output in `census-analysis-mapwide.txt`.

| Type | Matched | Total | % |
|---|---:|---:|---:|
| DolomiteRock | 73 | 82 | **89.0%** |
| AzuriteOre | 52 | 63 | 82.5% |
| RhyoliteOre | 40 | 50 | 80.0% |
| TitaniumOre | 164 | 211 | 77.7% |
| ScrapMetalWreckage | 980 | 1338 | 73.2% |
| MagnetiteOre | 30 | 41 | 73.2% |
| StravidiumOre | 72 | 99 | 72.7% |
| BasaltOre | 102 | 146 | 69.9% |
| BauxiteOre | 60 | 86 | 69.8% |
| FuelCellPart | 688 | 984 | 69.9% |
| RhyolitePickup | 535 | 795 | 67.3% |
| ScrapMetalPart | 1431 | 2151 | 66.5% |
| BrittleBush | 299 | 560 | 53.4% |
| BauxitePickup | 25 | 83 | 30.1% |
| BasaltPickup | 37 | 144 | 25.7% |
| DolomitePickup | 21 | 83 | 25.3% |
| StravidiumPickup | 17 | 77 | 22.1% |
| TitaniumPickup | 34 | 162 | 21.0% |
| MagnetitePickup | 6 | 41 | 14.6% |
| Cave / Shipwreck / Ecolab / TaxiService / Hazard\* / HomeBase | 0 | 32 | **0%** |

Three clean patterns:

- **`*Ore` and rock nodes: 66-89%.** The signature suits them well.
- **Small `*Pickup` nodes: 14-30%** — with `RhyolitePickup` and `AzuritePickup` the
  odd ones out at 67%. Small pickups very likely use a different record shape;
  this is the most promising lead for pushing past 65%.
- **POIs: 0%**, exactly as §5c predicts — structures are streamed per player and
  are simply not resident. Not a defect.

### Type attribution: four routes tested, all ruled out

Using 2,931 records with an **unambiguous** single-type label (matched markers agree
*and* no marker of another type within 300 uu):

| Route | Result |
|---|---|
| Actor chain (`actor -> RootComponent`) | No back-references exist at all (see above) |
| All 48 record offsets, 0..376 | No value is shared by every marker of a type and no others. `+0` is unique per record (2,931 distinct, **zero** collisions) — a per-instance handle |
| The object `+0` points at, first 256 bytes | Not a UObject. Holds float64 coordinate pairs — a bounding box — so no class pointer to read |
| Memory-address clustering | Address-adjacent labelled records share a type **30.9%** of the time against a **12.9%** chance baseline. A real signal, far too weak to classify; every type spans the same ~3.7 GB |

That is a genuine negative result, recorded so it is not re-derived: **the resource
type does not appear to live in the record, in what it points at, or in its
placement.**

### CORRECTED: the unmatched records may well be real nodes after all

**This section previously claimed the 80,803 unmatched records were "NOT all
undiscovered nodes", citing a Z-profile gap of 18,058 vs 3,715. That claim was
largely wrong and is corrected here in the same session it was made.**

The Z gap was mostly **exploration bias, not object type**:

| | median Z |
|---|---:|
| Records in well-explored cells (>=200 markers) | **4,701** |
| Records elsewhere | **19,317** |
| All DD markers | 3,229 |

Exploration is concentrated in low-lying terrain; the unexplored north (Shield
Wall) is simply high ground. Controlled to well-explored cells only, the gap
between marker-matched and unmatched records collapses from 3,715 vs 18,058 to
**3,559 vs 7,060**. The density argument weakens for the same reason.

A structural test then ruled out the "they are different objects" reading
outright. Deriving a signature from the 5,899 marker-confirmed records —
**47 of 48 record fields agree perfectly across all of them** — and applying it
map-wide:

| | |
|---|---:|
| Records passing the confirmed-node signature | **82,682 / 85,784 (96.4%)** |
| Records rejected | 3,102 (3.6%) |

**96.4% of all records are structurally indistinguishable from confirmed nodes.**
There is no structural signal separating "node" from "not node", most plausibly
because most of them *are* nodes.

**Why this cannot be settled from data alone.** This is a positive-unlabelled
problem: "no marker here" does not mean "no node here", because ore and scrap
markers (`long_range=false`) only appear on close approach. No amount of
re-analysis fixes that — it needs a ground-truth visit.

Two genuine data-quality issues did surface, both minor and filterable:

- **1,709 records (2.0%) sit within 1,000 uu of the world origin** and are junk;
  567 are inside 100 uu, including 43 at exactly `(1,1,1)`. `census.go`'s
  `plausibleXY` accepts any \|v\| >= 1, which is too permissive.
- **1,051 duplicate positions** (84,733 distinct out of 85,784).

### Ground-truth validation targets

Three clusters of predicted nodes in terrain with **no known marker within
8,000 uu**, i.e. never explored. A positive result proves the scanner finds
undiscovered nodes, which is precisely the post-storm capability. Z is +200 so
the operator lands above ground:

```
dune admin teleport '<fls-id>'  185338  711895  2180     # 141 predictions nearby
dune admin teleport '<fls-id>' -579810  515940  2133     # 125 predictions nearby
dune admin teleport '<fls-id>'  500113  214825  3288     # 110 predictions nearby
```

Needs `DUNE_ADMIN_ASSUME_YES=1`, and per §6c wait several seconds before
re-reading position.

## Where the next session should start

Positions are already at 100% recall. The open question is **type attribution
without an actor**. Two concrete leads, in order:

1. **Relax the record signature, then re-score.** The signature already gives 72%
   coverage at 1-2 copies per marker. Loosening the `+8 == 0x0000000100000001`
   constant and re-measuring against the 211-marker ground truth is the cheapest
   available win, and a wider filtered set gives the type search more to work with.
2. **The spawn record's `+0` heap pointer.** Sequential across adjacent records and
   the most likely per-record type/definition handle. Follow it and inspect what it
   points at, rather than scoring it blind as v3 did.
3. **The three EXE pointers at record `+280/+320/+336`.** These are the only
   module-relative values in the record, so unlike `ClassPrivate` they would be
   **stable across restarts** as an offset from the module base — which would also
   solve the ASLR/anchor problem in `CONTINUATION.md` §10a.

Both are falsifiable against this same 211-marker ground truth, and the harness to
do it is in `tools/`.
