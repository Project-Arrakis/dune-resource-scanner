# The real storm: what actually happened, and a genuinely new finding about how it works

The Coriolis storm this project spent 2026-08-24 preparing for actually fired. This is
the single most important test this repo could run — everything before it was measured
on one unchanging seed. It confirmed the core hypotheses, and turned up something new
and unexpected in the process.

## Timeline

- Predicted regeneration: 04:00:00 PDT, 2026-08-25 (decoded from the official Discord
  schedule's Unix timestamps, see `sessions/CONTINUATION-PROMPT.md`).
- Actual regeneration: **05:10:01 PDT** (12:10:01 UTC) — about 70 minutes later than
  predicted. The idempotent windowed watcher (checks every 10 min, 03:50-07:59 PDT)
  handled this without any intervention; a single-fixed-time plan would have missed it.
- `storm-watch.sh` detected the DeepDesert seed change (`2` → `3`) on its 15th firing and
  ran `post-storm-scan.sh` automatically. **Both maps regenerated simultaneously** —
  `HaggaBasin`'s seed also moved `2` → `3` — which the project's framing (built entirely
  around "a Coriolis storm regenerates Deep Desert") hadn't anticipated.
- Zero players were online for either regeneration or the scans that followed it.

## 1. The core hypothesis holds: markers reset, the census doesn't need them

DeepDesert: **0 markers** immediately post-storm (confirmed via `COPY ... markers.csv`,
header only, zero data rows) — exactly the "database knows nothing" moment this whole
project exists to work around.

Hagga Basin: markers didn't reset to zero, but collapsed from **6,270+ (end of the prior
session) to 684**, and — this is the clean part — **every one of those 684 survivors
except 22 edge cases (18 `NoIcon`, 4 `HomeBase`) has `long_range = true`**: `EnemyCamp`,
`Cave`, `EnemyOutpost`, `Ecolab`, `Shipwreck`, `Sietch`, `TradingPost`, every
`HouseRepresentative*`/`Trainer*` row. **Zero resource-type markers (Ore/Pickup/Rock/
Bush) survived on either map.** This is a live, first-ever confirmation of something
`findings/README.md` previously only inferred: POI-class markers are genuinely
independent of the storm cycle; every resource discovery is wiped.

The census mechanism itself held up structurally across the regeneration, with zero
players online for the scan:

| | Pre-storm (seed 2) | Post-storm (seed 3) |
|---|---:|---:|
| DeepDesert strict-signature records | 84,559 | 84,130 |
| Hagga (`Survival_1`) strict-signature records | 91,603 | 91,747 |

Both within 0.6% of their pre-storm count. The scan ran, unattended, against a freshly
respawned process (new PID each time, resolved fresh per the existing convention), and
produced the expected order of magnitude on both maps.

**Still open, genuinely blocked, not a tooling gap**: whether census *recall* (records
found near real markers) holds up on the new seed cannot be tested yet — there is no
ground truth. Nobody has explored/discovered anything on DeepDesert's new layout (0
players since the storm). This is the actual next step once someone plays; the tooling
side is done and waiting.

## 2. New finding: DeepDesert's regeneration doesn't move every node — and the two
populations look architecturally different

Comparing the pre-storm baseline (`findings/2026-08-24-storm-watch/pre-storm-baseline/`,
84,559 records, seed 2, byte-exact X/Y/Z floats) against the post-storm census
(84,130 records, seed 3) by exact position:

| | Count | % of pre-storm |
|---|---:|---:|
| **Stable** — identical (X,Y,Z) present in both scans | 32,930 | 38.9% |
| **Reshuffled** — pre-storm position not present post-storm | 51,629 | 61.1% |

This is not floating-point coincidence or a methodology artifact (verified: string-exact
comparison across ~84k records at full float precision; random collision at that
precision across a multi-hundred-thousand-unit coordinate space is effectively
impossible). **Roughly 39% of the map's resource-node memory positions are genuinely
invariant across a storm reseed.**

Spatially, this isn't uniform — three distinct regimes appear in a coarse (500m cell)
breakdown:

- A band of cells (mostly `y` in the 1.15M-1.25M unit range) sits at **90-99.8% stable**
  — looks like a map edge/boundary region outside whatever the storm actually reshuffles.
- A separate, disjoint set of cells sits at **exactly 0% stable** (zero overlap out of
  200-330 records each) — fully reshuffled terrain.
- The broad "core desert" region — most of the map — sits in a **40-60% mixed** band,
  tile by tile.

That mixed middle band is the interesting part. Labeling both populations against the
most recent pre-storm marker snapshot (12,906 DD markers, captured ~15:19 PDT
2026-08-24, nearest-marker match within 1 m):

| | Typed (has a nearby known marker) | Ore/Rock share of typed |
|---|---:|---:|
| Stable records | 2,107 / 32,930 (6.4%) | 1.8% (38 Ore, 0 Rock) |
| Reshuffled records | 7,017 / 51,629 (13.6%) | 12.5% (805 Ore, 73 Rock) |

Reshuffled records are **~2x as likely** to carry a known marker at all, and **~7x** as
likely to be an Ore/Rock type specifically when they do. Both populations are dominated
by `ScrapMetalPart`/`ScrapMetalWreckage`/`FuelCellPart`/generic `Pickup` debris either
way — this isn't "stable = junk, reshuffled = real resources," it's a real skew, not a
clean split.

**What this does and does not establish**: the skew is measured and repeatable-in-
principle (re-runnable against the committed data). **Why** it exists is not established.
Plausible, unverified explanations: the "stable" population could be a genuinely
different actor class that happens to also match the strict spawn-record signature
(static wreckage/clutter placed once at level-design time, not part of the
storm-reseeded seed) rather than false positives; or it could be real resource spawn
slots whose *position* is fixed by level geometry while only their *activation* is
reseeded — and the low marker-match rate would then mean discovered nodes cluster in the
reshuffled/active population for a reason not yet understood, not that the stable slots
are never real. Do not extend either theory further without new evidence — this is
exactly the kind of claim `findings/README.md`'s accuracy audit exists to keep honest.

## What to do next

1. **When DeepDesert gets a real player**, re-run this session's marker-pull +
   `analyse_census.py` against the post-storm census to get the first true recall number
   on a genuinely new seed — the actual "does this survive a real regeneration" answer
   the whole project has been waiting for.
2. **Extend this session's Hagga process-identity fix** (`Survival_1`, not `Overmap`) —
   already applied here; the Hagga post-storm census in this write-up used the correct
   process from the start.
3. **The stable/reshuffled split is worth a second look with fresh eyes**, ideally with
   an in-game check: pick a few "stable" positions and a few "reshuffled" ones, fly to
   both, and see what's actually there. That's a five-minute check that could settle the
   open question above outright.

## Files

- `dd-census-poststorm.jsonl.gz` — the full 84,130-record post-storm DeepDesert census.
- `dd-markers-prestorm-15h19.csv.gz` — the 12,906-marker DD snapshot (captured ~15:19 PDT
  2026-08-24, pre-storm) used to type-label the stable/reshuffled populations above.

`pre-storm-baseline/` (referenced above) already lives in
`findings/2026-08-24-storm-watch/`. Raw Hagga scan output and the intermediate
`post-storm-20260825T121001Z/` working directory remain on `dune-dev:~/scan-findings/`,
not duplicated here.
