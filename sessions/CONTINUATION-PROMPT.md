# Continuation prompt — dune-resource-scanner R&D

Lives at `~/projects/repos/dune-resource-scanner/sessions/CONTINUATION-PROMPT.md`.
Paste it into a fresh session, and **overwrite it in place before that session ends**.

Last rewritten: **2026-08-24**, end of a long session that closed off type attribution for
good and found something more useful instead.

**Scope note**: this prompt is R&D-only — memory scanning, position-finding, validation,
DB-structure investigation. Building the actual live map into the game server's Web Console
is now tracked separately as **Core issue
[`dune-awakening-selfhost-docker#462`](https://github.com/Project-Arrakis/dune-awakening-selfhost-docker/issues/462)**
— a different repo, a different session, DB-only by design, not this repo's concern. Don't
plan Core UI/API work from here.

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
| Zero-player operation | **Proven.** 64.0% coverage with nobody logged in, identical to a with-player scan |
| `dune.actor_spawners`, `fgl_entities`, `actor_state` | **Checked and ruled out** as a hidden resource-type source — 74/1,181/747 rows respectively, none resource-node related. No DB analog to `resourcefield_state` exists for solid resources; this is architectural (spice fields are server-simulated abstractly, ore nodes are static level content), not a gap |

## The single most important open item: does any of this survive a real regeneration?

Everything above — the recall numbers, the census signature, the bulk-area-discovery
mechanism, `area_id` boundaries — was measured on **one long-lived seed (`2`)**, unchanged
since 2026-08-21. Nothing has ever been tested against a genuinely regenerated map.

**A real storm is confirmed, precisely, for tonight**: decoded directly from the official
Discord schedule's Unix timestamps (not a guess) — North America Coriolis storm **ends
(the actual regeneration moment) at exactly 04:00:00 AM PDT, 2026-08-25**. A hardened,
idempotent cron job is already armed on `dune-dev` (`crontab -l`): checks every 10 minutes
from 03:50 through 07:59 PDT, fires `post-storm-scan.sh` the moment the DeepDesert seed
changes, captures markers + a full census. A same-format pre-storm baseline is already
captured and committed (`findings/2026-08-24-storm-watch/pre-storm-baseline/`, seed 2,
10,488 markers, 84,559 records, 2026-08-24T19:57:10Z) specifically to diff against it.

**When this fires, that diff is the actual next task**: does census recall hold up on a
fresh layout? Do `area_id` boundaries and the bulk-reveal mechanism still work the same
way? Check `~/scan-findings/storm-watch.log` on `dune-dev` and whatever
`~/scan-findings/post-storm-*/` directory appears first.

## Smaller open items, still real

- **Hagga Basin Jasmine discovery test — never actually completed.** Was set up
  (`~/scan-findings/re/hagga-markers-before.csv`, `hagga-census-before.jsonl` against PID
  380152, the "Overmap" process that serves Hagga) but the operator got redirected to the
  A4/Ecolab excursion instead, which turned out more valuable. Worth doing if there's a
  quiet moment: Hagga's spawn-record census only ever found **112 total records** for the
  whole map (vs. DD's ~84,500) — strongly suggesting Hagga's resources are ordinary
  pre-placed level actors, not this spawn-record system at all. A Jasmine discovery there
  would test a *different* system than everything else in this document.
- **The `*Pickup` recall gap** (from earlier sessions, not re-verified tonight):
  RhyolitePickup/AzuritePickup recall sat around 67% against 66-89% for ore types — a
  second record shape was suspected as the cause. Not touched tonight; check it's still
  accurate before acting on it.
- **Empty-scan `[]` vs `null`** — Go nil-slice JSON marshalling bug flagged in an earlier
  session, status not re-verified tonight.

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

`ssh dune-dev`, passwordless sudo. **DeepDesert = `DeepDesert_1` (PID resolved fresh each
time via `ps -eo pid=,rss=,args=` matching `DeepDesert_1` and RSS > 100MB — never hardcode,
the process respawns on every regeneration). Hagga Basin = the "Overmap" process** (matches
`farm_state.map = 'Overmap'`, confirmed via DB tonight — not literally named `HaggaBasin` as
a process arg). `gdb` is now installed on `dune-dev` (wasn't at session start). Cross-compile
locally (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`) and `scp` to
`dune-dev:~/scan-findings/bin/`. `dune database sql "<query>"` is the read-only DB
interface — for anything with special characters, write the SQL to a file and pipe it in
(`dune database sql "$(cat file.sql)"`) rather than fighting `ssh` quoting. `dune-dev` is
sanctioned for this work; `dune-prod` is not.

## Repo state as of 2026-08-24, end of session

`main` green, no open PRs. Merged tonight: #37 (pre-storm baseline + a real secret-exposure
fix in `post-storm-scan.sh`), #38 (four-session static/live RE closure), #39 (bulk-area-
discovery finding). `findings/README.md` and the root `README.md` are both current as of
this rewrite — read `findings/README.md` first for the full, maintained index; this prompt
is a working-session summary, that file is the source of truth.
