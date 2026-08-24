# The FNamePool block format: fully decoded, live-verified — with an honest limit

Direct continuation of `../2026-08-24-static-reverse-engineering/`. That investigation's
disassembly hinted at FNamePool-style resolution (a cached lookup index in writable
memory, a block-masked pointer, length/flag byte reads) but the traced call resolved to
a generic `libc` function — inconclusive. This picks the thread back up from the memory
side instead of the disassembly side, and this time it closes cleanly.

## The format, empirically derived and verified

Found the pool by searching the class of memory `census.go` had never covered — anonymous
regions, both writable and read-only, for literal ASCII strings (`scanro.go`). A hit for
`_Ore_Node_Component` landed inside a dense run of short, back-to-back class-name strings —
structurally exactly what an FNamePool block looks like.

Calibrated the header encoding against ~25 consecutive entries with independently
verifiable lengths (counted directly from the decoded ASCII), rather than guessing once:

```
[2-byte LE header][Length raw ASCII bytes][1 pad byte if Length is odd]
header = (Length << 6) | ProbeHash6   -- low 6 bits are a comparison hash, not string data
```

The one wrinkle — 1 byte of alignment padding after odd-length strings, none after
even-length ones — was found the same way: a decode that failed immediately after a
29-character string, with the very next 2 "header" bytes turning out to be `\x00` (padding)
followed by the real header once the correct 1-byte skip was applied.

**Verified, not assumed:** walking forward from a known-good anchor decoded **58 consecutive
entries with zero failures** on the first pass. Then built a real Go decoder
(`tools/namepool.go`) and ran it against the **full live 64KB block**: **1,771 entries,
zero decode failures, zero garbage.** That is the actual proof this is correct — a wrong
guess at the format does not survive 1,771 consecutive successful decodes by accident.

## What it reveals: the complete resource taxonomy, for free

The one block scanned happens to be exactly the block holding every mineral's blueprint
class hierarchy. For **every** known resource — Titanium, Stravidium, Azurite, Bauxite,
Magnetite, Dolomite, Rhyolite/Stone, Basalt, Erythrite, Jasmium, ImpureFuel, ScrapMetal —
the same systematic naming convention appears in full:

```
BP_<Mineral>_Static_[A-D]_Component        -- visual/mesh variants
BP_<Mineral>_Pickup_[A-D]_Spawner_C        -- the small hand-collected pickups
BP_<Mineral>_[A-D]_Component               -- the vein/node itself
Default__BP_<Mineral>Ore_Spawner_C         -- the actual ore-vein spawner
```

This independently confirms naming this project had only inferred from `dune.markers`
row types before (e.g. `BP_BauxiteOre_Spawner_C` matching the `BauxiteOre` marker type
exactly), and reveals detail the marker table alone never showed — the A–D visual variant
system, the `_Cliffside` / `_Sietch` special-placement variants, and confirms `RhyoliteStone`
(not `Rhyolite`) is the real internal family name.

## The honest limit: this does NOT (yet) solve type attribution

The natural next question: does either of the two live-confirmed ground-truth records
(`StravidiumOre`, `TitaniumOre` — see `../2026-08-24-live-confirmation/`) contain a raw
reference into this pool?

Checked properly rather than assumed. Loaded all 1,771 valid entry offsets, computed the
baseline chance of a random 16-bit value matching one by coincidence (**2.7%**, i.e.
1771/65536), then scanned every byte position of both 384-byte records for a 32-bit value
whose low 16 bits equal a valid offset:

| Record | Matches found | Chance-level expectation (~380 positions × 2.7%) | Semantically relevant? |
|---|---:|---:|---|
| StravidiumOre | 4 | ~10 | No — generic engine class names (`TArray<EBlendProfileMode>`, etc.) |
| TitaniumOre | 2 | ~10 | No |

**Both results are consistent with pure noise, not a real reference.** Neither record
contains a direct pointer or packed index into this specific pool block.

### Why this is not a contradiction — it fits the day's other findings

This session already established, independently, that these records have **no actor, no
`ClassPrivate`, no resolvable class chain at all** (`../2026-08-24-issue-16/`), and that
their `+0` handle leads to pure per-instance geometry, not a class descriptor
(`../2026-08-24-static-reverse-engineering/`). A record with no reachable UClass has
nothing that would carry an FName in the first place — the NamePool being unreachable from
it is the expected consequence of that, not a new, separate failure.

**Refined structural picture, now better evidenced than this morning's version:** these
384-byte records most likely are not a lightweight *view* of a full UObject — they appear
to be a genuinely separate, pre-actor "spawn slot" layer: position, a per-instance heap
handle, and bounding geometry, with **no type identity of their own**. Resource *actors*
were separately confirmed resident map-wide (unlike POI structures), which is presumably
why positions are found reliably; but "resident" and "already a named UObject" turn out to
be different claims, and today's evidence points at the record layer sitting on the wrong
side of that line.

**Correction, same day, later session: the "player proximity promotes a slot to a full
actor" theory stated above has been tested directly and refuted, not just left
unconfirmed.** This was the leading candidate for how/when a real, nameable object might
come into existence, and it was specific enough to test properly rather than take on faith.
Two independent tests, both negative:

1. Breakpointed the `+336` record pointer's target function (traced in
   `../2026-08-24-static-reverse-engineering/README.md`, Session 2) and watched it live for
   162 seconds total across three windows, the last 90 seconds of which had the operator
   standing directly next to a confirmed `TitaniumOre` and `BauxiteOre`/Aluminum node. Zero
   hits, all three windows.
2. More directly: queried `dune.actors` for any Titanium/Bauxite/Ore-named class on
   DeepDesert while the operator stood next to both nodes. **Zero rows.** Broadened to every
   distinct class on the whole map (18 actors total, 9 distinct classes) to rule out a
   naming-pattern miss -- none are resource-node related at all (player character/
   controller/state, buildings, doors, a totem, a loot container, an ornithopter).

**Conclusion: these resource nodes are never represented in `dune.actors` at all, under any
class name, proximity or not.** Whatever makes a node visible, interactable, and
harvestable to a player is handled entirely client-side or through some other server
mechanism that never touches this table or this code path. This isn't a "we didn't catch it
in time" result -- 162 seconds including 90 of direct proximity is ample if promotion were
proximity-driven and routed through either of the two things tested. The specific mechanism
remains unknown; what's now closed is the specific "proximity promotes to a
DB-visible/nameable actor" theory as previously framed.

## What this does and does not change

**Does not** unblock type attribution for the census today. **Does** leave the project
holding a working, verified decoder for a real, previously-opaque data structure, plus the
complete confirmed resource taxonomy, reusable the moment any pointer *from* a record
*into* a nameable object is found. The "visit a predicted node, re-scan while standing
there, see if an actor now resolves" test this section originally proposed has since been
run (see the correction above) -- it came back negative, so this reusability is no longer
tied to that specific mechanism. It remains reusable if some *other*, still-unknown link
from record to nameable object is ever found.

## Files

- `tools/namepool.go` — the decoder, `//go:build ignore`. Given a live PID and any address
  inside a suspected block, block-aligns it, brute-forces the true entry-start offset (the
  block boundary is not an entry boundary), and decodes forward.
- `tools/scanro.go` — the string search that found the pool in the first place; extended
  during this session to cover the writable heap too, not just the read-only region it was
  originally built for.
- `tools/findref.go` — the direct-reference test against the two known records.
- `namepool-block1.txt.gz` — the full 1,771-entry decode, gzipped.
