# Discovery reveals a whole `area_id` zone at once, not one node at a time

A live, real-time-observed mechanism, found by accident while testing something else
entirely (whether a `long_range` POI already exists in `dune.markers` before client-side
discovery -- it does, see below) -- and it turns out to matter more for this project's
actual goal than the question that prompted it.

## What happened

The operator was in an unexplored area of DeepDesert (grid area later identified as
`area_id 64` and `area_id 10`), found an `Ecolab` structure still under in-game fog, and
discovered it. A `Shipwreck` was visible nearby too.

Captured a full snapshot of `dune.player_markers` (the per-player discovery table --
`player_id`, `marker_hash_id`, `discovery_level`, `discovery_method`; distinct from the
global `dune.markers` table) immediately before, then again immediately after the
discovery:

- Before: 17,800 rows
- After: 19,777 rows
- **New rows for this player: 1,977**

Expected roughly 1-2 new rows (the Ecolab, maybe the Shipwreck). Got 1,977.

## What actually got revealed

Joined the 1,977 new `marker_hash_id`s back against the global `dune.markers` table:

| area_id | markers revealed | notable contents |
|---|---:|---|
| 64 | 1,052 | includes the `Ecolab` (1) and `Shipwreck` (1) actually walked to |
| 10 | 913 | no POI walked to here at all -- entirely a byproduct |
| 65 | 11 | a sliver, likely a boundary area |
| 0 | 1 | edge case |

The other 1,975 rows -- everything except the one `Ecolab` and one `Shipwreck` -- are
ordinary resource nodes: `ScrapMetalPart`, `ScrapMetalWreckage`, `FuelCellPart`,
`RhyolitePickup`, `AzuritePickup`, `BrittleBush`, `FuelCellWreckage`, `BauxitePickup`,
`BauxiteOre`, `BasaltOre`, `BasaltPickup`, `StravidiumOre`, `TitaniumOre`, `AzuriteOre`,
`RhyoliteOre`, `TitaniumPickup`, `StravidiumPickup` -- spanning basically every known
resource type, all `long_range=false` (i.e. these are exactly the discovery-only nodes this
project has never been able to type from memory alone). See
`area-type-breakdown-raw.txt` for the full per-area, per-type counts.

**Conclusion: discovery grants area-wide reveal keyed by `area_id`, not per-node
proximity.** Interacting with one structure (or possibly just entering/flying through the
zone -- the exact trigger boundary between "found the Ecolab" and "flew into area_id 10,
which has no POI at all" isn't isolated by this one observation) reveals every marker in
that `area_id`, all at once, fully typed.

## Confirms an existing claim, but for the first time watched happening live

This project already established, from retrospective/static analysis (`../2026-08-24-validation/README.md`,
"60.2% of markers discovered *after* the scan"), that the global `dune.markers` table
grows over time independent of any specific census -- i.e. that discovery creates real,
new global rows, not just per-player UI state. Confirmed again here directly: total DD
marker count was 10,488 in this morning's pre-storm baseline
(`../2026-08-24-storm-watch/pre-storm-baseline/`) and is **12,940** now, several hours and
some unrelated player activity later. This event's own 1,976 markers (all but the one
`Ecolab` that presumably already existed globally as a `long_range=true` POI) are part of
that growth, though not the entirety of it -- other players/activity contributed too over
those 5+ hours, so this one event's exact share of the delta isn't isolated.

**Also confirms, cleanly, something implied but not previously watched directly**: the
`Ecolab` marker was already present in the global `markers` table with `long_range=true`
*before* the operator discovered it (checked directly -- 6 `Ecolab` rows existed
DD-wide, all `long_range=true`, prior to this event). Client-side fog and global-table
existence are genuinely separate: `long_range` POIs are known globally regardless of
individual discovery; ordinary resource nodes are not created in the global table until
*someone* (not necessarily this operator) discovers them, and once created, every
subsequent discovery by a different player is a `player_markers` link to the
already-existing row, not a new global row.

## Why this matters more than it looks like it should

This whole project's central blocker has been: census (memory scanning) finds node
*positions* reliably but cannot determine node *type* through any method tried across four
separate sessions (`../2026-08-24-static-reverse-engineering/`) -- memory field analysis,
static disassembly, live debugging, full-record diffing, none of it works, because these
records genuinely carry no type identity of their own.

This finding sidesteps that problem entirely rather than solving it. **If discovery reveals
an entire area at once, systematically flying through DeepDesert's remaining unexplored
`area_id` zones is a far more efficient path to a fully-typed live map than any further
attempt at record-level type attribution.** DD has 57 distinct `area_id` values total; this
one, partly-accidental excursion covered 2 of them (plus a sliver of a third) and yielded
1,975 freshly-typed resource markers in a single pass. That is roughly the same order of
magnitude as an entire session's worth of targeted single-node confirmations, for one
flythrough's effort.

**Practical implication for the post-storm plan:** the existing `../2026-08-24-validation/README.md`
already flagged 54 of 86 census grid-cells (a different, finer-grained partition than
`area_id`, note -- not the same scheme) as "essentially unexplored," holding over half the
whole census. A systematic sweep of unexplored `area_id` zones -- not necessarily
one-node-at-a-time verification -- is now the clearer, cheaper way to close that gap, both
before and after tonight's storm.

## What this does not establish

- The exact trigger boundary (does entering an `area_id`'s territory alone reveal it, or
  does it require finding a specific structure/POI within it?) is not isolated -- `area_id
  10` was revealed with no POI walked to in it at all, which argues for "just being in the
  zone is enough," but this is one observation, not a controlled test.
- Whether this generalizes to Hagga Basin or other maps is untested.
- Whether this mechanism (and the `area_id` boundaries themselves) survives a Deep Desert
  regeneration is unknown -- tonight's storm is the first opportunity to check.

## Files

- `area-type-breakdown-raw.txt` -- the full per-`area_id`, per-type count of all 1,977
  newly-discovered markers, raw `psql` output.

No raw `player_markers` snapshots or hash IDs are committed here -- `player_id` is
account-linked, and per this repo's established practice, per-player identifying data
stays off the public repo (`tools/check-public-safe.sh` also guards this). The before/after
row counts and the joined, aggregate type/area breakdown above are the reproducible,
non-identifying record of what was found.
