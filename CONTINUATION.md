# Continuation: Dune Awakening raw-resource discovery → Live Map integration

## Goal

**Produce a complete map of Deep Desert's resources and POIs immediately after a Coriolis storm
regenerates it**, so players know where to go for resources, ships, spice and stations without
having to re-explore the whole map first. That is the product requirement everything else serves.

The consequence, stated up front because it invalidates the obvious design: the game's own
`dune.markers` table is **discovery-driven** and is empty right after a wipe, so a database-backed
map can only ever report what somebody already found. **Reading the live server's process memory
is the only way to see undiscovered nodes**, which makes `dune-resource-scanner` — and
specifically its recall bug,
[#16](https://github.com/Project-Arrakis/dune-resource-scanner/issues/16) — the critical path.
See section 10 for the full reasoning and section 11 for the current next steps.

Delivery target is the Live Map feature of `Project-Arrakis/dune-awakening-selfhost-docker`
(some older references say `yacketrj/*`; `git remote -v` confirms the org is **Project-Arrakis**).
No Core code has been written yet — all work so far is investigation. The scanner is a permanent
Go tool, clean-room, replacing a Python prototype that hit a real performance wall.

**Read section 10 before anything else if you are picking this up fresh** — several earlier
sections were written before the post-storm requirement was stated and reach conclusions it
overturns.

**Where the paperwork lives.** This file is the single living document — every claim in it is
kept current, and corrections are made in place. Alongside it, [`sessions/`](sessions/) holds the
handoff prompt for the next session (`sessions/CONTINUATION-PROMPT.md`, overwritten each time)
and dated, append-only point-in-time records (`sessions/YYYY-MM-DD-findings.md`). Raw scan output
and captures go in [`findings/`](findings/). **None of these ever go in `/tmp`** — it is `tmpfs`
on these hosts, so anything a future session needs to find would not survive a reboot.

## Session 2 update (2026-08-21, later same day) — v1 core implementation shipped
- Repo created for real: **`Project-Arrakis/dune-resource-scanner`, private** (user confirmed
  this org/visibility explicitly this session). `origin` on the local clone already points at
  it; `main` branch (not `master`).
- **`internal/memscan` fully implemented via TDD** (18 tests, all green): `maps.go` (`/proc/<pid>/maps`
  parsing + `Region`/`FilterExecutable`/`FilterByPathname`), `validate.go` (`ValidTransform` —
  NaN/out-of-world/exact-origin rejection), `scan.go` (`FindInt32LE` seeded scan, `FindNearbyXY`
  proximity scan, `FindPointerReferences` backward pointer resolution), `actor.go`
  (`ValidateActor` — the full vtable/ClassPrivate/RootComponent/Transform chain, tested against a
  fake in-memory `MemReader`, no live process needed), `procmem.go` (`ProcMem` implementing
  `MemReader` over any `io.ReaderAt`, `ReadRegion` helper). All clean-room, matches the design in
  "What the Go tool needs to do" below exactly — never read the Python prototype's source.
- **`cmd/dune-resource-scanner/main.go`**: CLI wiring, `-mode seed|proximity`, `-seeds
  label=value,...`, `-near x,y`, `-tolerance`, JSON output (`label`/`value`/position/etc.).
  Deliberately scoped as untested "thin glue" (declared as such in issue #1 before writing it) —
  verify this manually against `dune-dev`, don't assume it's correct from the tests alone.
- **CI wired and verified green**, both on the PR and on `main` post-merge (Requirement 1/11):
  `go build`/`go vet`/`gofmt`/`go test -race -cover`, plus `Project-Arrakis/.github`'s
  `reusable-security-scan.yml@main` (gitleaks/semgrep/trivy — confirmed merged to that repo's
  `main` this session, superseding the "still unmerged" note that used to be in the meta README).
  Branch protection on `main` requires all four checks + no force-push/deletion. Dependabot
  (gomod + github-actions) enabled.
- gitleaks/semgrep/trivy all ran clean **locally** before every commit, not just in CI.
- **Cross-compiled successfully**: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` produces a
  static ELF binary, `-help` output confirms flags wire up correctly. **Not yet copied to or run
  against `dune-dev`** — that's the actual next step, see below.
- Issue #1 tracks this work and is **intentionally still open** — it covers live verification
  too, not just the code landing. PR #2 (merged) says as much in its body.
- Also fixed, same session: a stale claim in the `meta` README saying `Project-Arrakis/.github`
  PR #1 (the shared security-scan workflow) was still unmerged and every adopting repo pointed
  at the source branch as a workaround — both were false by the time this was checked (PR #1
  merged 2026-08-21T22:11:45Z, and all 7 adopting repos, verified directly via the GitHub API,
  already reference `@main`). Requirement 14 fix, pushed directly to `meta`'s `main` per that
  repo's own small-doc-change exception.

## Session 2, part 2 (same day) — live-tested against dune-dev, island survey done

The Go tool went through 4 real bugs found and fixed live against dune-dev's DeepDesert
server (PID 3693040 at the time, will differ next time — find fresh via `ps aux | grep -i
DuneSandboxServer.*DeepDesert`), each fixed via TDD, verified with a standalone Python
diagnostic against real memory *before* trusting the Go rewrite, then shipped through the
normal branch/PR/CI/merge cycle (issues #1, PRs #6/#7/#8/#9 — see repo history):

1. **`[heap]` alone is ~3MB; real allocations live in dozens of anonymous rw mmap regions up
   to 4GB each** (~16-20GB total, confirmed via `/proc/<pid>/maps`). Fixed:
   `memscan.HeapLikeRegions` — `[heap]` plus every anonymous writable region.
2. **`scanProximity`'s original one-scan-per-hit design was O(hits × heap size)** — fine for
   one point, catastrophic for a wide island survey. Fixed: `memscan.FindPointerReferencesMulti`
   + a rewritten `scanProximity` that does exactly two streaming heap passes total, independent
   of hit count.
3. **The vtable-validity check only matched the executable-permission (`r-xp`) segment of the
   main binary, but relocated vtables/RTTI live in its separate `r--p` rodata segment** —
   confirmed directly (`sudo grep DuneSandboxServer-Linux-Shipping /proc/<pid>/maps` shows 3
   segments: r-xp text, r--p rodata, rw-p data, at 3 different address ranges). This was the
   root cause of the first live run validating **zero** actors out of 2606 raw byte-pattern
   hits, despite the pattern search itself being correct (independently re-verified with a raw
   Python count: 2606 aligned hits for `int32(5000)` existed all along). Fixed:
   `memscan.MainModuleRegions` — every segment sharing the main executable's backing file,
   regardless of permission bits.
4. **`ActorInfo` never exposed `ClassPrivate`**, needed to group found actors by real class once
   position alone wasn't enough to distinguish resource-node types from base-building parts.
   Added it — it was already being read during validation, just never returned.

**Seed mode is now fully live-validated**: `spice-small=281, spice-medium=89, spice-large=6,
mystery=533`. The `mystery=533` figure is an exact match to the "533-position pool" the prior
session found via a completely different method (DB query), independently confirming both the
Go rewrite's correctness and that session's finding.

**Proximity mode found the base island's actor layout — and the tool's identification was
independently proven correct against known ground truth, but full identity resolution for
unknown classes hit a genuine, honestly-reported wall.** `FindNearbyXY`'s `tolerance` is a
per-axis box half-width, not a Euclidean radius (`-tolerance 50000` means a
100,000×100,000-unit search box centered on `-near`, not a 50,000-unit-radius circle) — keep
this in mind when picking a value.

**Ground-truth validation (the important, solid result):** re-running seed mode with the
now-current binary (after `ClassPrivate` was added) gave the *actual* class pointers for the
100%-DB-confirmed classes: `spice-small=0x75c26017b270`, `spice-medium=0x75c25f246970`,
`spice-large=0x75c236912500`, `mystery=0x75c25f247290`. Cross-checking these against the wide
proximity scan's class groups found **exact address matches** for `spice-small` and `mystery`
among the "natural-looking" clusters the scan had already flagged — independent proof that
proximity-scan + `ClassPrivate`-clustering correctly identifies real, distinct actor classes.
This is the strongest evidence yet that the whole tool (maps parsing, actor validation, class
grouping) is sound.

**A methodological correction, found and fixed by testing the assumption, not by trusting it:**
an earlier version of this section flagged several large classes (34-75 members) as "almost
certainly base-building components" because some instances had `Z` exactly `0.0`. Re-checked
directly: those classes' *closest* member to the base is 11,000+ units away — far outside any
plausible building footprint — which falsifies that claim outright (a real base-building class,
separately confirmed via a genuinely tight `-tolerance 3000` scan, has ~13 distinct classes each
with only 1-2 members, none overlapping the large classes at all). The `Z=0.0` instances are
more likely depleted/respawning resource nodes reset to a default transform, not proof of
anything about their real nature. **Lesson for whoever picks this up next: don't trust a
classification heuristic that hasn't been checked against its own stated claim (here, "tightly
bound to the base's footprint" was asserted without ever computing the actual minimum
distance) — this is exactly the kind of unverified claim Requirement 12 exists to catch, applied
to a live investigation instead of a doc.**

**Honest current state: 2 of ~28 distinct classes near the island are proven-identified (spice,
mystery); the rest are NOT.** After excluding the 4 known spice/mystery class pointers, **26
distinct classes with 3+ members remain unidentified** near the base's island (full data:
`findings/2026-08-21-base-island-survey.json` in this repo — positions, counts, class pointers
for all of them). The single strongest remaining candidate is still `ClassPrivate
0x75c268f7ade0` — 28 members within a clean, isolated distance band (8,166-48,680 units, then an
~18,500-unit gap before the next group at 67k+, i.e. other islands) matching the user's "at
least 20, if not more" Titanium Ore count and showing natural (non-grid) spatial/elevation
spread — but this is still circumstantial, not proven. **A real, bounded attempt was made to go
further and definitively resolve identity:**
- Checked `dune database tables` for anything that might track ore-node positions/types
  server-side: `actor_spawners` (only PlayerStart/encounter spawners, not resource nodes) and
  `fgl_entities` (a real per-entity JSONB component store, but only for player-persisted objects
  — bases, vehicles, characters; only 1175 rows total, no resource-node components at all).
  Neither helps — confirms the prior session's finding that ore-type resources genuinely have
  zero server-side tracking, memory scanning is the only path to their positions.
- Considered resolving the UClass's real name from its `ClassPrivate` pointer (UE's
  FName/global-name-table format) — **deliberately not attempted**: this requires guessing
  multiple unknown, version-specific implementation details (the `UObject::NamePrivate` field
  offset, the `FNamePool` block/offset encoding scheme) with no way to verify a guess is correct
  against ground truth we don't have, meaning a wrong guess would produce a *confidently wrong*
  identification — strictly worse than the current, honestly-labeled "unconfirmed candidate."
  This is a real R&D task for a future session with more time to spend verifying each assumption
  empirically, not something to rush.

**The reliable, fast path to full identification is still a short in-game visual check** — the
closest Titanium candidate is only ~8,166 units from the base (X=-612311.21, Y=-708329.60,
Z=4396.67). Once even one class is visually confirmed, the same seed/proximity + class-pointer
methodology already proven against spice/mystery can immediately attach a real name to it and,
from there, likely several of the remaining 25 classes at once (many share a pointer-neighborhood
prefix with the Titanium candidate — e.g. `0x75c268f76xxx`/`f73xxx`/`f74xxx`/`f71xxx`/`f7cxxx` —
suggesting they're sibling actor roles of the *same* resource family, spawner/pickup/component,
per the `BP_<Mineral>_Spawner`/`BP_<Mineral>_Pickup_[A/B]_Spawner`/`BP_<Mineral>_Component`
pattern found in session 1 — not 25 independent resource types).

## Session 3 (2026-08-22) — "mystery" identified, `field_id` decoded, ore classes named

**Read this section before acting on anything above it — several session-2 conclusions are
superseded here.** Everything below was verified live against `dune-dev` (PID 1270301,
DeepDesert_1 partition 8) in the same session it was written.

### 1. `field_kind_id = 0` is **Flour Sand** — CONFIRMED, twice, independently

- Operator stood **4.5 m** from the memory-scanned position `(-638575, -536975)` with flour sand
  directly beneath them.
- `dune.markers` contains a `FlourSand` row at **exactly** that coordinate (Z 1446 vs scan 1447).

The session-2 note above that "ruled out" a spice relationship was rejecting a *different*
hypothesis — "inactive spice that flips to `field_kind_id=1`" — and remains correct on its own
terms. Flour Sand being its own separate resource is consistent with all four of its points
(distinct class pointer, constant non-spice value, own active/dormant cycle, comparable
population). Stop calling this "mystery".

### 2. `field_id` is a packed 21-bit coordinate triple — the single most useful finding

```
x =  id        & 0x1FFFFF
y = (id >> 21) & 0x1FFFFF
z = (id >> 42) & 0x1FFFFF
for each: if v > 0xFFFFF: v -= 0x200000
```

Raw decoded values are world units — **no scaling**. Verified against memory-scan ground truth:
**spice 53/57 exact XY matches, flour sand 45/59**.

**Consequence: active spice and flour-sand positions can be read straight from Postgres.** The
memory scanner is *not* required on the Live Map's data path. Decode from string/BigInt —
`field_id` exceeds `Number.MAX_SAFE_INTEGER` and a JS `Number` silently loses low bits.

**Limit found 2026-08-24: the packing cannot represent the whole map.** Three 21-bit signed
fields give a range of -1,048,576 .. +1,048,575, but Deep Desert extends to about ±1.27M --
**1,237 of 9,601 real DD markers (12.9%) lie beyond it**, and bit 63 is unused (`0` in all 141
rows), so there is no escape flag. This layer is therefore only demonstrated for the inner
~87% of the map. Reported by another developer as the cause of decode failures; **that part
did not reproduce** -- of 19 DD rows whose decode misses memory, **none** is near the limit
(largest magnitude 803,675), while the most extreme value in the whole set, 1,044,975, decodes
correctly, and un-aliasing by ±2^21 rescues none. The misses' decoded Z distribution is
indistinguishable from the matches', so the decode is working and those fields are simply
absent from the memory scan -- the same under-count already recorded in section 8. See
`findings/2026-08-24-field-id-21bit/`.

### 3. Resource positions are static per map and identical across instances

Full set diff of two independent DeepDesert processes (partition 32 vs partition 8):
flour sand 533/533, spice-small 281/281, spice-medium 89/89 — **zero differences**. A single scan
is therefore valid for every instance, and does not go stale when an instance respawns.

### 4. Memory holds candidate pools; the DB holds the active subset

| Map | Flour Sand active | Spice active | Flour in memory | Spice in memory |
|---|---|---|---|---|
| DeepDesert | 59 | 56 | 533 | 376 |
| HaggaBasin | 17 | 4 | not scanned | not scanned |

This is what makes the operator's "solid = active, faded = inactive candidate" product ask
implementable. Flour sand candidates sit on a strict **6,350 UU (63.5 m) grid** (all 277 distinct
X and 247 distinct Y values are exact multiples apart) with terrain-following Z; spice is
free-placed (GCD 25) and **never** shares a position with flour sand (0 of 376 coincide).

### 5. `dune.markers` — a named POI/ore atlas, but **discovery-driven, not complete**

1,251 rows (DeepDesert 440, HaggaBasin 752), composite type `(type, x, y, z)`, joinable via
`dune.map_names`. Contains real named ore veins, pickups, scrap, shipwrecks, caves, sietches,
vendors, trainers and hazards. `dune.player_markers` (2,279 rows) tracks per-player discovery.

**Critical caveat — this table is populated live as players explore, and is never complete.**
Demonstrated directly this session rather than inferred: DeepDesert marker count went from **440
to 537** (and the whole table from 1,251 to 1,358) *during a single session* while the operator
flew around one grid cell. Mapping markers onto the 9×9 grid before and after:

| Cell | discovered (before → after) | long_range |
|---|---|---|
| F-4 | **0 → 87** | 1 |
| E-4 | 0 → 7 | 0 |
| F-5 | 0 → 2 | 0 |
| G-3 | 184 | 0 |
| A-2 | 171 | 5 |
| D-1 | 45 | 1 |
| H-3 | 31 | 0 |
| A-5, A-8 | 0 | 1 each |

The mechanic is the **`long_range` boolean column**, not geography:
- `long_range = true` (TaxiService ×3, Ecolab ×3, Cave ×2, Shipwreck ×1) — present without ever
  visiting. This is why cells the operator has never entered (A-5, A-8) still hold a TaxiService.
- `long_range = false` (all 527 ore/pickup/scrap/bush rows) — appear only once discovered up close.

**Never present `dune.markers` to players as an exhaustive atlas.**

### 5b. `area_id` decodes Deep Desert's prefab layout — and locates Imperial Testing Stations

`dune.markers.area_id` is a **foreign key into `dune.map_names.map_name_id`**, and it reveals that
Deep Desert is assembled from prefab "content block" islands, roughly one per 9×9 grid cell:

| Cell | `area_id` | Prefab (`map_names`) | Markers |
|---|---|---|---|
| F-4 / F-5 | 30 | **ClosedOffTestingStationIsland** | 92 |
| G-3 | 26 | FallenLight | 180 |
| A-2 | 11 | HaggaBasin | 175 |
| D-1 | 17 | ElectricityDungeon | 46 |
| H-3 | 46 | *(no map_names row)* | 31 |
| E-4 / F-4 | 41 | *(no map_names row)* | 19 |
| A-1 | 13 | ErythriteCaveIsland | 1 |
| A-8 | 2 | WaterFat | 1 |
| A-5 | 65 | *(no map_names row)* | 1 |

**An Imperial Testing Station is a cell whose prefab is `ClosedOffTestingStationIsland`.**
Confirmed live: the operator stood inside a testing station and their position resolved to the
`area_id=30` region, which also contains the only DeepDesert `Ecolab` marker in that area
(`-201473, -214617`, `area_radius=30000`). In DB terms a station presents as an `Ecolab` marker
inside an `area_id=30` region.

Testing stations have **no dedicated marker type** — searching all 88 marker types across every
map returns nothing station-like — and no row in `encounters_static` (empty) or any table/column
matching `%station%`/`%testing%`/`%facility%`. Only one station is currently discovered; the
operator reports there should be roughly ten (about four active, the rest present but
inaccessible), which the discovery mechanic above fully explains.

In memory a station is unmistakable **while a player is inside it**: a scan centred on the
operator standing in one returned 1,640 actors in a 120 m box, **1,505 of them underground**
(Z ≈ −2,200) across 241 classes. The two dominant classes (`0x7cff8a0c1380`, `0x7cff4ab11bc0`) are
**generic structural/ruin pieces** also found around Shipwreck and Ecolab markers elsewhere, so
they identify "a built interior", not a station specifically.

### 5c. Structure geometry is streamed; resource actors are resident — tested, not assumed

An idea proposed and then **disproven the same session**: enumerate every testing station by
scanning the whole map for underground (Z < 0) clusters. It does not work, and the reason matters
for anything built on the scanner.

A human-produced map places DeepDesert stations at D-4, F-4/5, D-5/6 and F-7/8. Our one
DB-discovered station (`area_id=30`) is in F-4/5, matching that map exactly — good independent
validation of the grid calibration. A targeted scan of the **D-4** cell (2.7 km box) was then run
to see whether an *undiscovered* station shows up in memory:

| | D-4 (no player present) | F-4 station (player inside) |
|---|---|---|
| Total actors | 863 | 1,640 |
| **Named resource actors** | **39** | 6 |
| **Underground structural actors** | **0** | **1,505** |

The inversion is the finding:

- **Resource actors are resident map-wide with no player anywhere near.** D-4 returned 10
  TitaniumOre, 10 StravidiumOre, 10 ScrapMetalWreckage, 6 BauxiteOre and 3 RhyoliteOre from
  ~27 km away, all identified by class pointer. **Full-map resource mapping via the scanner
  works.**
- **Structure geometry is streamed per-player and is absent otherwise.** The station interior
  existed in memory only because the operator was standing in it. D-4's station, if it has one,
  simply is not loaded.

**Consequence: the scanner cannot enumerate unoccupied structures.** Stations, caves, ecolabs and
shipwrecks are findable only via the DB's discovery mechanic or by a player physically going
there. Do not build a POI layer that claims completeness. Resource layers have no such limit.

(The 2 "underground" hits D-4 did return sit at exactly X=-304800, Y=101600 with an identical Z —
the same grid-round signature as the bogus `spice-large` hits, i.e. false positives, not a
structure. Also note D-4's 10/10/10 resource counts are suspiciously round and were not
cross-checked against markers; treat per-cell counts as unverified until they are.)

Related unexplored lead: `dune.actor_spawners` holds 14 `CB_WreckedShip_Medium_001` entries for
DeepDesert (10 in `dimension_index=0`, 4 in `dimension_index=1`, the latter a strict subset)
against only 1 discovered `Shipwreck` marker — so that table may hold a *complete*,
discovery-independent encounter list. It has no position columns, and its `id` is a small
sequential integer (not packed coordinates like `field_id`), so positions would have to come from
elsewhere.

**The architecture this implies — use both sources for what each is good at:**
the scanner yields *complete* positions but no names; `dune.markers` yields *names* but only where
explored. Identify a **class pointer** once from any single discovered marker (section 6), and
every instance of that class map-wide inherits the name — **including in cells nobody has ever
visited**. This was demonstrated directly: a scan of F-4 found 7 TitaniumOre, 6 StravidiumOre,
8 ScrapMetalWreckage, 3 BauxiteOre, 1 Shipwreck and 1 RhyoliteOre **by class pointer**, at a
moment when `dune.markers` still listed zero markers for that cell.

Semantics (confirmed by operator): `*Ore` = the minable vein; `*Part`/`*Pickup` = the small
hand-collected chunks.

### 6. Class pointer → resource name, established without any in-game visual check

Method: scan a coordinate that `dune.markers` *already labels*, and whatever class appears there
is that resource. Sub-metre position matching makes this unambiguous. **This supersedes the
session-2 plan of walking to candidates in-game, and makes the FName/`FNamePool` reverse
engineering unnecessary** — do not spend time on that.

| Class pointer (PID 1270301, ASLR-specific) | Resource | Best match | Confirmations |
|---|---|---|---|
| `0x7cfef4325730` | TitaniumOre | 0.08 m | 3 |
| `0x7cfef432bb90` | StravidiumOre | 0.06 m | 2 |
| `0x7cfef55052a0` | BauxiteOre | 0.38 m | 1 |
| `0x7cfeef5b0080` | RhyoliteOre | 0.25 m | 1 |
| `0x7cfc3f390700`, `0x7cfc44d23730` | RhyolitePickup | 0.04 m | 2 |
| `0x7cfc32810cd0`, `0x7cfc328120b0` | ScrapMetalPart | 0.27 m | 2 |
| `0x7cfef23bb270`, `0x7cfef23ba030`, `0x7cfef23b2500` | ScrapMetalWreckage | 0.05 m | 3 |
| `0x7cff4a976b80` | **generic long-range POI beacon** — *not* a single resource, see below | 0.00 m | 3 |

**Correction, found the same session:** `0x7cff4a976b80` was initially recorded as "Shipwreck" on a
0.00 m match. That was wrong — a nearest-marker-wins match hid a genuine ambiguity. Enumerating
all three of its instances shows one sits on a `Shipwreck` marker and **two sit on `Ecolab`
markers**, all at 0.00 m. It is a class common to `long_range=true` discoverable POIs
(Shipwreck / Ecolab / Cave), not a resource type. **Lesson for anyone extending the table: check
every instance of a class against the marker set, not just its single closest match** — a class
present at several *different* marker types is a generic role, and reporting its best match as an
identification is exactly the kind of confidently-wrong labelling this whole approach exists to
avoid.

Pointers are per-process (ASLR) — re-derive them after any restart; the *method* is what carries
over. Families cluster by memory pool (`0x7cfef432****` = Ore types, `0x7cfef23b****` = Scrap
Wreckage), confirming session 2's sibling-actor-role reasoning: many of the "26 unidentified
classes" are sibling roles of the same resource, not distinct resources.

**False-positive warning:** `0x7cfee8b30510` and `0x7cff4a926080` matched TitaniumOre at 0.86 m
but are the **player's own character and controller** — the operator was standing beside the vein.
Always exclude actors sitting at the player's live position before drawing conclusions.

### 6b. Inventory diffing — a better identification channel than scanning

Found late in the session and **preferable to memory scanning for resolving internal item names**:
snapshot `dune.items` joined to `dune.inventories`, have the operator gather one node, diff.
It is instant, exact, costs no CPU on the game server, and cannot produce the false positives
scanning does.

Confirmed live: the operator reported "504 iron ore", and the diff showed **`MagnetiteOre` 504**
appear, then 512 when they gathered eight more. **`MagnetiteOre` = "Iron Ore"** — settled from the
item side with zero inference, independently confirming the note in Core's own
`BaseInventoryTab.test.tsx`. This also validates the whole naming convention: **the game uses real
mineral chemistry internally and display names externally**, which makes the Bauxite→Aluminum,
Azurite→Copper and Dolomite→Carbon mappings far safer to rely on.

**`Rhyolite` = "Granite Stone"** — confirmed the same session by teleporting the operator to a
`RhyolitePickup` marker (they landed 0.2 m from it) and having them report what they were standing
on. This had been flagged twice as an unverified guess, with Basalt named as the plausible
alternative; it is now settled. Note the *node* types are distinct (Rhyolite vs Basalt) while the
*item* both drop is the generic `Stone` template — which is why gaming.tools lists "Granite Stone"
and "Basalt Stone" separately but inventories only ever show `Stone`.

**Two traps in the diff method, both hit live:**
- **Items leave the player's inventory.** Filtering on `i.actor_id = <player>` showed iron
  *vanishing* after a gather, because the operator had dropped it. Order by
  `dune.items.acquisition_time` instead — it cuts through inventory moves and shows exactly what
  was acquired and when, regardless of where it ended up.
- **Dropped items become a real actor.** They land in a
  `/Game/Dune/Systems/Looting/BP_LootContainer.BP_LootContainer_C` — a backpack on the ground —
  with its own row in `dune.actors` (live transform), `dune.inventories`, and full contents in
  `dune.items`. **Dropped loot is therefore fully DB-tracked and needs no memory scanning at all**,
  unlike hidden treasure or testing stations. A dropped-loot map layer would be trivial and exact.

### 6d. Confirmed node -> item -> display names (inventory diff, 2026-08-22)

Every row below was confirmed by the operator gathering a single node and diffing
`dune.items`; quantities matched what they reported in-game exactly.

| Node / marker | Item `template_id` | In-game display |
|---|---|---|
| `AzuriteOre` | `AzuriteOre` | **Copper Ore** |
| `MagnetiteOre` | `MagnetiteOre` | **Iron Ore** |
| `ErythriteOre` | `ErythriteCrystal` | Erythrite Crystal |
| `BrittleBush` | `PlantFiber` | Plant Fiber |
| `RhyoliteOre` / `RhyolitePickup` | `Stone` | **Granite Stone** |
| `FuelCellPart` | `Oil` | **Fuel Cell** |

**`Oil` displays as "Fuel Cell", NOT "Impure Fuel"** (operator-confirmed). This matters: earlier
notes in this session assumed `FuelCellPart` was the gaming.tools "Impure Fuel" entry. It is not.
**Impure Fuel is still unidentified** and has no known node type — treat it as outstanding
alongside Jasmium.

**Node name, item name and display name are three independent things.** All four combinations
occur: name carried through unchanged (`AzuriteOre`), mineral name internally with a different
display (`MagnetiteOre` -> "Iron Ore"), node name changed for the item (`ErythriteOre` ->
`ErythriteCrystal`), and a generic shared item across distinct node types (`Rhyolite*` and
`BasaltOre` both -> `Stone`). **Never infer one from another** -- gather and diff. Inferring from
mineral chemistry alone would have produced at least one confident error (Erythrite is a cobalt
mineral, but yields `ErythriteCrystal`, not a cobalt item).

Still unconfirmed by item as of this writing: `Dolomite*` (Carbon Ore, marker only),
`BasaltOre`/`BasaltPickup`, `TitaniumOre`, `StravidiumOre`, `BauxiteOre` (strong class matches but
never gathered), `ScrapMetalPart` vs `ScrapMetalWreckage` (no clean single-variable gather), and
Jasmium (Hagga only, never located).

### 6e. Sandworms, sandstorms and the Coriolis seed

- **Coriolis seed is readable from the DB** via `dune.debug_get_coriolis_seeds()` -- DeepDesert
  and HaggaBasin both seed `2` at time of writing, matching the seed gaming.tools' API uses.
  Related functions exist: `coriolis_update_seed`, `coriolis_cleanup_partition`,
  `update_coriolis_for_player`, `coriolis_cleanup_farm`. **A seed change is the signal that the
  map has regenerated** -- the natural trigger for a full re-scan after a Coriolis storm.
- **Ordinary sandstorms do not change the seed** (verified across one live storm) and are not
  tracked in any table. They are visible only indirectly, as `game_events` `event_type=13` rows
  with `m_TotemShieldStateChangedReason: "Sandstorm"` -- i.e. only where a player owns a structure
  the storm passes over. Three clean Disabled -> Restored pairs on 2026-08-22 give a consistent
  duration of **~7 minutes**; the irregular gaps between them (1h to 8h) reflect storms that
  spawned elsewhere and missed the totem, not irregular timing.
- **Sandworms are not tracked at all.** Checked live while one was active: no worm actor in
  `dune.actors` (the only sandworm match is a decorative statue placeable), and no worm row in
  `game_events`. Worm presence is memory-only, and worms despawn well inside the 2-5 minute a
  scan takes, so catching one would need a fundamentally faster capture.

### 6c. Correction: `dune admin teleport` does work, with a delay

This document previously recorded admin teleport as accepted by RabbitMQ but never moving the
character. **That is wrong, or at least no longer true.** A teleport issued this session
(`DUNE_ADMIN_ASSUME_YES=1 dune admin teleport '<fls-id>' -812980 495 -1200`) was initially judged
broken because an immediate position re-read returned byte-identical coordinates — but the
operator did in fact arrive, landing 0.2 m from the target marker. **The position check was simply
made too soon.** Wait several seconds and re-read before concluding a teleport failed. The earlier
"confirmed broken" finding should be treated as suspect for the same reason.

### 7. Corrections to third-party claims that entered this session

Pasted reference material was **right** about `field_kind_id=0` = Flour Sand and the `field_id`
packing, but **wrong** about: 23,413 marker rows (real: 1,251), `AzuriteOre` 3,067 (real: 6),
`Shipwreck` 30 (real: 14), and `game_events.event_type=0` = "devoured by Shai-Hulud" (real:
`event_type=13` is base totem shield state). Its Leaflet `coordScaleX`/`coordOffsetX` calibration
constants do **not** match Core at all — Core uses `LIVE_MAP_CONFIGS` min/max bounds
normalisation on 4096×4096 images (`duneDb.js`, `worldToLiveMapPoint` in `LiveMapPanel.tsx`).
Treat that whole document as unverified: some of it is accurate, much of it is not.

### 8. Open items

- 14 of 59 active flour-sand fields decode to positions **absent** from the memory scan's 533 —
  the scan under-counts. Cause unknown. Not blocking, since the DB is authoritative for active
  fields.
- The 6 `spice-large` memory hits look like false positives (grid-round coordinates
  -304800/-1016000/0, identical Z of -4143.93). The DB reports only 2 active large fields, both
  of which decode correctly.
- ~~`ValidTransform` accepts denormal near-zero coordinates → garbage hits. Filed as issue #14.~~
  **Fixed 2026-08-24 (PR #17)** -- denormals, `Inf` and an out-of-world `Z` are all rejected now.
  Note the Z bound is `WorldBound`, not a tight value, so it does **not** reject the `Z = 228598`
  hit recorded in 10k; see Session 4 item 5 for why that was a deliberate choice.
- Core bug (different repo): `liveMapBases()` filters `coalesce(a.partition_id,0) > 0`, so a base
  whose instance has despawned (NULL partition — observed live this session) **silently vanishes
  from the Live Map**.
- Sandworm spawns and the desert mouse: no data source found in any table checked. Parked.
  (Imperial Testing Stations were resolved later the same session — see 10c.)

### 10. THE ACTUAL GOAL, and why it reframes everything above

Stated explicitly by the operator late in the session, and it invalidates any DB-only design:

> the whole point is to find everything **after the storm** so players can easily know where to go
> for resources, ships, spice, stations

A Coriolis storm regenerates Deep Desert. `dune.markers` is **discovery-driven**, so at that moment
it is empty and stays empty until players re-explore — it can only ever report what somebody has
already found. **The memory scanner is the only source that can see undiscovered nodes.** Its
recall is therefore not a limitation to design around; it is the critical path, and
[issue #16](https://github.com/Project-Arrakis/dune-resource-scanner/issues/16) is the blocking
defect for the entire project.

The two sources are **sequential, not competing**:
- **`dune.markers` is the Rosetta Stone.** The only source with a name *and* a position, so the
  only thing that can attach a real name to a memory class pointer. Discovery-driven is fine for
  that job — each resource type only has to be found **once, ever**.
- **The scanner is the map-wide instrument.** Once a class is named, it finds every instance
  anywhere, including on day zero after a wipe when markers are empty.

### 10a. Class pointers are disposable — use anchor positions instead

Class pointers are **per-process and per-restart**, and do not transfer between maps. Hagga runs
as `Survival_1`, DeepDesert as `DeepDesert_1` — separate processes, independent ASLR. Worse,
`ClassPrivate` is a runtime heap allocation, so even an offset-from-module-base would not be
stable. And the obvious workaround fails: both the actor's vtable and the UClass's own vtable
point into the main binary at fixed offsets, but **every Blueprint actor shares the same C++
type**, so vtables cannot discriminate `TitaniumOre` from `StravidiumOre`.

**Solution: store anchors, not pointers.** Record *"whatever class sits at (X, Y) is
TitaniumOre"*, and have the scanner probe those anchors at startup to rebuild the table itself.
- **Hagga anchors are permanent** — it is authored terrain, so node positions never move.
- **DD anchors rebuild per storm**, but cheaply: only **one discovered node per resource type** is
  needed to reseed, not a full re-exploration.

This makes issue #16 doubly critical — anchor re-derivation only works if the scanner reliably
finds the node sitting at a known anchor, which it currently does about two times in five.

### 10b. Authored vs procedural: readable straight from the DisplayName

A clean, queryable discriminator found by comparing DD's static A-row wrecks with the dynamic
G-7 cluster:

| DisplayName pattern | Meaning |
|---|---|
| Descriptive (`Shield_Wall_ShipWreck_01`, `Shield_Wall_Cave_06`, `Shield_Wall_ECOLAB_013`) | **authored terrain — permanent, survives a storm** |
| Bare uppercase (`SHIPWRECK`, `CAVE`, `ECOLAB`) | **procedurally placed — re-rolled each Coriolis cycle** |

So the Live Map can cache authored POIs permanently and only re-discover/re-scan the generic ones.

The same split explains the two maps: **Hagga is authored, Deep Desert is procedurally assembled.**
Hagga's spawner names carry real regions and features
(`Survival_1_Graben_..._TP_PinnacleStation`, `Survival_1_RedD_..._CB_RedDesert_Landmark_Mirzabah`),
while DD's carry only a numbered slot (`DeepDesert_1_CB_WL_<N>`). That is why DD resets and Hagga
does not.

### 10c. Imperial Testing Station = the `Ecolab` marker type

Proven by the DisplayName payload on HaggaBasin's Ecolab markers, which read literally
`Survival_ShieldWall_ImperialTestingStation1/2/3`,
`Survival_HaggaRift_ImperialTestingStation1/2`,
`Survival_JabbalEifrit_ImperialTestingStation1/2/3`,
`Survival_VermiliusGap_ImperialTestingStation1/2/3`. **This supersedes the earlier conclusion in
section 5b that testing stations have no marker type** — they do, it is `Ecolab`. The DD station
the operator entered (F-4, `area_id 30`) is an `Ecolab` marker too, just with the generic
`ECOLAB` display string. Note `Ecolab` is broader than testing stations (it also covers
`Survival_Oodham_Outpost5/6`), so the payload name is what disambiguates.

### 10d. Shipwrecks: 10 slots, 4 active, and `actor_spawners` is discovery-independent

`dune.actor_spawners` lists content-block spawners **regardless of discovery** — the one POI
source immune to the marker problem. For DeepDesert:

| dimension_index | Slots | `CB_WL_` indices |
|---|---|---|
| 0 | 10 | 35, 77, 85, 132, 166, 167, 191, 212, 236, 295 |
| 1 | **4** | 166, 167, 212, 295 (a strict subset) |

The operator independently reports **4 dynamic wrecks in rows B–I** (plus the 3 authored
`Shield_Wall_*` ones in the A row), matching dimension 1 exactly — the same
candidate-pool/active-subset architecture as flour sand (533/59) and spice (376/56).

**`CB_WL_<N>` cannot be decoded to coordinates.** The slot's position is generated per Coriolis
seed, so the same WL sits somewhere different after each storm; `actor_spawners.id` is a plain
sequential integer, not packed coordinates (checked). What the table *does* give, discovery-free:
how many wrecks exist this cycle, which slots are in play, and a change signal when the layout
regenerates.

### 10e. Ore vs Pickup answered, and three more item names

**`*Ore` and `*Pickup` yield the same item; only the quantity differs.** Settled with Basalt,
the first clean single-variable test (previous attempts were spoiled by pre-existing stock):
`BasaltPickup` gave `Basalt` x4, then `BasaltOre` gave `Basalt` x234 from the same stack. So the
`*Ore` types identified only by class can safely inherit their `*Pickup` counterpart's item name.

| Node | Item `template_id` | Display | Note |
|---|---|---|---|
| `BasaltOre` / `BasaltPickup` | `Basalt` | Basalt Stone | **distinct from `Stone`** |
| `SaguaroSeed` | `SaguaroResourceRaw` | **Agave Seed** | all three names differ |
| `PrimroseField` | **none** | (dew / water) | see below |

Two predictions were made and **both were wrong**, which is worth recording because it is now a
consistent pattern: Basalt was predicted to yield the generic `Stone` (it does not -- `Stone` is
specifically Granite), and Saguaro was predicted to yield `AgaveSeed` (it yields
`SaguaroResourceRaw`). Together with the earlier cobalt-for-Erythrite and Impure-Fuel-for-Oil
misses, that is **four failed template predictions out of four attempts**. Never infer; gather.

### 10f. Primrose: the first resource this methodology cannot reach

`PrimroseField` is Hagga-exclusive flora harvested for dew, using a `DewReaper` tool. Harvesting
it produced **no inventory item at all** -- `dune.items` is unchanged before and after -- and
`dune.player_state` has **no water/hydration column** (every column was listed and checked). The
water goes into client/memory character state that is never persisted.

**So it is invisible to both the inventory diff and the database.** This is a genuine boundary,
not a gap to be closed: there is nothing to read. A Live Map can still show *where* Primrose
fields are (they have markers), but nothing about yield or state -- unlike spice and flour sand,
where `resourcefield_state` carries exact live values. Any resource taxonomy must allow for this
category.

Related: the gaming.tools list of 14 raw resources is a **minerals/materials** list and is not
exhaustive for gatherables. Water-bearing flora and seeds do not appear on it.

### 10g. Hagga regions come from marker payloads, not `area_id`

Hagga is divided into named regions the player is notified of on entry. They are extractable from
the `Survival_<Region>_` fragment of `dune.markers.payload->>'DisplayName'`:

| Region | Named markers |
|---|---|
| VermiliusGap | 68 |
| HaggaBasinSouth | 42 |
| HaggaRift | 35 |
| JabbalEifrit | 29 |
| ShieldWall | 27 |
| Oodham | 21 |
| Sheol | 12 |

Seven discovered; the operator reports about nine, and `Graben` and `RedD` both appear in spawner
names without marker payloads yet, so those are most likely the missing two.

**Only POI markers carry region names.** Resource markers (ore, pickup, bush) have an empty `{}`
payload, so entering a new region and mining reveals nothing -- a *named* POI (Cave, Ecolab,
Sietch, Shipwreck) must be discovered there. This was confirmed live: Hagga's marker count went
861 -> 2,746 while the operator explored two new regions, and the region list did not change.

**Normalise before grouping.** The game's own strings are inconsistently spelled:
`ShieldWall` (24) vs `Shieldwall` (3), and `VermiliusGap` (66) vs `VermilliusGap` (2, double-L).
A naive `GROUP BY` invents phantom regions.

**Correction to 5b: `area_id` is NOT a reliable foreign key into `map_names`.** In DeepDesert the
matches looked convincing (`ClosedOffTestingStationIsland` sat exactly on the real testing
station, plus `WindPass`, `FallenLight`, `ErythriteCaveIsland`), but in HaggaBasin the same join
resolves to nonsense -- `area_id 12` -> "ArtOfKanly", a story map, with 205 markers. Several DD
`area_id` values (41, 42, 46, 47, 55, 58, 59, 65, 71, 81) resolve to nothing at all. **The DD
matches were probably coincidental ID collisions and were over-trusted.** `area_id` still groups
markers into real spatial blocks -- that part held up repeatedly -- but its *name* should not be
believed.

### 10h. Compass: +X is North

Confirmed by the operator describing a Basalt ore node as "in front to the N" while it lay in the
`+X` direction (dx +647, dy -123). This corrects an earlier mis-call in this session where a
Stravidium vein was described as "SE" by raw-axis reasoning and the operator reported SW.

### 10i. Hidden treasure: mechanic known, actor unreachable

**Mechanic:** use a compactor on sand -> a container spawns -> looting it completes a journey.

**Yield** (one sample, all at the same `acquisition_time`): `LandsraadTreasureComponent1` x5,
`SpiceSand` x5, `SpiceResidue` x1, `SteelBar` x1, `IronBar` x1, `Silicone` x1.

**Not in the database.** With a container exposed and unlooted directly beneath the stationary
operator, a full `dune.actors` diff (837 rows before and after) showed **no new actor** -- the
only change was their Sandbike respawning. **A memory scan of the same spot returned literally
nothing** (a 40 m box on the correct Hagga process, `Survival_1`).

This is the third consecutive failed treasure scan, and the first with every variable controlled.
**Recommend not spending further in-game time on treasure until #16 is fixed.**

Note the contrast, which fits a pattern seen throughout: **player-persisted things are in the DB,
world-generated content is not.** A dropped-items backpack registers as a real
`BP_LootContainer` actor with a live transform and full contents; base storage containers
register; a treasure container does not.

### 10j. Journeys are DB-tracked; Landsraad tasks are not active

`dune.journey_story_node` holds **12,312 rows**, plus `journey_tracked_cards` (3) and
`journey_story_node_cooldown`. So quest/journey progress *is* persisted server-side -- unlike
storms and worms. The `landsraad_task_progress`, `..._player` and `..._guild` tables all exist but
are **empty**, so that system is not live on this server, which is why the
`LandsraadTreasureComponent1` from the treasure has nowhere to feed.

### 10k. Two more scanner defects found

- **The recall bug (#16) is process-independent.** A scan of the **HaggaBasin** process
  (`Survival_1`, PID 1172356) with the operator standing still returned 1 hit, and that hit was a
  false positive 1,539 m away. Every earlier failing scan was against `DeepDesert_1`. This rules
  out any explanation specific to DeepDesert's memory layout, its procedural content-block
  assembly, or Coriolis regeneration. **The defect is in the tool.** Logged as a comment on #16.
- **`ValidTransform` does not bound Z.** `WorldBound` is applied to `x` and `y` only, so the false
  positive above passed validation with `Z = 228598`. **Fixed 2026-08-24 (PR #17)** -- but with
  `WorldBound` on all three axes, which does not reject `228598`; see Session 4 item 5.
- **Minor:** an empty scan emits `null` rather than `[]` (Go marshalling a nil slice), which
  breaks naive downstream parsers. **Still open as of 2026-08-24.**
- **Added 2026-08-24: `FindNearbyXY` accepted NaN pairs** (issue #18, PR #19) -- both range guards
  were negations, and every NaN comparison is false, so NaN matched every target at every
  tolerance. This was the source of ~17.4M of the 17.4M hits in a single scan. Fixed.

## Session 4 (2026-08-24) -- #16 root cause found; the pass-2 architecture is wrong

**Read this before section 11 -- it changes what Track A item 1 actually is.** All
of it was measured live against `dune-dev` (DeepDesert `DeepDesert_1`, PID 390735).
The Coriolis seed was still `2`, unchanged since session 3, so the map had not
regenerated and `dune.markers` was still valid ground truth. Full evidence,
including the four diagnostic tools, is in
[`findings/2026-08-24-issue-16/`](findings/2026-08-24-issue-16/).

### 1. The loss is entirely in pass 2. Pass 1 is perfect.

Ground truth: the WindPass box (`-near=-4368,-198837 -tolerance 15000`) holds
**211** `dune.markers` rows.

| Stage | Coverage |
|---|---|
| Pass 1 -- raw (X,Y) transform located in memory | **211 / 211 (100%)** |
| Pass 2 -- resolved to a validated actor | **6 / 211 (2.8%)** |

Recall is also worse than #16 estimated: the 96 actors the baseline returns occupy
only **42 distinct positions**, and just 6 sit on a real marker.

**Positions were never the problem.** What the scanner actually fails to produce is
a resource *type* per position.

### 2. The hypothesis #16 was opened with is disproved

#16 supposed actors were lost because the pointer to their `RootComponent` lay
outside the scanned regions. It does not. An **offset-agnostic** probe searched
`[hit-2048, hit]` for *any* pointer anywhere in memory, then walked back up to 2048
bytes from each looking for a valid vtable/ClassPrivate chain, assuming neither
`Transform=384` nor `RootComponent=576`:

| Probe | pass-1 hits | resolutions |
|---|---:|---:|
| TitaniumOre `-3814,-198877` (baseline "finds" it) | 13 | **0** |
| TitaniumOre `-7923,-207410` (baseline misses it) | 13 | **0** |
| ScrapMetalPart `-17893,-208453` (baseline misses it) | 14 | 2, at `tOff=1936, rcOff=1104` |

**Nothing points into the 2 KB preceding these transforms.** Widening the region set
cannot fix that. Note row 1: the node the baseline *does* report resolves to zero
actors when probed precisely -- its wide-scan hit was a neighbouring actor 8.7 uu
away, so several of the 6/211 "successes" are coincidental neighbours, not nodes.

### 3. Ore/scrap/pickup nodes are spawn records in an array, not actors

An annotated qword dump around a real TitaniumOre transform shows a repeating
**384-byte record** (bases 384 apart):

```
 +0    HEAP-PTR              (sequential across adjacent records)
 +8    0x0000000100000001
 +16   2
 +232  f64 x, y, z           z ~ 1_051_450   (pre-trace sentinel Z)
 +256  f64 x, y, z           z ~ 2_844       (terrain-snapped Z)  <- hits land here
 +280  EXE-PTR
 +320  EXE-PTR
 +336  EXE-PTR
```

The record before our node holds another node's position. The two triples 24 bytes
apart read as a downward ground-snap trace. This matches section 10d: these are
content-block spawner slots, the same family as `dune.actor_spawners`.

**Spice and flour sand are unaffected** -- seed mode reaches them through the
`BaseValue` field inside a genuine actor, which is why that path always worked. The
`actor -> RootComponent -> transform` architecture is correct for them and wrong for
everything else.

### 4. The fix direction: a spawn-record signature, not an actor chain

Filtering pass-1 hits to those whose `hit-256` matches the record signature (heap
pointer at `+0`, `0x0000000100000001` at `+8`):

| | Unfiltered | Signature-filtered |
|---|---:|---:|
| Plausible-Z hits | 106,467 | **274** |
| Markers covered | 211 / 211 | **152 / 211 (72%)** |
| Copies per marker | 8 - 334 | **1 - 2** |

A signature taken from a single dump cuts the candidate set by 99.7% and still
covers 72% of known nodes -- a **25x recall improvement** over the shipped 2.8%,
with no actor chain, no back-reference, and no `ClassPrivate`.

Still missing: a **type** per record. Scoring every offset from -2048 to +512 found
no value shared by all markers of a type and no others. Two leads remain, in order:
relax the (probably over-strict) signature and re-measure; then follow the `+0` heap
pointer to whatever it points at. The three EXE pointers at `+280/+320/+336` are the
only module-relative values in the record, so if one identifies the type it would
also be **stable across restarts**, solving the ASLR/anchor problem in section 10a.

### 4b. Map-wide census: 64.8% coverage, validated against 7,934 markers

The 72% above came from one box. `dune.markers` for DeepDesert has since grown from
440 (session 3) to **7,934** rows across **31 types** as the operator explored, so the
signature was re-tested map-wide rather than per-box.

Scanning the whole world in one pass, filtering on the signature inline and streaming
to disk: **59,254,091 raw triples -> 85,788 records, in 17.3s at 136 MB RSS.** The
shipped scanner takes 2-5 minutes on a *single box*.

**Coverage: 5,144 / 7,934 markers (64.8%)**, against the shipped 2.8% -- a **23x
improvement** measured across the whole map. Three clean patterns:

- **`*Ore` and rock nodes: 66-89%** (DolomiteRock 89.0, AzuriteOre 82.5, RhyoliteOre
  80.0, TitaniumOre 77.7, StravidiumOre 72.7, BasaltOre 69.9, BauxiteOre 69.8).
- **Small `*Pickup` nodes: 14-30%** -- except RhyolitePickup and AzuritePickup at 67%.
  A second record shape is the likely explanation and the best lead for going past 65%.
- **POIs: 0%** (Cave, Shipwreck, Ecolab, TaxiService, Hazards, HomeBase). Exactly what
  section 5c predicts -- structures are streamed per player. Not a defect.

Records are also clean: **84,744 distinct positions out of 85,788**, versus the shipped
scanner's 96 results collapsing to 42.

**Type attribution: four routes tested, all ruled out** (on 2,931 records with an
unambiguous single-type label):

| Route | Result |
|---|---|
| Actor chain | No back-references exist at all |
| All 48 record offsets 0..376 | No per-type value. `+0` is unique per record (2,931 distinct, zero collisions) -- a per-instance handle |
| The object `+0` points at | Not a UObject -- holds float64 coordinate pairs (a bounding box), so no class to read |
| Memory-address clustering | 30.9% same-type adjacency vs a 12.9% chance baseline -- real but far too weak; every type spans the same ~3.7 GB |

**The type does not live in the record, in what it points at, or in its placement.**

**Caveat, CORRECTED same session.** An earlier version of this section claimed the
unmatched 80,803 records were "not all undiscovered nodes", citing a Z gap of 18,058
vs 3,715. **That was largely wrong.** The gap is mostly exploration bias: records in
well-explored cells have median Z **4,701**, records elsewhere **19,317** -- the
operator explored the low ground, and the unexplored north is high terrain. Controlled
to well-explored cells the gap collapses to **3,559 vs 7,060**.

A structural test then ruled out the "different objects" reading: a signature derived
from the 5,899 marker-confirmed records (**47 of 48 fields agree across all of them**)
passes **82,682 of 85,784 records (96.4%)** map-wide. There is no structural signal
separating node from non-node, most plausibly because most of them are nodes.

This **cannot be settled from data alone** -- it is a positive-unlabelled problem,
since `long_range=false` resource markers only appear on close approach, so "no marker"
never means "no node". It needs a ground-truth visit; three validation sites in
never-explored terrain are recorded in `findings/2026-08-24-issue-16/README.md`.

Two real data-quality issues did surface: **1,709 records (2.0%) sit within 1,000 uu of
the world origin** and are junk (567 within 100 uu, 43 at exactly `(1,1,1)`) --
`census.go`'s `plausibleXY` accepting any `|v| >= 1` is too permissive -- and there are
1,051 duplicate positions.

### 4c. POIs are NOT discovery-limited -- every one is `long_range`, so the DB is their complete source

Operator report, 2026-08-24: **all Ecolabs on the map are discovered, and they never
entered any of them.** Checked directly against the DB, and it holds for every POI
type on both maps:

| Map | Type | Rows | `long_range=true` | named |
|---|---|---:|---:|---:|
| DeepDesert | Cave | 11 | **11** | 11 |
| DeepDesert | Ecolab | 6 | **6** | 6 |
| DeepDesert | Shipwreck | 6 | **6** | 6 |
| DeepDesert | TaxiService | 3 | **3** | 3 |
| HaggaBasin | Cave | 95 | **95** | 95 |
| HaggaBasin | Ecolab | 14 | **14** | 14 |
| HaggaBasin | Shipwreck | 13 | **13** | 13 |
| HaggaBasin | Sietch | 8 | **8** | 8 |

**100% `long_range`, 100% named, on every POI type.** The sole exception is `NoIcon`
(DD 5, Hagga 16), which is neither.

This **refines section 5's blanket warning**. "Never present `dune.markers` as an
exhaustive atlas" remains correct for **resource nodes** (`long_range=false`, revealed
only by going there). It is **wrong for POIs**: they are revealed at range, so the DB
already holds them all and **the memory scanner is not needed for the POI layer at
all**. That also confirms the census's 0% POI coverage is correct and expected, not a
defect -- section 5c already established that structure geometry is streamed per player.

The DisplayName payload splits them exactly along section 10b's authored/procedural
line, which tells the Live Map what to cache and what to re-read each cycle:

| Type | Procedural (bare name, re-rolled per Coriolis cycle) | Authored (`Shield_Wall_*`, permanent) |
|---|---:|---|
| Ecolab | 4 | 2 (`ECOLAB_005`, `ECOLAB_013`) |
| Shipwreck | 3 | 3 (`ShipWreck_01/02/03`) |
| Cave | 7 | 4 (`Cave_06` x2, `Cave_09` x2) |
| TaxiService | 3 | 0 |

The 4 procedural DD Ecolabs are slot-placed, not free-placed: three sit at exactly
`z = 2790`, and two share `y = 193344` exactly. The `area_id 30` one at
`(-201473, -214617)` is the F-4 station from section 5b. Every marker carries
`area_radius = 30000`.

**Open question, and it is the one that matters for the post-storm goal:** "discovered
without entering" proves long-range reveal works without visiting the POI. It does
**not** establish that markers populate with no player on the map at all, nor how
quickly they repopulate after a Coriolis storm wipes them. Until that is observed
across an actual storm, do not assume the POI layer is available at t=0 post-storm.

### 5. Two real defects found and fixed en route

- **#18** -- `FindNearbyXY` accepted NaN pairs: both guards were written as "skip if
  outside tolerance", and every NaN comparison is false, so NaN fell through both and
  matched **every** target at **every** tolerance. 16 bytes of `0xFF` (two int64
  `-1`s) is the common source. This produced **17,385,580** of the original 17.4M
  hits. After the fix: 623,083 hits, marker coverage unchanged at 211/211, peak RSS
  **12.6 GB -> 165 MB** -- material on a host with 24 GB free that also runs the live
  game server. PR #19.
- **#14** -- `ValidTransform` now rejects denormals, `Inf`, and out-of-world `Z`. The
  Z bound is deliberately `WorldBound` rather than a value tuned to reject the
  observed `Z = 228598` hit: across all 12,667 markers and live `dune.actors`, real
  `|z|` never exceeds 75,246, but flying actors have no established ceiling, so a
  tight bound risked real misses. **It therefore does not reject that specific hit.**
  PR #17.

### 6. Correction to section 11's Track A

Track A item 1 said to instrument the passes, then "try widening the region set" and
"relax the vtable check". The instrumentation is now done and **both of those
follow-ups are ruled out** -- there is no back-reference to find at any region
setting. Item 1 is not a bug fix; it is a **redesign of the resolution strategy**,
and should be sized accordingly.

Item 4 (class-anchor mode) is also affected: anchors were meant to re-derive
`ClassPrivate` pointers after a restart, but ore nodes have no reachable
`ClassPrivate` at all. If a record-relative EXE pointer turns out to identify the
type, anchors may be unnecessary for these nodes.

### 11. Revised next steps -- three tracks (A blocks C)

#### Track A: fix the tool (blocking)

Nothing downstream works until the scanner is trustworthy.

1. **#16 -- recall.** ~20-40% of node actors are never returned; proven tool-side (fails
   identically in both map processes). Instrument pass 1 (X,Y candidates), pass 2 (pointer
   references) and each `ValidateActor` rejection stage so the loss is *observed*, not inferred.
   Then test widening the region set beyond `[heap]` + anonymous-writable, and relaxing the vtable
   check to any file-backed executable/rodata mapping. Validate against the WindPass box, which
   has a known 200+ marker ground truth.
2. **#14 -- `ValidTransform`.** Reject denormals and `Inf`, **and add a Z bound** (currently only
   X and Y are bounded, which let a `Z = 228598` false positive through).
3. **Emit `[]` not `null`** for an empty scan.
4. **Add a class-anchor mode.** Given a table of `resource -> known (X,Y)`, probe each anchor and
   emit the resolved class pointers. This makes the ASLR problem self-healing (see 10a) and is the
   mechanism the Live Map needs after every server restart.
5. **Consider a fast-capture mode** -- dump candidate regions to disk and analyse offline. A 2-5
   minute inline scan cannot catch a sandworm, and probably cannot catch a storm either.

#### Track B: data gathering for the dynamic Live Map

Target is a live map of **both** DeepDesert and Hagga Basin covering resources, POIs, player
buildings, and storm/worm activity. Current source-by-source status:

| Layer | Source | State |
|---|---|---|
| Spice (3 tiers), Flour Sand | `resourcefield_state` + `field_id` decode | **Ready for the inner ~87% of the map** -- exact and live, but the 21-bit packing cannot represent \|x\| or \|y\| beyond 1,048,575, where 12.9% of real DD markers sit (see section 2) |
| Ore / stone / flora nodes | scanner + class->name | **Blocked on #16** |
| Named POIs (cave, ecolab/testing station, shipwreck, sietch, vendor, camp, hazard) | `dune.markers` | **Ready but discovery-limited** |
| Player bases, storage, vehicles, players | existing `liveMap*` queries in Core | **Already shipped** |
| Dropped loot | `dune.actors` `BP_LootContainer` + inventories | **Ready**, unused |
| Hagga region labels | marker `DisplayName` `Survival_<Region>_` | **Ready** (normalise spelling, see 10g) |
| DD grid labels | computed, 9x9 @ 270,000 uu, origin (-52656,-52066) | **Ready** (verified 5 cells) |
| Map regeneration signal | `debug_get_coriolis_seeds()` | **Ready** |
| Sandstorm (per-structure) | `game_events` type 13, `Sandstorm` reason | **Partial** -- only where a player owns a structure |
| **Active storm position** | none found | **Open** -- memory only, never attempted |
| **Sandworm activity** | none found | **Open** -- absent from `dune.actors` and `game_events`, checked live twice |
| Hidden treasure | none found | **Open** -- not in DB, three empty scans |
| Primrose water | none exists | **Unreachable** -- not an item, not persisted (10f) |

Remaining identification work, cheapest first:

6. **Finish DD item names** -- `Dolomite*` (Carbon) is the only one still genuinely unknown;
   `TitaniumOre`, `StravidiumOre` and `BauxiteOre` can now inherit from their `*Pickup`
   counterparts per 10e, though a direct gather is still cheap.
7. **Hagga pass** -- **Jasmium** (its only home, last of the 15); confirm whether Titanium,
   Stravidium, Bauxite, Dolomite and Magnetite exist there at all (currently zero discovered
   markers, but Basalt appeared the moment the operator walked past one, so absence is probably
   discovery, not reality); find the two missing regions (`Graben`, `RedD` expected); build the
   **permanent anchor set**, which never expires because Hagga is authored.
8. **Impure Fuel** -- unidentified, no known node type. `FuelCellPart` yields `Oil` which displays
   as "Fuel Cell", so it is something else entirely.
9. **Storm and worm hunting** -- both are memory-only and both are fast-moving. Do not attempt
   with the current inline scanner; revisit after Track A item 5.
10. **Buildings** -- Core already queries player bases, storage and vehicles for the existing Live
    Map. Reuse those rather than rebuilding; note the NULL-`partition_id` bug (a base whose
    instance despawned silently vanishes) recorded in section 8.

#### Track C: build it

11. Only once Track A is done. Sources per the table above. Design constraints from the earlier
    eight-hats audit still stand: no continuous privileged sidecar, scoped read-only DB role,
    console API reads a read-only bind-mounted scanner output file rather than talking to the
    scanner. An interim `resourcefield_state`-only layer needs none of that and could ship early,
    but it does not serve the post-storm goal on its own.

- Go 1.24.4 is available locally on this machine; NOT installed on `dune-dev` — doesn't matter,
  cross-compile locally (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`) and `scp` the static
  binary over (`/tmp/dune-resource-scanner` on dune-dev — not persisted anywhere, redeploy after
  every `main` change).

## Why Go, why not the Python prototype
A third-party zip (`dune-spice-tools`) was reviewed, deeply audited (Layer-1 eight-hats design
audit with 8 parallel agents + `/code-review max`, 848 tool calls), and had every finding
remediated — but it's **still license-unresolved** (no LICENSE, no author, own `NOTICE.md`
explicitly says don't vendor it anywhere). It was reviewed from a session scratchpad under `/tmp` and is **deliberately not preserved
anywhere in this repository** — vendoring it is precisely what its own `NOTICE.md` prohibits, so
the usual "never leave anything in `/tmp`" rule does not apply to it. Treat it as reference-only
methodology that no longer has a durable copy, not as code to go find and port. Because the new Go tool is meant to be **permanent** (run weekly
to feed the Live Map), it must be a **clean-room reimplementation** based on facts we
independently re-derived tonight (memory offsets, the actor-validation technique), never a port
of that Python source. This also fully resolves the license concern for the new artifact.

The Python prototype's byte-pattern actor scans (spice, the mystery resource, a string search)
worked fine and fast, because `bytes.find()` is C-level. The NEW capability we need — scanning
for a numeric (X,Y) coordinate near a known position, across a 16GB heap — has no fast stdlib
primitive, and pure-Python loops over it timed out repeatedly (even after bulk `struct.unpack`
and restricting to "hot regions"). That's the actual reason for the rewrite: Go's compiled,
plain-loop performance solves this trivially.

## What the Go tool needs to do (v1 scope)
1. Open `/proc/<pid>/mem` read-only + parse `/proc/<pid>/maps` for heap regions (reimplement
   fresh, don't copy).
2. Actor-shape validation using these **independently verified, empirically re-derived**
   offsets (facts about the compiled game binary, not copyrightable expression):
   - `CLASS_PRIVATE = 16` (UObject-level: vtable-adjacent ClassPrivate pointer)
   - `ROOT_COMPONENT = 576` (AActor-level: root scene component pointer)
   - `TRANSFORM = 384` (offset within the root component: 3 consecutive `float64`s, X/Y/Z)
   - `BASE_VALUE = 1440` (offset within the actor: a "full/base" numeric field — works
     identically for spice AND the second resource kind, confirmed by us tonight, not assumed)
   - Validation chain: `addr+0` must be a pointer into the executable's mapped range (vtable);
     `addr+CLASS_PRIVATE` must be a heap pointer whose own `+0` is also in-exe; `addr+ROOT_COMPONENT`
     must be a heap pointer; read 24 bytes at `rc+TRANSFORM` as 3 float64s, reject NaN/out-of-world
     (`abs(x) or abs(y) > 1_250_000`)/exact-origin.
3. Seeded byte-pattern scan: given a numeric seed value (int32, e.g. spice's 5000/150000/2500000,
   or the mystery resource's 60000), find candidates and validate per above. Generalize to a list
   of `(label, seedValue)` pairs so multiple resource "kinds" can be scanned in one pass.
4. **New capability** (the reason for this rewrite): given a target `(nearX, nearY, tolerance)`,
   scan for any 8-byte-aligned float64 X within tolerance immediately followed by a float64 Y
   also within tolerance — a plain, fast, compiled loop, no vectorization tricks needed. For each
   hit, work backward: `rc = hitAddr - TRANSFORM`; scan for the raw pointer value `rc` elsewhere in
   memory; for each such reference at `refAddr`, candidate actor `= refAddr - ROOT_COMPONENT`;
   validate via the same chain as #2.
5. JSON output (position, class pointer, seed/value, label) — this will eventually feed the Live
   Map's console API.

## Confirmed facts / findings so far (verified live on `dune-dev`, don't re-derive from scratch)

**Database schema** (`dune.resourcefield_state`: `field_id, map, dimension_index, spawn_time,
value_remaining, field_kind_id` — no position column, confirmed independently by this repo's own
`console/api/src/duneDb.js` comments too):
- `field_kind_id=1` = confirmed spice. Tiers 5000/150000/2500000 = Small/Medium/Large.
- `field_kind_id=0` = a real, separate resource. **Single constant** `value_remaining=60000`
  (never varies). Shares IDENTICAL memory offsets with `SpiceActor` (see above) — confirmed via
  live calibration on `dune-dev`. Its **identity is still unconfirmed** — two in-person visual
  checks by the user both turned out to likely be genuine, already-known nearby spice fields
  (field_kind_id=1) instead, because field_kind_id=0 instances despawn within minutes (count
  dropped 30→9 in one observed window) — faster than travel time allows. The user's own
  hypothesis ("`field_kind_id=0` = inactive spice-spawn candidates, flips to 1 when spawned") was
  tested and **ruled out**: counts are the same order of magnitude as `field_kind_id=1` (should be
  ~15x bigger if it were a full inactive pool), the constant value doesn't match any spice tier,
  and memory scanning found a **different class pointer** than SpiceActor's own — plus the
  active/dormant split (16-29 active out of a 533-position pool) already happens entirely within
  `field_kind_id=0`'s own actor population, independent of the DB flag.
- **Raw ore-type resources are NOT in `dune.resourcefield_state` at all, and NOT in `dune.actors`
  either** (checked live while the user stood directly on a Titanium Ore node — `dune.actors` for
  DeepDesert showed only the player's own character/controller/vehicle, zero resource rows).
- **But they ARE real, discrete actors** — confirmed via a plain ASCII string scan of the live
  process's memory, which found genuine internal names following a `BP_<Mineral>_Spawner` /
  `BP_<Mineral>_Pickup_[A/B]_Spawner` / `BP_<Mineral>_Component` pattern:
  - **Titanium** → Titanium Ore (direct name match, confirmed while user stood on one)
  - **Bauxite** → likely Aluminum Ore (Bauxite is aluminum's real-world ore mineral)
  - **Azurite** → likely Copper Ore (copper carbonate mineral)
  - **Dolomite**, **Rhyolite** → two more stone/ore types, not yet mapped to
    Basalt Stone/Granite Stone/Carbon Ore from the gaming.tools name list
  - **ImpureFuel**, **CompactedFlourSand** → direct matches (Impure Fuel, Flour Sand)
  - Internal names use real mineral chemistry, NOT player-facing names (already independently
    confirmed elsewhere: `MagnetiteOre` internally = "Iron Ore" display, per this repo's own
    `console/web/src/features/bases/BaseInventoryTab.test.tsx`). **Don't assume a display name
    matches its internal string** — search generic category suffixes (`Ore_C`, `Stone_C`, etc.)
    or real mineral names, not guessed display-name strings.
  - These have NO database tracking of any kind — position can only ever come from memory
    scanning (this Go tool), never a DB query.

**Canonical raw-resource name list** (from `dune.gaming.tools/deep-desert`'s own live map
filters, NOT its separate item-catalog page which lists item types generally): Aluminum Ore,
Basalt Stone, Carbon Ore, Copper Ore, Erythrite Crystal, Flour Sand, Granite Stone, Impure Fuel,
Iron Ore, Jasmium Crystal, Plant Fiber, Scrap Metal, Stravidium Mass, Titanium Ore — plus Spice
Field (Large/Medium/Small).

**gaming.tools has its own live map API**, `https://dune-api-v2.gaming.tools/actors?world=deepdesert_1&seed=2`
— keyed by **Coriolis seed**, not by week/date. Our `dune-dev` server's own current seed is
**exactly `2`** (read via `coriolis_live()`), an exact match — meaning if this data is
per-terrain-iteration (very plausible, since Deep Desert only rotates through ~12 known fixed
iterations, confirmed by the user), it could be directly reusable. **Blocked by a real Cloudflare
JS challenge** (403, not a simple bot-block) — no browser automation available this session
(`claude-in-chrome` extension not installed/connected). Untried alternative: ask the user to check
it in their own real browser, or revisit with browser automation if available later.

## Live infrastructure / access (dune-dev)
- SSH aliases in `~/.ssh/config`: `dune-dev` (192.168.21.10, user `dune`, in `docker`+`sudo`
  groups, **passwordless sudo**), `dune-prod` (192.168.20.10), `acp-bot` (192.168.22.10).
- Real `dune` CLI at `/usr/local/bin/dune` on `dune-dev` → execs
  `/home/dune/dune-awakening-selfhost-docker/runtime/scripts/dune`.
- **Deep Desert is ephemeral on dune-dev** — `dune-autoscaler` despawns any 0-player map instance
  after a 300s idle grace period. This is normal, expected, NOT caused by our scanning (verified
  directly via the autoscaler's own logs before concluding that, when asked). Player must
  respawn DD and get a character in before any live test.
- `dune admin teleport <player-id> <x> <y> <z>` needs `DUNE_ADMIN_ASSUME_YES=1` env var (no TTY
  over SSH → interactive y/N prompt silently cancels otherwise, exit 1, zero output — cost real
  debugging time to find). **Even with that fixed, the command is accepted by RabbitMQ
  (`publish=ok`) but the character never actually moves in-game** (confirmed via
  `player-location` before/after) — a real, separate, unresolved bug in this admin tooling, not
  investigated further, just flagged. **Don't rely on admin teleport working.**
- `dune admin player-location <player-id>` and `dune admin players --online --show-full-ids` are
  reliable, read-only, safe — use these freely.
- Test character: FLS `BeretGenesis#24872`, name `DarkDante`.
- Confirmed base/anchor point on the Titanium-ore island: `##Totem_Small_Placeable` at
  **X=-611736.35, Y=-700183.46**, DeepDesert, partition 32 — a precise, zero-scanning-uncertainty
  reference (found via a plain, already-proven-safe SQL query on `dune.buildings` etc., the same
  pattern the Live Map's own `liveMapBases()` uses).
- **CRITICAL SAFETY LESSON**: a local `timeout N ssh ...` only kills the LOCAL ssh client on
  timeout — NOT the remote process. Three orphaned Python scans were left pegging ~90-99% CPU
  each for minutes on `dune-dev` (a box also running the live game server) after repeated
  timeouts, found via `ps aux | grep python3` and killed manually. **Always check for and clean
  up orphaned remote processes after any timed-out SSH command**, especially anything
  CPU-intensive.
- Saved memory this session: `dune-dev-is-for-live-experiments.md` — dune-dev is explicitly fine
  for this kind of risky-but-read-only work; `dune-prod` is NOT, full Strict Requirement 7
  caution still applies there.

## User's explicit product ask for the Live Map (once positions exist)
- Show `field_kind_id=1` (spice) markers labeled by **size** (Small/Medium/Large from
  `value_remaining` tiers) — not the literal remaining amount.
- Show `field_kind_id=0` too, but label it honestly as something like **"Unidentified Resource"**
  — not "inactive spice," which would misrepresent it (identity still unconfirmed).
- Show ALL other raw resources too (ore/stone/fiber/etc.) — user wants **one consolidated update**
  covering everything, not incremental partial ships.
- Already-reviewed target files: `console/web/src/features/liveMap/LiveMapPanel.tsx` (existing
  marker types: player/vehicle/base/storage; has an `overlays` reason-map mechanism for honest
  freshness/unavailability messaging — reuse this for the new resource layer's
  "still collecting"/"stale"/"unavailable on this map" states) + `console/api/src/duneDb.js`
  (`LIVE_MAP_CONFIGS` per-map coordinate transform, `liveMapMarkers()` composing all marker
  queries).

## Design constraints already established (from the earlier 8-hats audit — still valid)
- **No continuous privileged sidecar container.** hostPID + Docker-socket access stacked together
  is a near-trivial path to full host root if compromised (Security + Network hats both rated
  Critical). Run the Go scanner as a manual/scheduled host job instead.
- Use a scoped, **read-only** Postgres role for any DB queries — not the superuser default.
- Console API should only read a **read-only bind-mounted** output file from the scanner —
  never talk to the scanner process directly.
- Findings/decisions from a Layer-1 design audit should still be filed as GitHub issues once
  real implementation work in `dune-awakening-selfhost-docker` begins, per that repo's own
  Requirement 20/21 (branch-based work, audit trail).

## Immediate to-do list for the next session

**Superseded — this list is from session 2 and every item is done or obsolete. See section 11
("Revised next steps") for the current list.** Kept only as history of how the work was sequenced;
the session-2 items were: create the repo, write the Go source, cross-compile and deploy to
`dune-dev`, cross-reference positions against string-scan names, catch a live `field_kind_id=0`
field for in-person ID (**done — it is Flour Sand**), and then build the Live Map integration.

**Operational notes still worth keeping from that list:**
- Negative coordinates need the actual `-` sign typed in-game; the game shows no on-screen
  coordinates, so cross-check exact position via `dune.actors`' live transform (more reliable
  than `dune admin players`, whose map field was observed stale — see below).
- `dune admin player-location` and `dune admin players --online --show-full-ids` are read-only
  and safe. **But `dune admin players` reported the wrong map** while `dune.actors`' transform was
  correct; prefer the direct DB query.
- **Confirm the operator is stationary before any scan.** The scanner takes its `-near`
  coordinates at launch and runs 2–5 minutes; two scans were wasted this session because the
  operator travelled mid-scan and the empty results looked like real negative findings.

## Cleanup notes
- `~/spice-discover/` on `dune-dev` still has the disposable Python probe scripts — fine to leave
  or remove.
- Always check `ps aux | grep -E 'python3 probe|dune-resource-scanner'` on `dune-dev` for
  orphaned processes before starting any new scan.
