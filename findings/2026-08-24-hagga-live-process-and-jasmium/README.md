# Hagga Basin's live process is `Survival_1`, not `Overmap` — and the Jasmium
# discovery test the continuation prompt had queued finally ran

Two findings from one live session, both real, both correcting things this repo had
gotten wrong (one this session, one going back to the original validation work):

## 1. `Overmap` is not the process to scan for live Hagga data

The environment notes carried in `sessions/CONTINUATION-PROMPT.md` said "Hagga Basin =
the 'Overmap' process (matches `farm_state.map = 'Overmap'`)". That is real —
`farm_state.map` does say `Overmap` — but it is the wrong process for anything that
needs live, player-populated memory:

```
 server_id  | connected_players |     map      | game_port
------------+--------------------+--------------+-----------
 8fT5...    |                  0 | DeepDesert_1 |      8779
 QLRx...    |                  1 | Survival_1   |      8778
 WXSv...    |                  0 | Overmap      |      8777
```

`SELECT * FROM dune.farm_state;` — the operator's actual character was on
`Survival_1`, which had **1** connected player. `Overmap` had **0**. Scanning `Overmap`
(PID 380152, confirmed via the safe `argv[2]` extraction convention this session
established) returned a static, tiny, geographically clustered set —**112 strict-signature
records, byte-identical across three separate scans taken over 15 minutes** including one
while the operator stood directly on a resource node — because that process was never
running the world the operator was actually in.

Scanning the correct process (`Survival_1`, PID 2772183 today) returned **91,603**
records, the same order of magnitude as DeepDesert's ~84,500 and consistent with the
original `findings/2026-08-24-validation/README.md` cross-map numbers (which, it turns
out, *were* measured against `Survival_1` all along — that document's PID 2772183 was
correctly identified at the time; the confusion this session was entirely about the
`CONTINUATION-PROMPT.md` environment note giving the wrong process name for future
sessions to reach for).

**Fix going forward**: identify the live Hagga process by `connected_players > 0` in
`dune.farm_state`, not by assuming a fixed process/map name. `Overmap` may be a real,
separate, always-near-idle service (unconfirmed what it actually does) — it is not the
gameplay world.

**Safe PID/map-name resolution, used throughout this session**: never print full `ps`
args (carries the live `ServiceAuthToken`, see `sessions/CONTINUATION-PROMPT.md`).
Instead:

```bash
sudo tr '\0' '\n' < /proc/<pid>/cmdline | sed -n '3p'   # argv[2] is the map name
```

## 2. The Jasmium discovery test, finally run

The continuation prompt had this queued as an incomplete open item since an earlier
session redirected to the A4/Ecolab excursion instead. Tonight it actually ran, live,
with the operator discovering a brand-new resource type (`JasmiumOre`/`JasmiumPickup`,
in-game name "Jasmium Crystal", in the "Shoel" sector of Hagga — a radiation zone the
operator had only flown over before, requiring a rad suit to approach on foot) that had
**zero** prior entries anywhere in `dune.markers`.

Before-snapshot: 5,755 Hagga markers, zero Jasmium. After discovery: 6,270 markers, 33 new
(31 `JasmiumOre`, 2 `JasmiumPickup`) — a single bulk `area_id=11` reveal, matching the
`bulk-area-discovery` mechanism already documented.

Matching a census taken **on the correct process, with the operator standing on a node**
against those 33 markers, at the standard 1 m (100 uu) threshold used everywhere else in
this repo:

| Type | Matched | Total | % |
|---|---:|---:|---:|
| JasmiumOre | 2 | 31 | 6.5% |
| JasmiumPickup | 0 | 2 | 0.0% |

That alone reads as "the census barely finds Jasmium" — and would have been reported
that way if the investigation had stopped there. It's the wrong conclusion. Checking the
actual nearest-record distance for **every** Jasmium marker, not just whether it fell
under the standard threshold:

| Threshold | Matched | % |
|---|---:|---:|
| 1 m | 2/33 | 6.1% |
| 5 m | 3/33 | 9.1% |
| 10 m | 10/33 | 30.3% |
| 20 m | 18/33 | 54.5% |
| 50 m | 24/33 | 72.7% |
| 100 m | 24/33 | 72.7% |
| 151 m | **33/33** | **100%** |

**Every single Jasmium node has a census record nearby.** The census genuinely does find
newly-discovered nodes it could not have known about — reconfirming the DD retrospective
test's core claim on a second map, live, for a type that did not exist in the database
five minutes earlier. What's different is precision: most other types match within ~1 m;
Jasmium's nearest matches cluster in the 7-40 m band, with a handful further out to
~110-150 m. Whether that is Jasmium-specific (a different anchor point on a multi-crystal
formation vs. a single ore rock), an artifact of these nodes being freshly revealed by a
bulk-area event rather than long-settled, or something else, is **not established** —
flagging it rather than guessing further tonight.

One relevant fact from the operator, not independently verified via RE: **every other ore
and pickup type is shared between DeepDesert and Hagga — Jasmium is the only ore type
exclusive to Hagga.** If Jasmium's Blueprint class was implemented separately from the
shared cross-map mineral hierarchy (rather than reusing the same
`BP_<Mineral>_[Static|Pickup|Ore]_[A-D]_[Component|Spawner]_C` pattern the shared types
follow), a class-specific difference in where the spawn record's position field sits
relative to the actual node mesh would plausibly explain a consistent tens-of-metres
offset that other, shared types don't show. Plausible, not confirmed — the offsets above
are all that's actually measured.

## What this does and does not change

**Established**: the census mechanism generalizes to a brand-new type discovered live,
mid-session, on a second map — the strongest version of the "does this work post-storm"
question this project can ask without an actual storm. Positional precision is not
uniform across types; a single fixed 1 m threshold understates recall for at least this
one type, possibly others not yet checked this way.

**Not established**: why Jasmium's offset is larger, whether other low-recall types (the
`*Pickup` gap on DD, `ErythriteOre` on Hagga at a similarly low 5.3%) show the same
"actually close, just past a too-strict threshold" pattern if checked the same way. Worth
a look with fresh eyes, not tonight.

**Corrected**: `sessions/CONTINUATION-PROMPT.md`'s Environment section, which named the
wrong process for live Hagga work. Every claim in `findings/2026-08-24-validation/README.md`
about Hagga's 58.5% recall stands unchanged — that work used the right process the whole
time.

## Files

Raw scan output and intermediate CSVs live only on `dune-dev:~/scan-findings/jasmine-test/`
(not copied into this repo — large, superseded-by-this-summary JSONL/CSV working files;
see this project's own "findings are summaries, not data dumps" convention elsewhere in
`findings/README.md`). The `hagga-markers-before.csv`/`hagga-markers-after.csv` timestamps:
before-snapshot 2026-08-24T22:53:32Z, discovery confirmed live in conversation shortly
after, on-node census 2026-08-24T23:02:54Z–23:02:58Z (wrong process) and
2026-08-24T23:03:35Z–23:04:10Z (`Survival_1`, correct).
