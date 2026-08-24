# Continuation prompt — Dune Awakening dynamic Live Map

Lives at `~/projects/repos/dune-resource-scanner/sessions/CONTINUATION-PROMPT.md`.
Paste it into a fresh session, and **overwrite it in place before that session ends**.

---

Continue the Dune Awakening Live Map work. **Read
`~/projects/repos/dune-resource-scanner/CONTINUATION.md` first — the Goal, then sections 10 and
11.** Earlier sections predate the real product requirement and reach conclusions it overturns;
the document flags those inline.

## The goal

Build a **dynamic live map of both Deep Desert and Hagga Basin** showing **all resources, POIs,
player buildings, and storm/worm activity** — and critically, one that works **immediately after a
Coriolis storm regenerates Deep Desert**, so players know where to go without re-exploring.

That last clause drives everything. `dune.markers` is discovery-driven and empty right after a
wipe, so a database-only map can only report what someone already found. Reading the live server's
process memory is the only way to see undiscovered nodes.

## Two tracks. Track A blocks Track C.

### Track A — fix the scanner (do this first)

**Session 4 (2026-08-24) found #16's root cause. Read `CONTINUATION.md` Session 4 and
`findings/2026-08-24-issue-16/README.md` before touching this.** The summary:

- Pass 1 finds **211/211 known markers (100%)**. Pass 2 resolves **6/211 (2.8%)**.
  Positions were never the problem — the missing output is a resource *type*.
- The hypothesis this issue was opened with (back-references outside the scanned
  regions) is **disproved**. Offset-agnostically, **nothing points into the 2 KB before
  these nodes' transforms**, so widening the region set cannot help.
- Ore/scrap/pickup nodes are **384-byte spawn records in an array**, not UObject actors.
  Spice and flour sand are unaffected — seed mode reaches them through a real actor.
- Filtering on the spawn-record signature gives **152/211 (72%)** at 1–2 copies per
  marker — a **25× improvement** with no actor chain at all.

**Map-wide follow-up:** the signature approach now scans the whole world in **17 s at
136 MB** and hits **64.8% marker coverage (5,144 / 7,934)** vs the shipped 2.8% — ore and
rock 66–89%, small pickups 14–30%, POIs 0% (expected; they are streamed). Four
type-attribution routes are ruled out — the actor chain, all 48 record offsets, the object
`+0` points at, and memory-address clustering. **The 80,803 unmatched records are not all
undiscovered nodes** (median Z 18,058 vs 3,715; 5.5x the markers even in the best-explored
cell) — do not present the census as a complete map.

1. **[#16](https://github.com/Project-Arrakis/dune-resource-scanner/issues/16) — still the
   blocker, but it is a redesign, not a bug fix.** Next steps, cheapest first:
   (a) relax the spawn-record signature (the `+8 == 0x0000000100000001` constant is
   probably over-strict) and re-measure against the same 211-marker ground truth;
   (b) follow the record's `+0` heap pointer and inspect what it points at, rather than
   scoring it blind; (c) test whether one of the EXE pointers at record `+280/+320/+336`
   identifies the type — those are the only module-relative values in the record, so a hit
   there would also be **stable across restarts** and would solve the ASLR/anchor problem.
   The harness for all three is in `findings/2026-08-24-issue-16/tools/` (`//go:build ignore`;
   run with `go run`). Ground truth box: `-near=-4368,-198837 -tolerance 15000`.
2. ~~#14 `ValidTransform`~~ — **done, PR #17** (denormals, `Inf`, Z bound). Note the Z bound
   is `WorldBound`, so it does **not** reject the `Z = 228598` hit; that was deliberate.
3. **Emit `[]` not `null`** on an empty scan (Go nil-slice marshalling; breaks naive parsers).
   **Still open.**
4. **Class-anchor mode** — reconsider. Anchors were meant to re-derive `ClassPrivate` after a
   restart, but ore nodes have no reachable `ClassPrivate`. If 1(c) works, anchors may be
   unnecessary for them.
5. **Consider fast-capture** — dump candidate regions to disk, analyse offline. A 2–5 minute
   inline scan cannot catch a sandworm and probably cannot catch a storm.

**Also fixed in session 4: [#18](https://github.com/Project-Arrakis/dune-resource-scanner/issues/18)**
— `FindNearbyXY` accepted NaN pairs (both guards were negations; every NaN comparison is
false), producing ~17.4M of 17.4M hits in a scan. Fixed in PR #19: 623k hits, coverage
unchanged at 211/211, peak RSS 12.6 GB → 165 MB.

**PRs #17 and #19 are green on all four required checks but NOT merged** — the previous
session's tooling declined the merge action. Merge them before building on `main`.

### Track B — data gathering

| Layer | Source | State |
|---|---|---|
| Spice (3 tiers), Flour Sand | `resourcefield_state` + `field_id` decode | **Ready** — exact, live, both maps |
| Ore / stone / flora nodes | scanner + class→name | **Blocked on #16** |
| Named POIs | `dune.markers` | **Ready**, discovery-limited |
| Bases, storage, vehicles, players | existing `liveMap*` in Core | **Already shipped** |
| Dropped loot | `dune.actors` `BP_LootContainer` | **Ready**, unused |
| Hagga region labels | marker `DisplayName` | **Ready** — normalise spelling |
| DD grid labels | 9×9 @ 270,000 uu, origin (−52656, −52066) | **Ready** — verified on 5 cells |
| Map regeneration signal | `debug_get_coriolis_seeds()` | **Ready** |
| Sandstorm per-structure | `game_events` type 13 | **Partial** — only where a player owns a structure |
| **Active storm position** | — | **Open**, memory only, never attempted |
| **Sandworm activity** | — | **Open**, absent from DB, checked live twice |
| Hidden treasure | — | **Open**, not in DB, three empty scans |
| Primrose water | — | **Unreachable** — not an item, not persisted |

Remaining identification work:

- **Hagga pass** — **Jasmium** (its only home, last of the 15 raw resources); whether Titanium,
  Stravidium, Bauxite, Dolomite and Magnetite exist there at all (zero markers now, but Basalt
  appeared the instant the operator walked past one, so absence is probably discovery);
  the two missing regions (`Graben` and `RedD` expected); and the **permanent anchor set**, which
  never expires because Hagga is authored terrain.
- **`Dolomite*` (Carbon Ore)** — the only DD item name still genuinely unknown.
- **Impure Fuel** — unidentified, no known node type. `FuelCellPart` yields `Oil`, which displays
  as "Fuel Cell", so Impure Fuel is something else.
- **Storm and worm** — both memory-only and fast-moving. Do not attempt with the current inline
  scanner; revisit after Track A item 5.

### Track C — build the Live Map

Only after Track A. No Core code exists yet. Constraints from the earlier eight-hats audit stand:
no continuous privileged sidecar, scoped read-only DB role, console API reads a read-only
bind-mounted scanner output file rather than talking to the scanner. Core's Live Map already
exists (`console/web/src/features/liveMap/LiveMapPanel.tsx`, `console/api/src/duneDb.js`
`liveMap*`, routes at `server.js:1148-1157`) and uses `LIVE_MAP_CONFIGS` min/max bounds
normalisation on 4096×4096 images — **reuse it; do not introduce a second calibration scheme.**
Known bug to fix while there: `liveMapBases()` filters `coalesce(a.partition_id,0) > 0`, so a base
whose instance despawned silently vanishes from the map.

## Working practices that matter

- **Operator must be stationary before any scan.** The scanner reads `-near` at launch and runs
  2–5 minutes. Two scans were wasted when the operator travelled mid-scan and the empty results
  looked like real negative findings.
- **Match marker-centric, not actor-centric** — for each marker, ask which class sits on it.
  Nearest-marker-wins produced two confidently-wrong labels.
- **Exclude the operator's own character and controller** — they sit at the player's position and
  match whatever marker is underfoot.
- **Inventory diffing beats scanning for names.** Snapshot `dune.items`, gather one node, diff.
  Order by `acquisition_time`, not `actor_id` — dropped items move to a `BP_LootContainer`.
- **Never infer item names.** Four predictions in the last session, four wrong. Node name, item name and
  display name are three independent things.
- **Verify pasted reference material.** Third-party docs supplied at the start were right about
  some things (`field_kind_id`, `field_id` packing) and badly wrong about others (marker counts
  off by 20×, event-type meanings, Leaflet calibration constants that match nothing in this
  codebase).
- **Use a remote-side `timeout`** on every scan — a local `timeout ssh` leaves the remote process
  running and CPU-pegged. **Bound the working set inside the tool too**: a wall-clock timeout does
  not stop a diagnostic from allocating its way toward an OOM on a host that also runs the live
  game server. Session 4 had to kill one manually and re-verify the game PIDs afterwards.
- **A scoring metric can look like a result.** Session 4's first type-discriminator search
  reported a confident `pure = 15` that was pure artifact (`0x0` satisfies "all markers of this
  type share a value"). Check what a metric actually rewards before believing its ranking.
- **Never write session summaries or continuation prompts to `/tmp`.** They go in the repository,
  under `sessions/` — see [`README.md`](README.md) for the two file types and their lifecycles.
  `/tmp` is `tmpfs` on these hosts, so anything a future session needs to find would not survive a
  reboot. Genuine throwaway scratch (a scan's raw JSON, an intermediate diff) is still fine there;
  a handoff note is not.

## Environment

`ssh dune-dev`, passwordless sudo. **DeepDesert = `DeepDesert_1` process, Hagga = `Survival_1`** —
find PIDs with `ps aux | grep DuneSandboxServer`. Deploy by cross-compiling locally
(`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`) and `scp` to `/tmp/dune-resource-scanner`.
`dune database sql "<query>"` is the read-only DB interface; for complex SQL write a file and
`scp` it rather than fighting shell quoting. `dune-dev` is sanctioned for this work; `dune-prod`
is not.

Open PR: [#15](https://github.com/Project-Arrakis/dune-resource-scanner/pull/15) (docs, CI green).
Open issues: **#16 (recall — the blocker)**, #14 (`ValidTransform`), #1 (v1 tracking).
