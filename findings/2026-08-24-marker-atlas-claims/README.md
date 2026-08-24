# Validating a third-party "marker atlas" document

A document describing `dune.markers` as "the global geographical atlas of Arrakis
(**23,413 entries**)" with per-type counts, SQL queries, and claims about
`game_events`, `world_partition` and `resourcefield_state`. Checked against
`dune-dev` on 2026-08-24.

This is the same source `CONTINUATION.md` §7 already flagged (the 23,413 / AzuriteOre
3,067 / Shipwreck 30 figures match exactly), re-checked against a much larger marker
set — 15,553 rows now versus 1,251 then.

**Headline: the structure is accurate and useful. The counts are not wrong so much as
not checkable, and the document's real defect is presenting them as fixed properties
of the game.**

## Verdict by claim

| # | Claim | Verdict |
|---|---|---|
| 1 | `marker` is a composite `(resource_type, x, y, z)` | **FALSE** — it is `(marker_type, x, y, z, payload_type)`: five fields, and the first is `marker_type` |
| 2 | The provided `SPLIT_PART` SQL extracts type/x/y/z | **TRUE** — verified: `(ScrapMetalPart,-1042785,-109694,627,EMarkerPayloadType::Default)` yields `ScrapMetalPart` and `627`. It works by position despite claim 1 being wrong |
| 3 | 23,413 marker entries | **NOT CHECKABLE** — 15,553 here; see "Why counts differ" |
| 4 | All named POI/vendor/trainer/house types exist | **TRUE** — all 26 spot-checked names exist, including every trainer, vendor, fortress and House representative |
| 5 | `JasmiumOre` (18) | **UNCONFIRMED** — no such type here. Consistent with §11: Jasmium is Hagga-only and has never been located on this server. Absence here is not evidence the type is invalid |
| 6 | Per-type counts (32 claims) | **NOT CHECKABLE** for discovery-driven types; **broadly TRUE** for `long_range` types (see below) |
| 7 | `field_kind_id` 1 = Spice, 0 = Flour Sand | **TRUE** — matches §2, independently confirmed |
| 8 | Tiers 5,000 / 150,000 / 2,500,000 | **TRUE** — exactly these three values are present (35 / 24 / 2 rows) |
| 9 | Tier *names*: Small / **Large** / **Colossal** | **DOUBTFUL** — a `SpiceFieldMedium` marker type exists, supporting §2's Small/**Medium**/Large. Not conclusive; only "Medium" has been observed as a name |
| 10 | `field_id` BigInt unpacking | **TRUE** — the JS is correct and matches §2. **But incomplete**: the 21-bit fields cannot represent \|x\| or \|y\| beyond 1,048,575, where 12.9% of real DD markers sit. See `../2026-08-24-field-id-21bit/` |
| 11 | `game_events` type 0 = devoured by Shai-Hulud | **CONTRADICTED** — no `event_type = 0` rows exist, and §6e found no worm row in `game_events` while a worm was live |
| 12 | `game_events` type 10 = base/structure breached | **PLAUSIBLE** — 5 rows, `custom_data` carries `m_CauserType`, `m_BuildableName: Totem_Small_Placeable`, `m_bWasShielded` |
| 13 | `game_events` types 20, 23 | **NOT PRESENT** — only 10, 13, 19, 95, 97 occur here. Absence is not disproof; they may simply never have fired |
| 14 | The worm-hunting query (`custom_data ILIKE '%worm%'`) | **RETURNS NOTHING** — consistent with §6e |
| 15 | `world_partition` holds "all 39 maps" | **FALSE** — 30 rows, 30 distinct maps |

Event types the document does **not** mention, all present here: **13** (totem shield
state — §7 already established this), **19** (totem on/off), **95** (power time left),
**97** (permission rank change).

## Why the counts differ — and why that is not an error

Initially the two-directional spread looked like evidence the numbers were not from a
real snapshot: some claimed counts are 12× ours, others are half ours, and "a bigger
server" only explains ratios above 1. **That inference was wrong.** The operator pointed
out the source is a large populated server, and the per-map split confirms it:

| Type | DeepDesert | HaggaBasin | claimed ÷ ours |
|---|---:|---:|---:|
| TitaniumOre | 334 | **0** | 0.53× |
| TitaniumPickup | 254 | **0** | 0.51× |
| StravidiumOre | 130 | **0** | 0.51× |
| StravidiumPickup | 101 | **0** | 0.29× |
| RhyoliteOre | 63 | 349 | 11.56× |
| AzuriteOre | 75 | 162 | 12.94× |
| PrimroseField | 0 | 286 | 4.19× |
| SaguaroSeed | 0 | 2 | 11.50× |

**Every heavily over-claimed type is Hagga-dominant; every under-claimed type is
DeepDesert-exclusive.** Two independent mechanisms produce exactly this:

1. **HaggaBasin is authored and never resets**, so discoveries accumulate indefinitely.
   A busy server will always have more. All ratios > 1.
2. **DeepDesert is procedurally re-rolled every Coriolis storm** (§10b), so DD node
   counts are a property of the **live seed**, not of the game. Two servers on different
   seeds legitimately differ in *either* direction, and a recent storm resets the count
   to near zero.

So the counts are a snapshot of one server's exploration state and one DD seed. They are
neither verifiable nor falsifiable from here.

The confirming detail: `long_range` types — revealed at range, complete without
exploration, therefore genuinely server-independent — **all agree within 1.36×, three of
them exactly**:

| Type | claimed | ours | ratio |
|---|---:|---:|---:|
| EnemyOutpost | 72 | 72 | **1.00×** |
| EnemyLaborOutpost | 14 | 14 | **1.00×** |
| Sietch | 8 | 8 | **1.00×** |
| Ecolab | 19 | 20 | 0.95× |
| Cave | 116 | 108 | 1.07× |
| EnemyCamp | 446 | 397 | 1.12× |
| Shipwreck | 30 | 22 | 1.36× |

Against 0.29×–12.94× for discovery-driven types. That split is the whole story, and it
is a good sign for the document: where its numbers *can* be checked, they hold up.

## The actual defect

The document frames `dune.markers` as a fixed atlas — "the global geographical atlas of
Arrakis (23,413 entries)", "**`Cave`** (116 nodes)". It is not. For `long_range=false`
rows it is a **per-server discovery log**, and for DeepDesert it is additionally
**per-Coriolis-seed**. Anyone building on those numbers as world constants will be wrong
on every other server and on this one after the next storm.

This is the same trap §5 already records ("Never present `dune.markers` to players as an
exhaustive atlas") — the document reproduces it as a headline claim.

## How to validate a claim like this in future

The counts are the least checkable part and the most quoted, so check in this order:

1. **Schema first, it is cheap and absolute.** `pg_attribute` on the composite type
   settles field names and arity in one query — that alone falsified claim 1.
2. **Split every count claim by `long_range` before comparing anything.**
   `count(*) FILTER (WHERE long_range)` per type. Only 100%-`long_range` types are
   comparable across servers. Comparing the rest produces noise that looks like error.
3. **Split DeepDesert from HaggaBasin.** DD is seed-dependent and resets; Hagga
   accumulates. A single combined total hides both effects and is what made the spread
   look inexplicable at first.
4. **Treat absence as weak evidence.** `JasmiumOre` missing here means undiscovered, not
   invalid. Say "not present" rather than "does not exist".
5. **For enum claims, read `custom_data`, not just the code.** `event_type = 10` is only
   confirmable by its payload naming a buildable and a causer.
6. **Re-verify anything a decode enables at its boundaries.** The `field_id` unpacking is
   correct and still cannot express 12.9% of the map — a claim can be true and unusable.

## Files

| File | What it is |
|---|---|
| `marker-counts-2026-08-24.csv` | All 99 marker types with counts, 15,553 rows total |
| `validation.txt` | Output of the tool below |
| `tools/validate_claims.py` | Re-runnable; splits claims by `long_range` so checkable and non-checkable are never mixed |
