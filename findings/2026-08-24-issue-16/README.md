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
