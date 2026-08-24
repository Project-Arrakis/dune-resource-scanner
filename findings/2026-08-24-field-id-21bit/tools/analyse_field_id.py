#!/usr/bin/env python3
"""Verify the field_id bit-packing decode against memory-scanned ground truth.

Tests two separate claims:
  1. Whether the 21-bit signed fields can represent the whole map.
  2. Whether decode failures are caused by range overflow.

  analyse_field_id.py <map> <resourcefield_state.csv> <seed-scan.json> [markers.csv]

<map> must match the map the seed scan was taken from (e.g. DeepDesert).
Comparing one map's rows against another map's memory actors silently turns
every row into a "miss" -- the filter exists so that cannot happen by accident.

resourcefield_state.csv : field_id,map,dimension_index,value_remaining,field_kind_id
seed-scan.json          : dune-resource-scanner -mode seed output (ground truth positions)
markers.csv             : optional dune.markers export, to measure real map extent
"""
import csv, json, math, sys, collections

LIMIT = 0xFFFFF          # 1,048,575 -- largest positive value a 21-bit signed field holds
WRAP = 0x200000          # 2,097,152 -- the aliasing period
MATCH_UU = 50.0


def decode(fid):
    """The documented 21/21/21 signed packing (CONTINUATION.md section 2)."""
    out = []
    for shift in (0, 21, 42):
        v = (fid >> shift) & 0x1FFFFF
        if v > LIMIT:
            v -= WRAP
        out.append(v)
    return out


def quant(vals, p):
    vals = sorted(vals)
    return vals[int(p * (len(vals) - 1))]


def main():
    want_map = sys.argv[1]
    allrows = list(csv.DictReader(open(sys.argv[2])))
    rows = [r for r in allrows if r["map"] == want_map]
    acts = json.load(open(sys.argv[3]))
    mem = [(a["X"], a["Y"]) for a in acts]
    markers = []
    if len(sys.argv) > 4:
        for r in csv.DictReader(open(sys.argv[4])):
            markers.append((float(r["x"]), float(r["y"])))

    print(f"map: {want_map}   rows for this map: {len(rows)} of {len(allrows)}   memory actors: {len(mem)}")
    if not rows:
        print("no rows for that map -- check the map name")
        return
    print(f"21-bit signed range: {-WRAP//2} .. {LIMIT}")

    # Claim 1: can the packing represent the whole map?
    if markers:
        beyond = sum(1 for x, y in markers if abs(x) > LIMIT or abs(y) > LIMIT)
        print(f"\n=== claim 1: representability ===")
        print(f"real marker extent: X {min(m[0] for m in markers):.0f}..{max(m[0] for m in markers):.0f}"
              f"  Y {min(m[1] for m in markers):.0f}..{max(m[1] for m in markers):.0f}")
        print(f"markers beyond the 21-bit limit: {beyond}/{len(markers)} "
              f"({100*beyond/len(markers):.1f}%) -- these positions CANNOT be encoded")

    # Claim 2: are decode failures range-overflow cases?
    hits, misses = [], []
    bit63 = collections.Counter()
    for r in rows:
        fid = int(r["field_id"])
        bit63[(fid >> 63) & 1] += 1
        x, y, z = decode(fid)
        d = min(math.hypot(mx - x, my - y) for mx, my in mem) if mem else float("inf")
        (hits if d <= MATCH_UU else misses).append((x, y, z, d, r))

    print(f"\n=== claim 2: are misses caused by overflow? ===")
    print(f"bit 63 (unused by a 21/21/21 packing): {dict(bit63)}")
    print(f"decoded within {MATCH_UU:.0f}uu of a memory actor: {len(hits)}/{len(rows)} "
          f"({100*len(hits)/max(1,len(rows)):.1f}%)")
    if not misses:
        print("no misses; nothing further to test")
        return
    mag = lambda e: max(abs(e[0]), abs(e[1]))
    near = sum(1 for e in misses if mag(e) > 0.90 * LIMIT)
    print(f"misses: {len(misses)}   of which within 10% of the limit: {near}")
    print(f"largest |coordinate| among MISSES: {max(mag(e) for e in misses)}")
    print(f"largest |coordinate| among ALL rows: {max(mag(e) for e in hits + misses)}")
    if near == 0:
        print("=> misses are NOT range-overflow cases: every one is comfortably in range.")

    # Aliasing test: would un-wrapping rescue any miss?
    rescued = 0
    for x, y, z, d, r in misses:
        for shift in (WRAP, -WRAP):
            for nx, ny in ((x + shift, y), (x, y + shift)):
                if mem and min(math.hypot(mx - nx, my - ny) for mx, my in mem) <= MATCH_UU:
                    rescued += 1
    print(f"misses rescued by un-aliasing (+/-2^21 on either axis): {rescued}/{len(misses)}")

    # Is the decode itself sane on the misses? Compare Z distributions.
    zh = [e[2] for e in hits]
    zm = [e[2] for e in misses]
    if zh and zm:
        print(f"\n=== is the decode wrong on the misses, or is memory incomplete? ===")
        print(f"decoded Z, matched rows (n={len(zh)}): p10={quant(zh,.1)} med={quant(zh,.5)} p90={quant(zh,.9)}")
        print(f"decoded Z, missed rows  (n={len(zm)}): p10={quant(zm,.1)} med={quant(zm,.5)} p90={quant(zm,.9)}")
        print("Indistinguishable distributions of plausible terrain heights mean the decode is")
        print("working on the misses too -- they are absent from the memory scan, not mis-decoded.")


if __name__ == "__main__":
    main()
