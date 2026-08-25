# Continuation prompt — dune-resource-scanner R&D

Lives at `~/projects/repos/dune-resource-scanner/sessions/CONTINUATION-PROMPT.md`.
Paste it into a fresh session, and **overwrite it in place before that session ends**.

Last rewritten: **2026-08-25, ~07:20 PDT, end-of-session pass.** The storm actually fired
overnight (05:10 PDT, 70 min later than the 04:00 prediction) and regenerated **both**
DeepDesert and Hagga simultaneously (not just DD, which this file's framing had assumed),
confirming the project's core hypotheses live for the first time — see
`findings/2026-08-25-storm-regeneration/README.md` and "The single most important open
item" below. Since the storm-fired pass at ~06:25, this session also filed
[`dune-awakening-selfhost-docker#479`](https://github.com/Project-Arrakis/dune-awakening-selfhost-docker/issues/479)
(a design-input handoff on whether the scanner itself, not just DB data, should ever ship
to other operators — explicitly not a decision, see the Scope note) and merged everything
through PR #49. The "Repo state" section at the end of this file is now accurate as of
this pass — the earlier "not yet committed" caveat it carried is resolved.

**Scope note**: this prompt is R&D-only — memory scanning, position-finding, validation,
DB-structure investigation. Building the actual live map into the game server's Web Console
is now tracked separately as **Core issue
[`dune-awakening-selfhost-docker#462`](https://github.com/Project-Arrakis/dune-awakening-selfhost-docker/issues/462)**
— a different repo, a different session, DB-only by design, not this repo's concern. Don't
plan Core UI/API work from here.

**Also tracked separately, filed 2026-08-25**: whether *this scanner itself* (not just
DB data) should ever ship as an operator-facing capability is a genuinely open,
undecided question — **Core issue
[`dune-awakening-selfhost-docker#479`](https://github.com/Project-Arrakis/dune-awakening-selfhost-docker/issues/479)**,
a design-input handoff, not a plan. It exists because #462 explicitly deferred this exact
question rather than answer it. Don't treat #479's filing as a decision to build this —
it isn't one, and this repo's own job (finding out what's knowable and how) is unaffected
either way.

---

## The goal (unchanged)

Build the R&D foundation for a live map of Deep Desert (and Hagga Basin) that works
**immediately after a Coriolis storm regenerates Deep Desert** — the moment `dune.markers`
is emptiest and a database-only approach knows the least. This repo's job is finding out
what's knowable and how, not shipping the map itself.

## Where this actually stands — read before planning anything

| Capability | State |
|---|---|
| **Node positions, including undiscovered ones** | **Working.** 58.5–64.3% recall, whole map, 17s DD / 36s Hagga, proven with zero players online |
| **Naming a class from a discovered position** (this class is TitaniumOre) | **Works** — scan a coordinate the DB already labels |
| **Typing an undiscovered census record directly** (memory/code/console) | **Closed, not just blocked.** Four full sessions tried: actor chain, all 48 record offsets, static disassembly, live `gdb` tracing, a full 48-field diff of 372 records while an operator stood next to two of them. All negative. The last two *refute* — not just fail to confirm — the "proximity promotes a record" theory that was the leading hypothesis this morning. Do not re-attempt without genuinely new evidence; see `findings/2026-08-24-static-reverse-engineering/README.md` for the full four-session account before spending any more effort here |
| **Typing via bulk area discovery** | **The real path forward, found by accident tonight.** Discovery reveals an entire `area_id` zone at once (57 total on DD), not per-node — one event revealed 1,975 fully-typed nodes in a single pass. See `findings/2026-08-24-bulk-area-discovery/README.md`. Verified only partially complete per-zone (~29% of a bounding-box's census density in the one zone checked, and that's an over-estimate since the true zone shape is smaller than the box) — not proven to be 100% coverage of a zone, worth a cleaner check if this becomes load-bearing |
| Spice by tier, Flour Sand | Exact from `resourcefield_state` + `field_id` decode — inner ~87% of the map (21-bit packing ceiling; theoretical gap, never observed to actually cause a miss) |
| Named POIs (`Cave`, `Ecolab`, `Shipwreck`, `TaxiService`) | **Complete and confirmed globally known** from `dune.markers` where `long_range=true`, independent of any discovery — re-verified live tonight, not just inferred |
| Zero-player operation | **Proven, including across a real regeneration.** 64.0% coverage with nobody logged in pre-storm; both post-storm scans (DD 84,130 records, Hagga 91,747) also ran unattended with zero players |
| `dune.actor_spawners`, `fgl_entities`, `actor_state` | **Checked and ruled out** as a hidden resource-type source — 74/1,181/747 rows respectively, none resource-node related. No DB analog to `resourcefield_state` exists for solid resources; this is architectural (spice fields are server-simulated abstractly, ore nodes are static level content), not a gap |
| **Survives a real Coriolis regeneration** | **Partially confirmed 2026-08-25.** Markers reset exactly as predicted (DD: 0; Hagga: 6,270→684, every survivor but 22 edge cases `long_range=true`). Census record count held within 0.6% on both maps. **What's NOT yet confirmed: recall on the new layout** — no ground truth exists until a player explores DD's new seed. See below |

## The single most important open item: does recall hold up on the new seed?

**The storm fired at 05:10 PDT, 2026-08-25** — 70 minutes later than the 04:00 prediction;
the windowed watcher (checks every 10 min, 03:50-07:59 PDT) caught it on its 15th firing,
unattended, exactly as designed. **Both DeepDesert and Hagga regenerated simultaneously**
(seed `2` → `3` on both) — this file's framing had assumed only DD would move; that was
wrong, corrected here.

What's confirmed, live, for the first time (previously only measured on one long-lived
seed since 2026-08-21): markers reset to (near-)zero on both maps: DD to literally 0,
Hagga from 6,270+ to 684 survivors, of which every one but 22 edge cases is
`long_range=true` — **zero resource-type markers survived either map's regeneration**.
The census mechanism held structurally with zero players online: DD 84,559→84,130 records
(-0.6%), Hagga 91,603→91,747 (+0.2%), both against freshly-respawned PIDs.

**A genuinely new, unexplained finding turned up while checking this**: ~39% of
DeepDesert's resource-node census positions are byte-identical pre- to post-storm, not
reshuffled by the reseed at all. The reshuffled 61% is ~2x as likely to carry a known
marker and ~7x as likely to be Ore/Rock-typed when it does. Real and measured (string-exact
comparison across 84k records, not a rounding artifact); the *why* is not established —
see `findings/2026-08-25-storm-regeneration/README.md` §2 for the two unverified theories
floated and explicitly not extended further.

**What's still genuinely open, and it's the actual point of this whole project**: does
census *recall* — finding nodes near real markers — hold up on the new seed? This cannot
be answered yet. **No ground truth exists**: DeepDesert has had zero players since the
storm, so nothing has been discovered on the new layout. This is blocked on a player
exploring, not on tooling — the census (84,130 DD records) is already captured and
waiting. **The actual next task, the moment DD gets a player**: pull fresh DD markers,
re-run `findings/2026-08-24-issue-16/tools/analyse_census.py` against
`findings/2026-08-25-storm-regeneration/dd-census-poststorm.jsonl.gz`, and get the first
true post-storm recall number.

## Smaller open items, still real

- ~~**Hagga Basin Jasmine discovery test**~~ — **Done this afternoon**, and it corrected a
  real environment bug in the process. See
  [`findings/2026-08-24-hagga-live-process-and-jasmium/README.md`](../findings/2026-08-24-hagga-live-process-and-jasmium/README.md)
  for the full account. Short version: the operator discovered `JasmiumOre`/`JasmiumPickup`
  ("Jasmium Crystal," a real Ore-type resource, not flora) in Hagga's "Shoel" sector — the
  first genuinely new resource type ever seen there. Scanning it live showed the census
  finds every one of the 33 new nodes (100% within 151 m), just with more positional slop
  than the standard 1 m threshold assumes for this type — a real, useful, positive result,
  not the "112 total records forever" ceiling this file previously worried about. **That
  112-record ceiling turned out to be a red herring**: it came from scanning the wrong
  process (`Overmap`, 0 connected players) instead of the one the operator was actually on
  (`Survival_1`) — see the Environment section below, corrected as a direct result.
- **The `*Pickup` recall gap — re-verified this afternoon, and the earlier characterization
  in this file was misleading.** This file previously said "RhyolitePickup/AzuritePickup
  recall sat around 67% against 66-89% for ore types" and named those two as the anomaly.
  Re-run against fresh data (12,906 markers, up from the 9,601 baseline;
  `findings/2026-08-24-issue-16/tools/analyse_census.py` against a fresh whole-map census +
  a freshly-pulled `dune.markers` snapshot — no live RE, no gdb, doesn't touch the
  type-attribution question closed off below): **Rhyolite/AzuritePickup are not the
  anomaly** — they track their Ore siblings normally (66.6%/67.8% vs 73.3%/77.3% Ore). The
  real, sharp, six-way pattern:

  | Type | Ore/Rock | Pickup |
  |---|---:|---:|
  | Titanium | 69.8% | **22.0%** |
  | Basalt | 70.8% | **25.1%** |
  | Bauxite | 70.2% | **27.8%** |
  | Stravidium | 74.4% | **19.4%** |
  | Dolomite | 89.0% | **25.3%** |
  | Magnetite | 73.8% | **14.0%** |

  Cleanly bimodal, not noise — six types sit at 14-28% against a 70-89% Ore/Rock
  counterpart, and Rhyolite/Azurite are the sole exception. Worth a look if map-building
  ever needs Pickup-type recall specifically, but **not investigated further this
  afternoon** — the obvious next step (why these six and not Rhyolite/Azurite) risks
  wanting the same static/live-RE tools the type-attribution effort already exhausted, and
  this is a recall question, not identity, so don't assume the two investigations need the
  same techniques without thinking it through first.
- ~~**Empty-scan `[]` vs `null`**~~ — **Fixed this afternoon**, PR
  [#42](https://github.com/Project-Arrakis/dune-resource-scanner/pull/42), closes
  [#41](https://github.com/Project-Arrakis/dune-resource-scanner/issues/41). `scanSeeds`/
  `scanProximity` declared their result slice as `var results []result` (one path
  explicitly `return nil`), so a zero-match scan encoded as JSON `null` instead of `[]` —
  confirmed still present, TDD'd (3 new tests, watched them fail for the right reason
  first), fixed, merged after green CI (build/vet/test/shellcheck/gitleaks/semgrep/trivy
  all passed on both the PR and `main` post-merge).

## Working practices that matter (accumulated across sessions, still true)

- **`dune admin teleport` does not work.** Confirmed broken three ways in an earlier
  session. Pick validation targets near where the operator already is; never plan around
  teleporting them.
- **The game server runs inside Docker.** `/proc/<pid>/maps` shows paths as the process's
  own mount namespace sees them, not the host's — reach the real files via
  `/proc/<pid>/root/<path>`, confirmed working this session.
- **A live `gdb` breakpoint must always be bounded and cleanly detached** — send SIGINT to
  the *gdb process itself* (not the target) after a fixed wait, never a hard kill. A hard
  kill skips gdb's own cleanup and can leave an `int3` byte patched into the live game
  server's executable memory, crashing it the next time that code executes. Verify original
  bytes are restored after every run, not assumed. See
  `findings/2026-08-24-static-reverse-engineering/tools/trace-336.gdb` for the exact,
  working pattern.
- **Never capture full `ps ... args` into anything that gets committed.** The game server's
  launch command line carries a live Funcom `ServiceAuthToken`. Use `ps -o pid,etime,rss,comm`
  instead — this was a real bug, found and fixed in `post-storm-scan.sh` this session.
- **A public-facing web search summary is not a primary source.** A fan-site's storm-timing
  claim ("10:00 UTC") was internally inconsistent and imprecise; the official Discord
  schedule's raw Unix timestamps, decoded directly, were the actual authoritative source
  and turned out to validate the existing plan almost exactly. Don't skip straight to
  trusting a search-engine synthesis when a primary source is available.
- **Check the real DB schema before assuming a data source doesn't exist.** The
  `actor_spawners`/`fgl_entities`/`actor_state` check this session (all negative, but
  actually checked, not assumed) is the model — enumerate `information_schema.tables`,
  don't guess.
- **Never write anything a future session needs to `/tmp`.** Hard requirement, stated twice
  this session after being violated. Session notes go in `sessions/`, evidence in
  `findings/<date>-<topic>/`, committed. Persistent scan data lives on
  `dune-dev:~/scan-findings/`, never `/tmp` there either.
- **Every branch gets its own PR, merged only after green CI.** Three PRs went out tonight
  (#37, #38, #39) on independent branches off `main` specifically so they could merge in any
  order without blocking each other — matches this project's own branch discipline.

## Environment

`ssh dune-dev`, passwordless sudo. **DeepDesert = `DeepDesert_1`** (PID resolved fresh each
time via `ps -eo pid=,rss=,args=` matching `DeepDesert_1` and RSS > 100MB — never hardcode,
the process respawns on every regeneration).

**Hagga Basin is NOT the "Overmap" process for live/player-populated scanning — this file
was wrong about that until this afternoon.** `Overmap` is a real process (`farm_state.map =
'Overmap'`) but it runs with **0 connected players** and its memory is static/geographically
clustered, disconnected from whatever the operator is actually doing. The process real
players are on is **`Survival_1`** — confirmed via `dune.farm_state.connected_players > 0`
while the operator stood on a live node, and independently by scanning it (91,603
spawn-record hits, matching DD's scale, vs. `Overmap`'s static 112). **Resolve the live
Hagga PID by querying `farm_state` for the row with `connected_players > 0` and its
`map` column, then matching that map name against the process's `argv[2]`** (see the safe
extraction command below) — don't hardcode `Overmap` or any other name. See
`findings/2026-08-24-hagga-live-process-and-jasmium/README.md` for the full story; this was
found because the earlier `~/scan-findings/re/hagga-*-before.*` snapshots (used for the
first, incomplete Jasmine-test attempt) were also taken against `Overmap`, so they were
never going to show anything useful even if that test had been completed then.

**Never print full `ps ... args`** (carries the live Funcom `ServiceAuthToken` — this bit a
committed script once already this session, then an ad hoc interactive command a second
time). Use `ps -o pid,etime,rss,comm` for a quick liveness check, and for the map-name
argument specifically:

```bash
sudo tr '\0' '\n' < /proc/<pid>/cmdline | sed -n '3p'   # argv[2] is the map name
```

`gdb` is now installed on `dune-dev` (wasn't at session start). Cross-compile
locally (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`) and `scp` to
`dune-dev:~/scan-findings/bin/`. `dune database sql "<query>"` is the read-only DB
interface — for anything with special characters, write the SQL to a file and pipe it in
(`dune database sql "$(cat file.sql)"`) rather than fighting `ssh` quoting; for a CSV
export use `COPY (...) TO STDOUT WITH (FORMAT csv, HEADER true)` inside that query, not a
`--format` flag (the CLI doesn't have one). `dune-dev` is sanctioned for this work;
`dune-prod` is not.

## Repo state as of 2026-08-25, ~07:20 PDT (end-of-session pass)

`main` green, no open PRs, `git log --oneline -1 origin/main` matches this local clone.
Full PR sequence this session (2026-08-24 night through 2026-08-25 morning): #37
(pre-storm baseline + a real secret-exposure fix in `post-storm-scan.sh`), #38
(four-session static/live RE closure), #39 (bulk-area-discovery finding), #42 (empty-scan
`null`-vs-`[]` fix, closes #41), #43 (afternoon touch-up), #45 (Hagga
`Overmap`-vs-`Survival_1` process-identity fix + the Jasmium test, closes #44), #46
(retracted the Jasmium-exclusivity claim after an operator correction — Primrose, not
Jasmium, is Hagga-exclusive), #48 (the real storm firing, closes #47), #49 (links Core
#479, the scanner-packaging design handoff). Every PR went through its own branch,
tracking issue, and green CI before merge, per this repo's own Requirement 21 discipline
— none direct-to-`main`.

`findings/README.md` is current as of this session (storm-regeneration entry + updated
capability-table rows). The root `README.md` was **not checked this session** — verify
before trusting it if it becomes relevant. `CONTINUATION.md` (the older, root-level
"living document" `sessions/README.md` still points to) has **not been touched since
2026-08-24 12:47 PDT** — it predates the type-attribution closure, the storm firing, the
Jasmium test, and the #479 handoff entirely. Flagged, not fixed, this session — it's a
1,300-line file and a real update was judged out of scope for the specific ask that
prompted this rewrite. A future session should either bring it current or fold its
still-useful content (the Live Map product-ask section, the design constraints from the
earlier 8-hats audit) into `findings/README.md`/this file and retire it, rather than
maintain three separate "living" documents indefinitely.

One more real, small thing found this afternoon and worth remembering: a direct `ps -eo
pid=,rss=,args=` grep for the Overmap/DeepDesert PIDs (done to confirm the Hagga process was
up before attempting the Jasmine test) reprinted the full launch command line — including the
live Funcom `ServiceAuthToken` — straight into this session's own conversation transcript.
Nothing was committed, but it's a second instance of the exact class of mistake
`post-storm-scan.sh` was already fixed for tonight, this time in an ad hoc interactive
command rather than a script. `ps -o pid,etime,rss,comm` (no `args`) is the safe form for a
PID lookup that doesn't need the map name argument to disambiguate — reach for that first
even for a one-off check, not just inside committed scripts.
