# Pre-storm baseline (2026-08-24T19:57:10Z, seed 2)

Captured deliberately, ahead of the expected ~0400 PT 2026-08-25 Coriolis
storm, as a same-format "before" snapshot to diff against whatever
`post-storm-scan.sh` captures once the seed actually changes. Produced by
[`pre-storm-scan.sh`](../pre-storm-scan.sh), the direct counterpart of
[`post-storm-scan.sh`](../post-storm-scan.sh).

- DeepDesert seed at capture time: `2` (the baseline observed continuously
  since 2026-08-21 -- confirmed unchanged, see `seeds.txt`)
- Markers in `dune.markers` for DeepDesert: 10,488 (`markers.csv`)
- Census records: 84,559 strict-signature spawn records (`positions.csv.gz`,
  gzipped `addr,base,x,y,z,strict` -- the raw 67MB `census.jsonl` this was
  extracted from is not committed, matching this repo's existing convention
  for `dd-census-positions.csv.gz` elsewhere in `findings/`)

## What this is for

1. **Prove real regeneration happened.** After the storm, diff post-storm
   `markers.csv`/`positions.csv.gz` against this baseline. If positions are
   genuinely new (not just a re-shuffled seed number with the same layout),
   very little of this file's coordinate data should still match.
2. **Re-run the census recall validation against a fresh map.** Every
   position-recall number in this repo to date (`58.5-64.3%` map-wide) was
   measured against this same long-lived seed. This baseline lets that
   number be recomputed against a genuinely different layout for the first
   time.
3. **Test whether any spatial signal survives regeneration for type
   inference** -- see the note below.

## Type-inference check run against this baseline (negative result)

Before capturing this baseline, an earlier ad hoc probe this session
(`type1.txt`, offset-32 pointer-chase test against a 211-marker subset)
appeared to show a split between "debris" types (`*Part`/`*Wreckage`, up to
332 duplicate matches for a single marker) and ore types (mostly 1-2). That
looked like a possible categorical signal.

Re-tested cleanly against this baseline (all 10,488 markers, plain radial
proximity match at 150uu, not a pointer-chase): **the split does not hold
up.** Every marker type -- ore, pickup, debris, alike -- has a median of
exactly 1 census record within 150uu and a max of 2-3. The earlier 332x
figure was very likely an artifact of the offset-32 probe matching a shared
pointer value across unrelated records, not a real physical-clustering
signal. Recorded here so this specific claim isn't silently repeated as if
it were still live -- it isn't.

**Practical consequence:** there is currently no known signal -- direct
record field, pointer reference, or positional/count clustering -- that
distinguishes marker types from census data alone. A cluster of N census
records near a location cannot today be labeled with a specific resource
type (e.g. "Titanium") with any defensible confidence. See the parent
[`findings/README.md`](../../README.md) for the full status of the
type-attribution problem.
