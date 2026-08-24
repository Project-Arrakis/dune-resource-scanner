# `field_id`'s 21-bit ceiling — verified, and what it does *not* explain

Prompted by a report from another developer: the `field_id` decode works for
~75% of Large spice and "fails only past ±1,048,575", with every miss claimed to
be a range-overflow case and every hit in range.

Verified live against `dune-dev` (DeepDesert PID 390735, Coriolis seed 2) on
2026-08-24. Re-run any time with:

```
tools/analyse_field_id.py DeepDesert resourcefield_state.csv spice-seed.json dd-markers-prestorm.csv
```

## The premise is correct

`CONTINUATION.md` §2 decodes three 21-bit signed fields, so the representable
range is **−1,048,576 .. +1,048,575**. Deep Desert is bigger than that:

- Real marker extent: X `−1,172,411 .. 1,055,765`, Y `−1,092,600 .. 1,121,130`
- **1,237 of 9,601 real DD markers (12.9%) lie beyond the limit** — those
  positions simply cannot be encoded in this packing.

Bit 63 is unused (`0` in all 110 DD rows and all 141 rows overall), so there is
no escape flag or extra bit hiding in the word.

**This is a real structural limit and it constrains the Live Map**: the
spice/flour-sand layer, which `CONTINUATION.md` §11 Track B lists as "Ready", is
only demonstrated for the inner ~87% of the map.

## The conclusion does not reproduce

Matching each DD row's decoded position against 909 memory-scanned actors,
position-only (`value_remaining` is deliberately **not** used as a tier label —
it decreases as a field is harvested, so it does not identify the tier):

| | |
|---|---:|
| DD rows | 110 |
| Decoded within 50 uu of a memory actor | **91 (82.7%)** |
| Misses | 19 |
| **Misses within 10% of the 21-bit limit** | **0** |
| Largest \|coordinate\| among misses | **803,675** |
| Largest \|coordinate\| among all rows | **1,044,975** (matches fine) |
| Misses rescued by un-aliasing (±2²¹ on either axis) | **0 / 19** |

The pattern is the opposite of the one predicted. The most extreme in-range value
in the whole set — 1,044,975, sitting on the ceiling — **decodes correctly**,
while misses top out at 803,675, comfortably inside the range.

## What the misses actually are

The decoded Z of matched and missed rows is statistically indistinguishable:

| | p10 | median | p90 |
|---|---:|---:|---:|
| matched (n=91) | 1,613 | 4,010 | 7,514 |
| missed (n=19) | 2,070 | 3,735 | 6,604 |

A wrong decode does not produce plausible, identically-distributed terrain
heights. **The decode is working on the missed rows too; they are absent from the
memory scan.** That is already on record in `CONTINUATION.md` §8 — "14 of 59
active flour-sand fields decode to positions absent from the memory scan's 533 —
the scan under-counts. Cause unknown." These 19 are the same phenomenon,
now measured across all tiers rather than flour sand alone.

**The direction matters**: this is evidence against the *scanner*, not the decode.

## Large spice: cannot be tested here

DD currently holds only **2** rows at `value_remaining = 2,500,000`, and both
decode to the *same* position, `(−812800, −1016000, −4144)`, each matching a
memory actor at 0 uu. But that match is against one of the 6 `spice-large` hits
that §8 already flags as **false positives** — grid-round coordinates
(−304800 / −1016000) and an identical Z of −4143.93, which is exactly the
`−4144` decoded here.

So the Large case is entangled with a known scanner artifact rather than being a
clean test. The other developer evidently has more Large fields than these two,
so they may be observing something real that this dataset cannot show.

## The question that settles it

**Is there a Large spice field whose true position — from memory, not the DB —
exceeds ±1,048,575 on X or Y, and what does its `field_id` decode to?**

- If it aliases to a plausible-looking wrong position, that is the dangerous
  case: silently wrong coordinates rather than missing ones, and the spice layer
  needs a range guard before it ships.
- If no such field exists in any dataset, the likelier reading is that the game
  does not place spice in the outer band at all, and the 21-bit width is correct
  by design.

Note the highest decoded magnitude observed anywhere here is 1,044,975 — just
under the ceiling — which is consistent with *either* reading and so cannot
settle it alone.

## Files

| File | What it is |
|---|---|
| `resourcefield_state.csv` | All 141 `dune.resourcefield_state` rows, both maps |
| `spice-seed.json` | `-mode seed` scan: 909 actors (spice 281/89/6, flour sand 533) |
| `dd-markers-prestorm.csv` | 9,601 DeepDesert markers, for the real map extent |
| `analysis.txt` | Output of the tool below |
| `tools/analyse_field_id.py` | Re-runnable; takes a map name so one map's rows can never be scored against another map's actors |
