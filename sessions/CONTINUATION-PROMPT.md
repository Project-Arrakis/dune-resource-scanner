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

1. **[#16](https://github.com/Project-Arrakis/dune-resource-scanner/issues/16) — recall, ~20–40%.**
   The blocker. Already ruled out: broken scanner (a known-good target reproduces at 0.08 m),
   depletion (untouched terrain fails identically), class layout (`StravidiumOre` 0/5 in one box
   but 0.06 m elsewhere), player-proximity streaming, and anything DeepDesert-specific (the Hagga
   process fails the same way). **The defect is in the tool and the failure is per-instance.**
   Leading suspect: pass 2 only finds an actor if a pointer to its `RootComponent` lies within the
   regions `HeapLikeRegions` scans.
   **Start by instrumenting** pass 1 candidates, pass 2 references, and each `ValidateActor`
   rejection stage — observe the loss rather than guessing. Then try widening the region set, and
   relaxing the vtable check to any file-backed executable/rodata mapping (repeating the #8 fix
   one level out). Validate against the WindPass box at `-near=-4368,-198837 -tolerance 15000`,
   which has 200+ markers as ground truth.
2. **[#14](https://github.com/Project-Arrakis/dune-resource-scanner/issues/14) — `ValidTransform`.**
   Reject denormals and `Inf`, **and add a Z bound** — only X and Y are bounded today, which let a
   `Z = 228598` false positive through.
3. **Emit `[]` not `null`** on an empty scan (Go nil-slice marshalling; breaks naive parsers).
4. **Add a class-anchor mode.** Given `resource -> known (X,Y)`, probe each anchor and emit the
   resolved class pointers. Class pointers are per-process and per-restart and cannot be stored;
   anchors make re-derivation automatic. See section 10a.
5. **Consider fast-capture** — dump candidate regions to disk, analyse offline. A 2–5 minute
   inline scan cannot catch a sandworm and probably cannot catch a storm.

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
  2–5 minutes. Two scans were wasted when they travelled mid-scan and the empty results looked
  like real negative findings.
- **Match marker-centric, not actor-centric** — for each marker, ask which class sits on it.
  Nearest-marker-wins produced two confidently-wrong labels.
- **Exclude the operator's own character and controller** — they sit at the player's position and
  match whatever marker is underfoot.
- **Inventory diffing beats scanning for names.** Snapshot `dune.items`, gather one node, diff.
  Order by `acquisition_time`, not `actor_id` — dropped items move to a `BP_LootContainer`.
- **Never infer item names.** Four predictions this session, four wrong. Node name, item name and
  display name are three independent things.
- **Verify pasted reference material.** Third-party docs supplied at the start were right about
  some things (`field_kind_id`, `field_id` packing) and badly wrong about others (marker counts
  off by 20×, event-type meanings, Leaflet calibration constants that match nothing in this
  codebase).
- **Use a remote-side `timeout`** on every scan — a local `timeout ssh` leaves the remote process
  running and CPU-pegged.
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
