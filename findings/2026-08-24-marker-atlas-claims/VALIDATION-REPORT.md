# Validation Report — "Overview & Architecture" marker-atlas document

**Validated against:** `dune-dev`, live PostgreSQL, 2026-08-24.
**Server state at validation:** DeepDesert and HaggaBasin both on Coriolis seed `2`;
`dune.markers` holding **15,553** rows across **99** distinct marker types.
**Method and re-runnable tooling:** `README.md` and `tools/validate_claims.py` in this
directory.

---

## How to read this report

Claims fall into three categories, and they need different treatment:

| Category | Meaning |
|---|---|
| **Accurate** | Verified true against the live database. |
| **Inaccurate** | Verified false. The document states something the database contradicts. |
| **Not checkable** | Neither true nor false from here. The claim describes a *server state*, not a property of the game. |

The third category is the important one and it covers **every per-type count in the
document**. Those numbers are not errors — they are a snapshot of a different server. The
document's real problem is presenting them as if they were constants.

---

## Summary

- **Accurate:** the schema access pattern, all named entity types, `field_kind_id`
  semantics, the three spice tier values, and the `field_id` bit-unpacking.
- **Inaccurate (4 claims):** the `marker` composite type definition, the `world_partition`
  map count, `event_type = 0`, and the spice tier *names*.
- **Not checkable (all counts):** including the headline "23,413 entries".
- **Incomplete but not wrong (1):** the `field_id` unpacking is correct and silently
  cannot represent ~13% of the Deep Desert map.

---

## 1. Inaccurate claims

### 1.1 The `marker` composite type

> "The `marker` column is a PostgreSQL composite type `(resource_type, x, y, z)`."

**Inaccurate.** The type has **five** fields, and the first is not called `resource_type`:

| # | Field | Type |
|---|---|---|
| 1 | `marker_type` | `text` |
| 2 | `x` | `double precision` |
| 3 | `y` | `double precision` |
| 4 | `z` | `double precision` |
| 5 | `payload_type` | `text` |

**Why it matters.** Anyone writing `(m.marker).resource_type` gets a SQL error, and code
assuming a 4-field composite breaks on unpacking. The document's own `SPLIT_PART` queries
happen to survive this because they index by position and `payload_type` is last — but the
stated definition is wrong.

**The truth:**
```sql
SELECT (m.marker).marker_type, (m.marker).x, (m.marker).y, (m.marker).z
FROM dune.markers m;
```
This is also cleaner than the document's `SPLIT_PART(m.marker::text, ',', 2)::float`
approach, which parses the composite's text rendering rather than accessing its fields.

Real text rendering, for reference:
`(ScrapMetalPart,-1042785,-109694,627,EMarkerPayloadType::Default)`

### 1.2 `world_partition` map count

> "Table: `dune.world_partition` (All 39 Maps, Instances & Server Ports)"

**Inaccurate.** The table holds **30 rows** and **30 distinct maps**, not 39.

### 1.3 `game_events` event type 0

> "`event_type = 0`: Player devoured by Shai-Hulud (`m_KillerType: "ShaiHulud"`)."

**Inaccurate, and contradicted from two directions.**

No `event_type = 0` row exists. The event types actually present are:

| `event_type` | Rows | What `custom_data` actually contains |
|---|---:|---|
| 10 | 5 | `m_CauserType`, `m_BuildableName: Totem_Small_Placeable`, `m_bWasShielded` |
| 13 | 68 | `m_TotemShieldState: Restored`, `m_BuildableName` |
| 19 | 4 | `m_StateChange: On`, `TotemId` |
| 95 | 24 | `m_PowerTimeLeftSeconds` |
| 97 | 2 | `m_NewPermissionRank: CoOwner`, `m_OldPermissionRank` |

Separately, the document's own worm query —
`WHERE custom_data::text ILIKE '%ShaiHulud%' OR ILIKE '%worm%'` — returns **nothing**.
This matches an earlier live observation on this project: a sandworm was active, and there
was no worm actor in `dune.actors` and no worm row in `game_events`.

**The truth:** sandworm activity does not appear to be persisted server-side at all.
Event types 20 and 23 are also absent here — that is weaker evidence, since they may
simply never have fired, but no positive evidence supports them either.

**What is accurate:** `event_type = 10` is plausibly "base/player structure breached" —
its payload names a buildable, a causer, and whether it was shielded.

**Types the document omits entirely:** 13, 19, 95, 97 — and 13 is by far the most common
event on this server.

### 1.4 Spice tier names

> `5,000` = Small · `150,000` = **Large** · `2,500,000` = **Colossal / Huge**

**Doubtful — the values are right, the names look wrong.**

All three values exist exactly as claimed (35, 24 and 2 rows respectively). But a marker
type **`SpiceFieldMedium`** exists in `dune.markers`, which fits a Small / **Medium** /
Large progression rather than Small / Large / Colossal.

This is evidence, not proof — only "Medium" has been observed as an in-game name. But if
the middle tier is "Medium", the document's naming is off by one position across all three
tiers, which would mislabel every spice marker on a map.

---

## 2. Not checkable — every count, including the headline

> "over **23,400** static resources, ore veins, scrap wrecks, caves…"
> "The table `dune.markers` acts as the global geographical atlas of Arrakis
> (**23,413 entries**)."
> "**`Cave`** (116 nodes)", "**`AzuriteOre`** 3,067x", "**`RhyoliteOre`** 4,761x" …

Our count is **15,553**. That difference is **not an error in the document**, and it is
not evidence of anything. Two independent mechanisms make these counts a property of a
particular server at a particular moment:

**Mechanism 1 — discovery.** `dune.markers` rows with `long_range = false` (every ore,
pickup, scrap, bush and hazard) appear only when a player goes near them. The table is a
per-server discovery log that grows with play. A large populated server will always have
more.

**Mechanism 2 — the Coriolis seed.** Deep Desert is procedurally re-assembled every
Coriolis storm. Its node counts are a property of the **live seed**, not of the game. Two
servers on different seeds legitimately differ *in either direction*, and a recent storm
resets DD counts toward zero.

### The evidence that this is the explanation

Splitting the claimed counts by map shows a clean pattern — **every heavily over-claimed
type is Hagga-dominant, and every under-claimed type is DeepDesert-exclusive**:

| Type | DeepDesert | HaggaBasin | claimed ÷ ours |
|---|---:|---:|---:|
| RhyoliteOre | 63 | 349 | 11.56× |
| AzuriteOre | 75 | 162 | 12.94× |
| SaguaroSeed | 0 | 2 | 11.50× |
| PrimroseField | 0 | 286 | 4.19× |
| **TitaniumOre** | **334** | **0** | **0.53×** |
| **TitaniumPickup** | **254** | **0** | **0.51×** |
| **StravidiumOre** | **130** | **0** | **0.51×** |
| **StravidiumPickup** | **101** | **0** | **0.29×** |

HaggaBasin is authored terrain and never resets, so discoveries accumulate indefinitely —
a busy server races ahead. DeepDesert resets, so a server whose DD was recently
regenerated has *fewer* markers there than ours regardless of population.

### The confirming test

`long_range = true` content is revealed at range and is therefore complete **without any
exploration** — genuinely server-independent, and so genuinely comparable. Those claims
hold up:

| Type | Claimed | Ours | Ratio |
|---|---:|---:|---:|
| EnemyOutpost | 72 | 72 | **exact** |
| EnemyLaborOutpost | 14 | 14 | **exact** |
| Sietch | 8 | 8 | **exact** |
| Ecolab | 19 | 20 | 0.95× |
| Cave | 116 | 108 | 1.07× |
| EnemyCamp | 446 | 397 | 1.12× |
| Shipwreck | 30 | 22 | 1.36× |

**Everything server-independent agrees within 1.36×, three of them exactly. Everything
discovery-driven ranges from 0.29× to 12.94×.** That split is decisive, and it is a good
result for the document: where its numbers can be checked, they are right.

### So what is actually wrong here

Not the arithmetic — the framing. Calling `dune.markers` "the global geographical atlas of
Arrakis" with fixed per-type counts presents a **per-server, per-seed discovery log** as a
set of world constants. Any consumer treating "Cave (116 nodes)" or "AzuriteOre 3,067x" as
a game fact will be wrong on every other server, and wrong on that same server after its
next Coriolis storm.

**The truth:** these counts describe one server at one moment. Cite them with the server
and the seed, or not at all.

### One special case: `JasmiumOre` (claimed 18)

No such marker type exists in our data. **This is not evidence the claim is false** —
Jasmium is Hagga-exclusive and has never been located on this server. Absence here means
undiscovered, not invalid. Recorded as **unconfirmed**.

---

## 3. Accurate claims

| Claim | Status |
|---|---|
| All named POI, vendor, trainer, fortress and House-representative types exist | **Verified** — all 26 spot-checked names present, including `TrainerSwordmaster`, `TrainerBeneGesserit`, `TrainerMentat`, `TrainerPlanetologist`, `TrainerTrooper`, `DuncansDojo`, `CHOAMExchange`, `Bank`, all six vendor types, `AtreidesFortress`, `HarkonnenFortress`, `ImperialConsulate`, and every named House representative |
| The `SPLIT_PART` extraction queries work | **Verified** — returns correct type and z despite the composite-type claim being wrong |
| `field_kind_id = 1` → Spice, `= 0` → Flour Sand | **Verified** — independently confirmed on this project |
| Flour Sand has a fixed value of 60,000 | **Verified** — 85 rows, all exactly 60,000 |
| Spice tier values 5,000 / 150,000 / 2,500,000 | **Verified** — exactly these three, 35 / 24 / 2 rows |
| The `field_id` BigInt unpacking function | **Verified correct** — see the caveat below |
| The `dune` schema holds spice state, player positions and marker data | **Verified** |

---

## 4. Accurate but incomplete — the `field_id` limit

The unpacking function is correct. It is also **silently unable to represent about 13% of
the Deep Desert map**, which the document does not mention.

The three fields are 21 bits signed, so the representable range is
**−1,048,576 … +1,048,575**. Deep Desert extends to roughly ±1.27M:

- **1,237 of 9,601 real DeepDesert markers (12.9%)** lie beyond that range.
- Bit 63 of the 64-bit id is **unused** (`0` in all 141 rows), so there is no escape flag
  or extension mechanism.

A coordinate beyond the limit does not fail loudly — it **aliases** to a plausible-looking
in-range value of the opposite sign, which is worse than a missing row.

**The truth:** the decode is correct and safe to use for the inner ~87% of the map. Any
consumer should range-check decoded coordinates rather than trust them blindly. Whether
the game ever actually places spice in the outer band is still unresolved — no
correctly-decoded field beyond the limit has been observed in any dataset examined here.

---

## 5. Recommended corrections to the document

1. Correct the composite type to `(marker_type, x, y, z, payload_type)` and switch the
   example queries from `SPLIT_PART` to direct field access.
2. Correct `world_partition` from 39 maps to 30.
3. Remove the `event_type = 0` / Shai-Hulud claim, and the worm query built on it. Add
   types 13, 19, 95 and 97, which are the ones that actually occur.
4. Re-check the spice tier names against in-game strings; `SpiceFieldMedium` suggests
   Small / Medium / Large.
5. **Label every count with the server and Coriolis seed it was taken from**, and state
   plainly that `long_range = false` counts are a discovery snapshot rather than world
   content. This single change would fix the document's most misleading property.
6. Note the 21-bit range limit alongside the `field_id` unpacking function.
