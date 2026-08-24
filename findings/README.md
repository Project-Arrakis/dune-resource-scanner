# Findings index

The maintained, top-level view of what this project has established, what it has
disproved, and where the evidence for each claim lives.

**This file is the entry point.** Every investigation gets an entry here in the same
session it happens — a finding that is not indexed is one nobody will locate in six
months, which is the failure mode this file exists to prevent.

- **Status legend:** ✅ established · ❌ disproved · ⚠️ partly true / bounded · ❓ open
- **Every claim links to the evidence that supports it.** If a row has no link, it does
  not belong here yet.
- A **shareable** version of this index is published as a web page — see
  [Sharing externally](#sharing-externally). The page carries the substance rather than
  links, so it reads standalone.

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

**This repository is public.** It was private when these findings were first written, and
that changed on 2026-08-24 — so the links above do resolve for anyone, and the rule below
matters more than it did, not less.

The published page is a **self-contained** view: it carries the substance rather than
links, so it reads standalone and does not require a reader to navigate the repo.

**Live page:** <https://claude.ai/code/artifact/a71d45b1-b8a4-4aba-b33f-359a003653d2>
**Source:** [`shared/findings-index.html`](shared/findings-index.html) — edit that file and
re-publish it to the **same URL**; publishing a different path creates a second, competing
page. Keep the `<title>` and favicon stable, since readers find the page by name and tab
icon.

### What must never be committed

| Never commit | Why |
|---|---|
| Player identifiers (`Name#NNNN`) and in-game character names | Personal identifiers |
| Any RFC1918 address (`192.168.*`, `10.*`, `172.16–31.*`) | Internal topology |
| Email addresses | Personal identifiers |
| Database credentials or role names | Requirement 5 / 24 |

Use placeholders instead — `<fls-id>`, `<character>`, `<dev-vm-ip>`, `<scan-host>`. **Host
aliases are fine and should be kept**: they carry no address, and the operational docs are
useless without them.

**Safe to publish:** findings and conclusions, marker and record counts, coverage
percentages, game-side type names, memory offsets and record layouts, and in-game
coordinates.

### Enforcing it

```sh
./tools/check-public-safe.sh
```

Run it before publishing and before any push that touches documentation. It greps tracked
files for the patterns above and exits non-zero on a hit. The patterns are generic by
design — hard-coding the literal values would reintroduce exactly what it exists to remove.

A redaction pass ran on 2026-08-24 covering player identifiers, character names, internal
addresses and a hostname across six files. **Redaction limits future exposure only** —
those values remain in git history, and removing them there would require a history
rewrite across already-merged PRs, which was judged disproportionate given that no
credential was ever exposed (gitleaks is clean across the full history).
