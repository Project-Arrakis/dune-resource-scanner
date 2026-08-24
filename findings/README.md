# Findings index

The maintained, top-level view of what this project has established, what it has
disproved, and where the evidence for each claim lives.

**This file is the entry point.** Every investigation gets an entry here in the same
session it happens — a finding that is not indexed is one nobody will locate in six
months, which is the failure mode this file exists to prevent.

- **Status legend:** ✅ established · ❌ disproved · ⚠️ partly true / bounded · ❓ open
- **Every claim links to the evidence that supports it.** If a row has no link, it does
  not belong here yet.
- A **shareable, sanitised** version of this index is published as a web page — see
  [Sharing externally](#sharing-externally). This repo is private, so links below are
  useless to anyone outside it; the published page carries the substance instead.

---

## Current state at a glance

| Capability | Status | Evidence |
|---|---|---|
| Node **positions**, including undiscovered ones | ✅ ~60–64% recall, whole map, 17 s | [validation](2026-08-24-validation/README.md) |
| Node **types** (which node is Titanium) | ❌ Unsolved — four routes ruled out | [issue-16](2026-08-24-issue-16/README.md#type-attribution-four-routes-tested-all-ruled-out) |
| Spice + Flour Sand positions from the DB | ⚠️ Exact, but only the inner ~87% of the map | [field-id-21bit](2026-08-24-field-id-21bit/README.md) |
| Named POIs from the DB | ✅ Complete without exploration | `CONTINUATION.md` §4c |
| Scanner works with **zero players online** | ❓ Untested — the last post-storm dependency | [validation](2026-08-24-validation/README.md#the-instance-dependency--untested-and-it-is-the-real-constraint) |

---

## Findings

### 2026-08-24 — Issue #16 root cause: ore nodes are not actors

**[`2026-08-24-issue-16/`](2026-08-24-issue-16/)**

| | |
|---|---|
| ✅ | Pass 1 locates **211/211** known markers (100%); pass 2 resolves **6/211** (2.8%). The loss is entirely in actor resolution |
| ❌ | The hypothesis #16 was opened with — back-references outside the scanned regions — is **disproved**. Offset-agnostically, *nothing* points into the 2 KB before these transforms |
| ✅ | Ore/scrap/pickup nodes are **384-byte spawn records**, not UObject actors. Spice and flour sand are unaffected |
| ✅ | A signature-based census reaches **64.3% DD / 58.5% Hagga** in 17 s at 136 MB |
| ❌ | **Type attribution: four routes ruled out** — the actor chain, all 48 record offsets, the object `+0` points at, and memory-address clustering |

Evidence: the measured funnel before and after the NaN fix, offset-agnostic probes,
an annotated memory dump, and the census. Tools: `census.go`, `analyse_census.py`.

### 2026-08-24 — Validation: the scanner finds *undiscovered* nodes

**[`2026-08-24-validation/`](2026-08-24-validation/)**

| | |
|---|---|
| ✅ | **Retrospective test**: 60.2% of 1,667 markers discovered *after* the scan was captured, vs 64.8% of already-known ones. **Discovery is not required** |
| ✅ | **Cross-map test**: the DD-derived signature gives 58.5% on Hagga — different map, authored terrain, a process that had restarted with fresh ASLR. Map- and process-independent |
| ❓ | **Zero-player operation untested.** Every scan ran while a session was registered online; the autoscaler despawns 0-player instances after 300 s |
| ❌ | *Corrected*: an earlier claim that unmatched records were "not all nodes" was largely wrong — the Z gap cited was mostly exploration bias |

### 2026-08-24 — `field_id`'s 21-bit ceiling

**[`2026-08-24-field-id-21bit/`](2026-08-24-field-id-21bit/)**

| | |
|---|---|
| ⚠️ | The packing **cannot represent the whole map**: 21 bits signed gives ±1,048,575, and **12.9% of real DD markers lie beyond it**. Bit 63 is unused, so there is no escape flag |
| ❌ | A report that decode failures are range-overflow cases **did not reproduce**: none of 19 observed misses is near the limit, the most extreme in-range value decodes correctly, and un-aliasing rescues none |
| ✅ | The misses are a **scanner under-count**, not a decode error — matched and missed rows have indistinguishable decoded-Z distributions |

Tool: `analyse_field_id.py` (takes a map name, so one map's rows can never be scored
against another map's actors).

### 2026-08-24 — External document validated: the "marker atlas"

**[`2026-08-24-marker-atlas-claims/`](2026-08-24-marker-atlas-claims/)** ·
**[Standalone report](2026-08-24-marker-atlas-claims/VALIDATION-REPORT.md)**

| | |
|---|---|
| ❌ | `marker` composite is `(marker_type, x, y, z, payload_type)` — five fields, not the claimed four |
| ❌ | `world_partition` holds **30** maps, not 39 |
| ❌ | `event_type = 0` (Shai-Hulud) does not exist; the document's own worm query returns nothing |
| ⚠️ | Spice tier **values** correct; **names** doubtful — a `SpiceFieldMedium` type suggests Small/Medium/Large |
| ⚠️ | **Every per-type count is *not checkable*, not wrong** — Hagga accumulates discoveries forever, DD is re-rolled per Coriolis seed. `long_range` content, which *is* server-independent, agrees within 1.36× with three exact matches |
| ✅ | All 26 named POI/vendor/trainer/House types exist; the SQL works; `field_kind_id` semantics correct |

---

## Maintaining this file

When an investigation produces something worth keeping:

1. Create `findings/YYYY-MM-DD-<topic>/` with a `README.md`, the raw evidence, and any
   re-runnable tooling under `tools/`.
2. Add a section here **in the same session**, with a status marker per claim and a link
   to the evidence.
3. If the finding **changes or disproves an existing entry, edit that entry in place** and
   say so — do not leave two rows disagreeing. Corrections are findings too, and the
   entries above deliberately record several.
4. Update the "Current state at a glance" table if the finding moves a capability.
5. Re-publish the shared page (below) so the external view does not drift.

Large raw captures should be reduced or gzipped before committing; keep the full copy on
the scan host under a persistent path, never `/tmp`.

## Sharing externally

This repo is **private**, so the links above are useless to anyone without access, and the
evidence files carry operational detail that should not leave it.

The shareable view is a **published web page** (an Artifact) that is self-contained —
it carries the substance rather than links into the repo. It starts private to the
publisher and is shared deliberately from the page's own share menu.

**Live page:** <https://claude.ai/code/artifact/a71d45b1-b8a4-4aba-b33f-359a003653d2>
**Source:** [`shared/findings-index.html`](shared/findings-index.html) — edit that file and
re-publish it to the **same URL**; publishing a different path creates a second, competing
page.

**Sanitisation rules — apply before publishing, every time:**

| Never publish | Why |
|---|---|
| Host aliases (`dune-dev`, `dune-prod`) and any `192.168.*` address | Internal topology |
| Player FLS IDs and character names | Personal identifiers |
| SSH commands, absolute host paths | Operational detail |
| Database credentials or role names | Requirement 5 / 24 |

**Safe to publish:** findings and conclusions, marker/record counts, coverage percentages,
game-side type names, memory offsets and record layouts, and coordinates (game world data).

Before publishing, run the sanitisation check:

```sh
grep -nE 'dune-dev|dune-prod|192\.168\.|BeretGenesis|ssh |/home/' findings/shared/findings-index.html
```

It must return nothing. Keep the `<title>` and favicon stable across republishes — readers
find the page by its name and tab icon.
