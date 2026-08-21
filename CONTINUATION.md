# Continuation: Dune Awakening raw-resource discovery → Live Map integration

## Goal
Find real world positions for Dune Awakening's raw resources (spice + an unidentified
second resource + ore/stone/fiber types), then integrate them into the Live Map feature of
`yacketrj/dune-awakening-selfhost-docker` (org is actually **Project-Arrakis** on GitHub —
`git remote -v` confirms `origin = github.com/Project-Arrakis/dune-awakening-selfhost-docker`,
contradicting some stale references elsewhere). Currently mid-build on a permanent Go tool,
`dune-resource-scanner`, to replace a Python prototype that hit a real performance wall.

## Immediate next step (where this session stopped)
- Just created `~/projects/repos/dune-resource-scanner` (git initialized, branch `master` not
  yet renamed to `main`), moved there after starting it in `/tmp` scratchpad by mistake — the
  user correctly flagged both "why /tmp" (this is permanent, not disposable) and "this should be
  a repo under Project-Arrakis" (matching this workstream's convention: repos live flat under
  `~/projects/repos/`, get a real GitHub repo, `CHANGELOG.md`, GitHub Issues, added to the
  Project Arrakis board).
- **Not yet decided**: which GitHub org to push to. Two conventions coexist in this ecosystem —
  `dune-awakening-selfhost-docker` → `Project-Arrakis` org, `r740-dune-deployment-kit` →
  personal `yacketrj`. Ask the user before creating the real GitHub repo.
- **No Go source written yet** — only `go mod init dune-resource-scanner` ran. Go 1.24.4 is
  available locally on this machine; NOT installed on `dune-dev` — doesn't matter, cross-compile
  locally (`GOOS=linux GOARCH=amd64 go build`) and `scp` the static binary over.

## Why Go, why not the Python prototype
A third-party zip (`dune-spice-tools`) was reviewed, deeply audited (Layer-1 eight-hats design
audit with 8 parallel agents + `/code-review max`, 848 tool calls), and had every finding
remediated — but it's **still license-unresolved** (no LICENSE, no author, own `NOTICE.md`
explicitly says don't vendor it anywhere). It lives at
`/tmp/claude-0/-root-projects/23d506c7-e7da-4a33-a336-498708c85a63/scratchpad/dune-spice-tools/pkg/`
— **this is session-scratchpad tmpfs and will not survive** — treat it as reference-only
methodology, not code to copy. Because the new Go tool is meant to be **permanent** (run weekly
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
1. Confirm with the user: GitHub org (`Project-Arrakis` vs `yacketrj`) and visibility for the new
   `dune-resource-scanner` repo; create it for real once confirmed.
2. Write the actual Go source (see "What the Go tool needs to do" above) — clean-room, do not
   read or port the Python prototype's source.
3. Cross-compile (`GOOS=linux GOARCH=amd64`), `scp` the static binary to `dune-dev`, re-attempt
   the ore-actor position search anchored on the confirmed base position
   (X=-611736.35, Y=-700183.46) — should be dramatically faster than the Python attempts that
   timed out.
4. Cross-reference found positions against the string-scan-discovered internal names
   (AzuriteOre, DolomiteOre, RhyoliteStone, Titanium, Bauxite, ImpureFuel, CompactedFlourSand) —
   ideally with an in-game visual confirmation (remember: negative coordinates need the actual
   `-` sign typed; the game shows no on-screen coordinates, so cross-check exact position via
   `dune admin player-location`, not visual guessing).
5. Keep trying to catch a `field_kind_id=0` field while it's still active (they despawn within
   minutes) for an in-person ID — teleport is unreliable, so give the closest currently-active
   position to wherever the player already is, repeatedly, rather than one fixed target.
6. Once positions + names are solid: design and implement the real Live Map integration in
   `dune-awakening-selfhost-docker` (new branch, new `console/api` route reading the scanner's
   output, new marker/layer type with an `overlays`-based freshness indicator), following the
   no-sidecar/scoped-role/read-only-mount design above.

## Cleanup notes
- `~/spice-discover/` on `dune-dev` still has the disposable Python probe scripts — fine to leave
  or remove.
- Always check `ps aux | grep -E 'python3 probe|dune-resource-scanner'` on `dune-dev` for
  orphaned processes before starting any new scan.
