# Findings index

The maintained, top-level view of what this project has established, what it has
disproved, and where the evidence for each claim lives.

**This file is the entry point.** Every investigation gets an entry here in the same
session it happens — a finding that is not indexed is one nobody will locate in six
months, which is the failure mode this file exists to prevent.

- **Status legend:** ✅ measured · ❌ disproved by measurement · ⚠️ bounded or partly true ·
  🔹 inferred, not directly measured · ❓ open
- **Measured and inferred are marked differently on purpose.** An inference that reads like
  a measurement is how this file misleads someone six months from now, and an accuracy
  audit on 2026-08-24 found six claims stated more strongly than their evidence supported.
- **Every claim links to the evidence that supports it.** If a row has no link, it does
  not belong here yet.
- This file **is** the shareable artifact — see [Sharing externally](#sharing-externally).
  No separate generated page; GitHub renders it directly, and it pastes cleanly into
  Discord.

---

## Current state at a glance

| Capability | Status | Evidence |
|---|---|---|
| Node **positions**, including undiscovered ones | ✅ **58.5–64.3%** recall, whole map. 17 s on DeepDesert, 35.8 s on Hagga | [validation](2026-08-24-validation/README.md) |
| **Naming a resource class** (this class is TitaniumOre) | ✅ **Method works** — 1–3 confirmations per class | `CONTINUATION.md` §6 |
| **Typing the nodes the census finds directly** (memory/code) | ❌ **Cannot** — four RE sessions, all closed; the two channels barely overlap, see below | [issue-16](2026-08-24-issue-16/README.md), [static-reverse-engineering](2026-08-24-static-reverse-engineering/README.md) |
| **Typing nodes via bulk area discovery** (sidesteps the above) | 🔹 One event revealed 1,975 fully-typed nodes at once, area-wide — single observation, not yet a repeatable procedure | [bulk-area-discovery](2026-08-24-bulk-area-discovery/README.md) |
| Spice + Flour Sand positions from the DB | ⚠️ Exact for every field observed; a **21-bit ceiling** exists but no field beyond it has ever been seen | [field-id-21bit](2026-08-24-field-id-21bit/README.md) |
| Named POIs from the DB | 🔹 Every POI row is `long_range`; **completeness is inferred**, and post-storm timing is unobserved | `CONTINUATION.md` §4c |
| Scanner works with **zero players online** | ✅ **Proven** — 64.0% coverage with nobody logged in, identical to the with-player run | [experiment](2026-08-24-validation/zero-player-experiment.txt) |

### The type problem, stated precisely

This was previously summarised as "types unsolved", which was wrong, and an operator
challenge on 2026-08-24 forced the correction. Two true things were being conflated:

| Channel | Reach (same 211-marker box) | Carries a class? |
|---|---|---|
| Actors resolved by proximity scan | **6 / 211 = 2.8%** | ✅ yes — this is what §6 names |
| Census spawn records | **152 / 211 = 72.0%** | ❌ no |
| Census records with an actor within 1 m | **2 / 274 = 0.7%** | — the ceiling on borrowing a class |

(The 72% here is the WindPass test box specifically; the 58.5–64.3% in the table above is
map-wide across both maps. Different measurements, not a discrepancy.)

**A working method to name a class exists and is proven** (§6: scan a coordinate
`dune.markers` already labels; whatever class sits there is that resource). What does not
exist is a way to apply it to the 72% channel — only ~3% of nodes appear as actors at all,
and only 0.7% of census records have an actor close enough to borrow a class from.

So: *naming* is solved; *reaching* the nodes is not. Class pointers are also per-process,
so they must be re-derived after every restart, which needs at least one discovered marker
per type — unavailable at t=0 after a storm.

---

## Findings

### 2026-08-24 — Discovery reveals a whole `area_id` zone at once, not one node at a time

**[`2026-08-24-bulk-area-discovery/`](2026-08-24-bulk-area-discovery/)**

| | |
|---|---|
| ✅ | Found live, by accident, while testing something else: discovering one `Ecolab` structure added **1,977** new rows to `dune.player_markers` for the operator, not ~1-2 |
| ✅ | Joined the new rows back to `dune.markers`: 1,975 are ordinary, fully-typed resource nodes (every known type represented), clustered into just 4 `area_id` values -- **discovery is area-wide, keyed by `area_id`, not per-node proximity** |
| 🔹 | Practical implication, not yet acted on: systematically flying unexplored `area_id` zones is likely a far cheaper path to a fully-typed live map than any further record-level type attribution (see the RE entry below, now closed) -- DD has 57 total `area_id`s, this one partial excursion covered 2-3 |
| ❓ | Exact trigger boundary (structure interaction vs. just entering the zone) not isolated; generalization to Hagga or survival across a storm regeneration untested |

### 2026-08-24 — Post-storm scan hardened, tested, and actually scheduled

**[`2026-08-24-storm-watch/`](2026-08-24-storm-watch/)**

| | |
|---|---|
| ❌ | *Corrected*: this was reported as "still queued" in conversation while it had, in fact, never been scheduled -- the setup was built, then dropped mid-session. Caught when asked directly how to ensure it succeeds |
| ✅ | Replaced a fragile single-fixed-time plan with an idempotent windowed cron job (every 10 min, 04:00-07:59), self-bounded to one date |
| ✅ | Mechanism proven live before trusting it 17 hours unattended: 5 consecutive clean firings, `dune` CLI resolving correctly under cron's minimal environment (a real, common failure mode this catches) |
| ❌ | Two real bugs caught before deploy: a Python/bash quote-escaping mix-up that silently corrupted the first script version, and an awk permission-bit off-by-one |

### 2026-08-24 — Static/live reverse engineering, four sessions: type attribution now genuinely exhausted, not just stalled

**[`2026-08-24-static-reverse-engineering/`](2026-08-24-static-reverse-engineering/)**

| | |
|---|---|
| ✅ | **A real, unscanned 2.13 GB of read-only anonymous memory found and later closed** -- searched in the namepool-decode session below |
| ✅ | All 171 `dune` schema tables enumerated -- no analog to `resourcefield_state` exists for solid resources; the console/DB avenue is closed, architecturally, not by omission |
| ✅ | Zero resource-name strings exist anywhere in the 374MB binary itself (checked directly) -- closes static-string-XREF entirely |
| ✅ | `+320` confirmed a genuine, generic C++ vtable; `+336` traced further than `+280` ever did (its libc call's return value indexes a real lookup table), but the actual index requires single-stepping |
| ✅ | Installed `gdb`, live-traced `+336` for 162s across three windows including 90s of the operator standing next to confirmed nodes -- **zero hits, every window** |
| ✅ | **Full 48-field diff, live, of 372 matched records (pre-storm baseline vs. operator standing next to two of them): zero fields differ, anywhere** -- not just the three pointers, the complete record structure is proximity-invariant |
| ❌ | This refutes, not just fails to confirm, the "proximity promotes a record to a nameable actor" theory below -- corrected there too |

### 2026-08-24 — The FNamePool block format: fully decoded and live-verified

**[`2026-08-24-namepool-decode/`](2026-08-24-namepool-decode/)**

| | |
|---|---|
| ✅ | **Cracked the header encoding**: `(Length << 6) \| ProbeHash6`, plus 1-byte alignment padding after odd-length strings. Calibrated against ~25 known-length entries, not guessed once |
| ✅ | **Verified against 1,771 consecutive real entries, zero decode failures** -- a wrong format guess does not survive that |
| ✅ | **Reveals the complete resource taxonomy for free**: every mineral's full `BP_<Mineral>_[Static\|Pickup\|Ore]_[A-D]_[Component\|Spawner]_C` hierarchy, confirming names this project had only inferred before |
| ❌ | **Does not solve type attribution.** Checked both live-confirmed ground-truth records for a direct reference into this pool -- 4 and 2 matches respectively, both consistent with the ~2.7% chance-match baseline, none semantically relevant |
| ❌ | *Corrected, same day, later session*: the "proximity promotes a spawn slot to a real, nameable actor" theory these records were left on was tested directly (live `gdb` tracing plus a full-record diff, see the entry above) and **refuted**, not just left unconfirmed |



### 2026-08-24 — Issue #16 root cause: ore nodes are not actors

**[`2026-08-24-issue-16/`](2026-08-24-issue-16/)**

| | |
|---|---|
| ✅ | Pass 1 locates **211/211** known markers (100%); pass 2 resolves **6/211** (2.8%). The loss is entirely in actor resolution |
| ❌ | The hypothesis #16 was opened with — back-references outside the scanned regions — is **disproved**. Offset-agnostically, *nothing* points into the 2 KB before these transforms |
| 🔹 | Ore/scrap/pickup nodes appear to be **384-byte spawn records**, not UObject actors. The layout comes from **one** annotated memory dump; the census signature working across two maps supports it, but the record shape itself rests on a single sample. Spice and flour sand are unaffected |
| ✅ | A signature-based census reaches **64.3% DD / 58.5% Hagga**, at 136 MB. **17.0 s on DeepDesert, 35.8 s on Hagga** — the single "17 s" figure previously quoted was DeepDesert only |
| ❌ | **Four routes to typing a census record are ruled out** — the actor chain, all 48 record offsets, the object `+0` points at, and memory-address clustering. This is *not* the same as "types are unsolved": naming a class works (§6), it just cannot reach these records |

Evidence: the measured funnel before and after the NaN fix, offset-agnostic probes,
an annotated memory dump, and the census. Tools: `census.go`, `analyse_census.py`.

### 2026-08-24 — Validation: the scanner finds *undiscovered* nodes

**[`2026-08-24-validation/`](2026-08-24-validation/)**

| | |
|---|---|
| ✅ | **Retrospective test**: 60.2% of 1,667 markers discovered *after* the scan was captured, vs 64.8% of already-known ones. **Discovery is not required** |
| ✅ | **Three live confirmations, in-game -- including an unfiltered sample.** Flying into unexplored terrain, the operator confirmed `StravidiumOre` and `TitaniumOre` individually (1.8-10.9 m off, census predicted both within 1.1 m), then found a cluster of **five** never-before-seen nodes and checked all five, not just the hits: **3/5 matched the census (60%), landing almost exactly on the 60.2%/64.8% aggregate recall** measured earlier. The two misses were near-misses (107 m, 424 m), not wrong-area failures |
| ✅ | **Per-cell recall is uniform**: 61-79% across six independent well-explored cells, around the 64.0% map-wide mean — not an artifact of one region. **54 of 86 cells are essentially unexplored and hold 43,754 records**, over half the census, in terrain nobody has visited |
| ✅ | **Cross-map test**: the DD-derived signature gives 58.5% on Hagga — different map, authored terrain, a process that had restarted with fresh ASLR. Map- and process-independent |
| ✅ | **Zero-player operation PROVEN** (2026-08-24, after the operator logged out). The DeepDesert process stayed alive through T+120/240/330/420/540 s with **zero** online players — well past the autoscaler's 300 s grace period — and a full census then returned **84,569 records at 64.0% marker coverage, identical to the 64.0% measured with a player online**. This corrects the earlier "untested" entry, which the operator challenged and was right to |
| ❌ | *Corrected*: an earlier claim that unmatched records were "not all nodes" was largely wrong — the Z gap cited was mostly exploration bias |

### 2026-08-24 — `field_id`'s 21-bit ceiling

**[`2026-08-24-field-id-21bit/`](2026-08-24-field-id-21bit/)**

| | |
|---|---|
| ⚠️ | The packing **cannot represent the whole map**: 21 bits signed gives ±1,048,575, and **12.9% of real DD markers lie beyond it**. Bit 63 is unused, so there is no escape flag |
| 🔹 | **The impact on the spice/flour-sand layer is unquantified.** The 12.9% figure is over *markers*, not spice fields. Across all 141 `resourcefield_state` rows the largest decoded magnitude is **1,044,975** — just inside the limit — so **no field beyond the ceiling has ever been observed**. The practical impact may be zero; saying "only the inner 87% works" overstates what was measured |
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

### 2026-08-24 — Hagga's live process is `Survival_1`, not `Overmap`; the Jasmium discovery test finally ran

**[`2026-08-24-hagga-live-process-and-jasmium/`](2026-08-24-hagga-live-process-and-jasmium/)**

| | |
|---|---|
| ❌ | *Corrected*: `sessions/CONTINUATION-PROMPT.md`'s environment note ("Hagga Basin = the `Overmap` process") named the wrong process for live scanning. `dune.farm_state` shows `Overmap` at 0 connected players; the operator's actual session was on `Survival_1`. Scanning `Overmap` returned a static, geographically-clustered 112 records across three scans over 15 minutes, including one with the operator standing on a node — because that process was never running the operator's world |
| ✅ | The original `2026-08-24-validation/` Hagga numbers (58.5% recall, PID 2772183) were unaffected — that PID was `Survival_1` correctly at the time. Only this session's environment note, not the earlier data, was wrong |
| ✅ | **The Jasmium discovery test, queued since an earlier session, finally ran live**: operator discovered `JasmiumOre`/`JasmiumPickup` (zero prior `dune.markers` rows anywhere) in Hagga's "Shoel" sector. Scanning the correct process while standing on a node: only 6.1% of the 33 new markers matched within the standard 1 m threshold — but **100% (33/33) had a census record within 151 m**, most within 7-40 m. The census does find brand-new, just-discovered nodes; the fixed 1 m match radius understates recall for this type specifically |
| ❓ | Why Jasmium's positional offset runs larger than other types' — not established. *Corrected same session*: this row originally cited an operator claim that Jasmium is Hagga-exclusive as a possible explanation; the operator retracted it — **Primrose** (a water resource, absent from DD) is the Hagga-exclusive one, not Jasmium. Whether Jasmium exists on DD too, undiscovered, is unknown. Whether other low-recall types (DD's `*Pickup` gap, Hagga's `ErythriteOre` at 5.3%) show the same "close but past a too-strict threshold" pattern if re-checked with a looser radius — not checked |

## Accuracy audit — 2026-08-24

An operator challenge ("I thought we had the data to prove node types") triggered a full
re-check of this file. **Six claims were stated more strongly than their evidence
supported**, all in the same direction — inferences written as measurements, and one
outright wrong:

| Claim as written | Problem |
|---|---|
| "Node types ❌ unsolved" | **Wrong.** A working naming method exists (§6). What fails is reaching the census's nodes with it |
| "~60–64% recall" | Hagga is **58.5%** — the range understated its own low end |
| "in 17 s" | DeepDesert only; **Hagga took 35.8 s** |
| "only the inner ~87% of the map" | Measured over *markers*, not spice fields. **No field beyond the ceiling has ever been observed** |
| POIs "✅ complete without exploration" | An **inference** from `long_range` plus one operator report, written as a measurement |
| "are 384-byte spawn records" | Rests on **one** memory dump |

None of these were fabrications; each was a real result generalised past its evidence.
That is the specific failure mode this file now guards against with the 🔹 marker, and it
is why the legend distinguishes measured from inferred at all.

## Maintaining this file

When an investigation produces something worth keeping:

1. Create `findings/YYYY-MM-DD-<topic>/` with a `README.md`, the raw evidence, and any
   re-runnable tooling under `tools/`.
2. Add a section here **in the same session**, with a status marker per claim and a link
   to the evidence.
3. If the finding **changes or disproves an existing entry, edit that entry in place** and
   say so — do not leave two rows disagreeing. Corrections are findings too, and the
   entries above deliberately record several.
4. **State claims at the strength of their evidence.** One dump is not a population; one
   map's timing is not both maps'; a percentage over markers is not a percentage over
   spice fields. Use 🔹 when the claim is an inference and say what it was inferred from.
   If a claim cannot be traced to a number in `findings/`, it does not belong here.
5. Update the "Current state at a glance" table if the finding moves a capability.

Large raw captures should be reduced or gzipped before committing; keep the full copy on
the scan host under a persistent path, never `/tmp`.

## Sharing externally

**This file is the shareable artifact.** No separate generated page — it was tried
(GitHub Pages, built from this file) and dropped: GitHub renders this file directly and
well, and Markdown pastes cleanly into Discord as-is, which a generated HTML page never
could. One artifact, no drift risk between a source and a copy, no browser-rendering
uncertainty to chase.

- **View in browser:** the GitHub URL for this file, rendered natively —
  <https://github.com/Project-Arrakis/dune-resource-scanner/blob/main/findings/README.md>
- **Paste into Discord:** copy the section you need directly. Headers, bold, links, and
  code blocks all carry over; Discord does not render Markdown tables, so a pasted table
  shows as raw pipe-delimited text — prefer quoting the surrounding prose plus the specific
  numbers if a table's content needs to go into a message.

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
