# Live confirmation: the census predicted a node before it was discovered

An in-game validation, not a re-analysis of existing data. The operator flew into H-3
(area_id 46 -- 31 known markers, essentially unexplored) as a direct test of the census's
predictions, and stopped at a real node.

## Sequence of events, in order

1. **08:38** -- baseline marker snapshot captured (`findings/2026-08-24-issue-16/dd-markers-full.csv`, 7,934 rows).
2. **11:03** -- zero-player census captured (`findings/2026-08-24-validation/`), 84,569 records, no player online.
3. Operator flew from G-3 into H-3, reported: *"standing S of a stravidium ore"*.
4. Live position pulled from `dune.actors`: `X=-680175, Y=-950794, Z=4593`.
5. Nearest marker to that position: **`StravidiumOre` at `X=-680553, Y=-949772, Z=4621`,
   10.9 m away.**

## The check that matters

| Question | Answer |
|---|---|
| Did this marker exist in the 08:38 snapshot? | **No.** Nearest match in that snapshot was 158 km away -- genuinely absent |
| Did the 11:03 zero-player census predict a record here? | **Yes -- 1.1 m away, Z off by 14 units** (4607 predicted vs 4621 actual) |

**The census predicted this node's position before anyone had discovered it, using no
database, no player proximity, and no prior visit to this area.** This is the single
clearest confirmation of the project's central claim, because unlike the retrospective
test in `2026-08-24-validation/` (which compared two already-captured snapshots after the
fact), this one was set up and watched happen: fly into unexplored terrain toward a
specific coordinate the census already named, and the marker appears exactly there.

## Second confirmation, same session

A different resource type, ~5 km further into H-3: the operator stopped **1.8 m** from a
`TitaniumOre` marker. Same checks, same result:

| Question | Answer |
|---|---|
| Existed in the 08:38 snapshot? | **No** -- nearest match 155 km away |
| Predicted by the 11:03 zero-player census? | **Yes -- 0.6 m away, Z off by 13 units** -- tighter than confirmation #1 |

**Two for two, two different resource types (StravidiumOre, TitaniumOre), both discovered
live after a census taken with nobody online.** A single hit could plausibly be
coincidence in a dense field; a second, independent hit on a different type meaningfully
narrows that. This is no longer a hypothesis under test -- it is a repeated, direct
observation.

## The actor-resolution attempt, and why it failed as expected

A proximity scan (`-near=-680553,-949772 -tolerance 3000`) was run against the live
process to see whether this specific node resolves to an actor with a `ClassPrivate` --
per section 6's method, that is the only way to attach a *name* rather than just a
position. It returned 20 hits, **none within 900 uu of the marker's actual position** --
the closest were player-adjacent noise (many actors sharing one transform near the
operator, the exact false-positive pattern section 6 already warns about).

This is not a new problem. It is the same ~97% non-resolution rate already established:
only ~2.8% of nodes resolve to an actor at all (`2026-08-24-issue-16/README.md`), and this
node landed in the majority that does not. Type attribution for this node comes from the
in-game marker itself, not from memory.

## What this adds to the existing findings

- **Strengthens the retrospective test's conclusion** with a second, independently-timed
  method: watching a specific prediction resolve live, rather than comparing two already
  -captured snapshots.
- **Confirms the census's Z accuracy** at a fresh, previously-unseen coordinate (14 uu
  off), not just on the historical marker set the signature was tuned against.
- **Does not** resolve type attribution -- that remains blocked exactly as documented,
  and this node's own type came from `dune.markers`, not from the scanner.

## Raw evidence

- `player-position-and-nearby-markers.txt` -- the live position query and the marker
  proximity query, as run.
- `proximity-scan-attempt.json` -- the failed actor-resolution attempt.
