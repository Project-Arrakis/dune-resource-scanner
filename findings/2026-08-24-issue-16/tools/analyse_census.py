#!/usr/bin/env python3
"""Offline analysis of a census scan against the full dune.markers ground truth.

Investigation tool for issue #16. Runs locally -- it must never need the game
host, so the expensive work happens once in the scan and everything after is
re-runnable for free.

  analyse_census.py <census.jsonl> <markers.csv>
"""
import csv, json, math, sys, collections

MATCH_UU = 100.0  # 1 metre


def load(census_path, markers_path):
    recs = []
    with open(census_path) as f:
        for line in f:
            line = line.strip()
            if line:
                recs.append(json.loads(line))
    markers = []
    with open(markers_path) as f:
        for r in csv.DictReader(f):
            try:
                markers.append((r["marker_type"], float(r["x"]), float(r["y"]), float(r["z"])))
            except (ValueError, KeyError):
                continue
    return recs, markers


def grid_index(recs, cell=1000.0):
    idx = collections.defaultdict(list)
    for i, r in enumerate(recs):
        idx[(int(r["x"] // cell), int(r["y"] // cell))].append(i)
    return idx, cell


def nearest(idx, cell, recs, x, y):
    cx, cy = int(x // cell), int(y // cell)
    best, bi = float("inf"), None
    for dx in (-1, 0, 1):
        for dy in (-1, 0, 1):
            for i in idx.get((cx + dx, cy + dy), ()):
                r = recs[i]
                d = math.hypot(r["x"] - x, r["y"] - y)
                if d < best:
                    best, bi = d, i
    return best, bi


def main():
    recs, markers = load(sys.argv[1], sys.argv[2])
    print(f"census records: {len(recs)}   markers: {len(markers)}")
    if not recs:
        print("no records; nothing to analyse")
        return
    idx, cell = grid_index(recs)

    matched = {}
    by_type = collections.defaultdict(lambda: [0, 0])
    for t, x, y, z in markers:
        d, i = nearest(idx, cell, recs, x, y)
        by_type[t][1] += 1
        if d <= MATCH_UU:
            by_type[t][0] += 1
            matched.setdefault(i, []).append(t)

    tot_hit = sum(v[0] for v in by_type.values())
    print(f"\n=== marker coverage (record within {MATCH_UU:.0f} uu) ===")
    print(f"{'type':24}{'matched':>8}{'total':>8}{'pct':>7}")
    for t in sorted(by_type, key=lambda k: -by_type[k][1]):
        hit, n = by_type[t]
        print(f"{t:24}{hit:>8}{n:>8}{100*hit/n:>6.1f}%")
    print(f"{'TOTAL':24}{tot_hit:>8}{len(markers):>8}{100*tot_hit/len(markers):>6.1f}%")

    unmatched_recs = len(recs) - len(matched)
    print(f"\nrecords matching >=1 marker: {len(matched)}  ({100*len(matched)/len(recs):.1f}% of records)")
    print(f"records matching no marker:  {unmatched_recs}  -- either undiscovered nodes (the point of the tool) or false positives")

    # Type-discriminator search over the captured record fields.
    print("\n=== type-discriminator search over captured record offsets ===")
    fields = sorted({k for r in recs for k in r["fields"]}, key=int)
    rows = []
    for off in fields:
        vals_by_type = collections.defaultdict(collections.Counter)
        for i, ts in matched.items():
            v = recs[i]["fields"].get(off)
            if v is None:
                continue
            for t in set(ts):
                vals_by_type[t][v] += 1
        owner = collections.defaultdict(set)
        for t, c in vals_by_type.items():
            for v in c:
                owner[v].add(t)
        exclusive = 0
        covered = 0
        for t, c in vals_by_type.items():
            n_t = by_type[t][0]
            if n_t < 3:
                continue
            best_v, best_n = None, 0
            for v, n in c.items():
                if len(owner[v]) == 1 and n > best_n:
                    best_v, best_n = v, n
            if best_v is not None and best_n >= 0.8 * n_t:
                exclusive += 1
                covered += best_n
        distinct = len(owner)
        collisions = sum(1 for v, ts in owner.items() if len(ts) > 1)
        rows.append((off, exclusive, covered, distinct, collisions))
    rows.sort(key=lambda r: (-r[1], -r[2], r[4]))
    print(f"{'offset':>8}{'types>=80% exclusive':>22}{'markers covered':>17}{'distinct':>10}{'collisions':>12}")
    for off, ex, cov, dist, coll in rows[:20]:
        print(f"{off:>8}{ex:>22}{cov:>17}{dist:>10}{coll:>12}")

    best = rows[0] if rows else None
    if best and best[1] > 0:
        off = best[0]
        print(f"\n=== values at record offset {off}, by marker type ===")
        vals_by_type = collections.defaultdict(collections.Counter)
        for i, ts in matched.items():
            v = recs[i]["fields"].get(off)
            if v is None:
                continue
            for t in set(ts):
                vals_by_type[t][v] += 1
        for t in sorted(vals_by_type, key=lambda k: -by_type[k][0]):
            top = vals_by_type[t].most_common(3)
            s = "  ".join(f"{v:#x}({n})" for v, n in top)
            print(f"  {t:24}{s}")

    extra(recs, markers, matched, by_type)


def extra(recs, markers, matched, by_type):
    """Attribution routes tested and ruled out, plus an honest look at what the
    unmatched records are. Kept in the tool so the negative results are
    re-derivable rather than folklore."""
    import collections as C

    # Unambiguous single-type labels: the record's matched markers agree, and no
    # marker of a different type sits within 300 uu to muddy the label.
    idx, cell = grid_index(recs)
    near300 = C.defaultdict(set)
    for t, x, y, z in markers:
        cx, cy = int(x // cell), int(y // cell)
        for dx in (-1, 0, 1):
            for dy in (-1, 0, 1):
                for i in idx.get((cx + dx, cy + dy), ()):
                    if math.hypot(recs[i]["x"] - x, recs[i]["y"] - y) <= 300:
                        near300[i].add(t)
    clean = {i: ts[0] for i, ts in matched.items() if len(set(ts)) == 1 and len(near300[i]) == 1}
    print(f"\nunambiguous single-type labelled records: {len(clean)}")

    # Route: do same-type records sit next to each other in memory?
    lab = sorted(clean.items(), key=lambda kv: recs[kv[0]]["base"])
    same = sum(1 for a, b in zip(lab, lab[1:]) if a[1] == b[1])
    pairs = max(1, len(lab) - 1)
    cnt = C.Counter(clean.values())
    chance = sum(n * n for n in cnt.values()) / max(1, len(clean) ** 2)
    print(f"address-adjacent labelled pairs sharing a type: {100*same/pairs:.1f}% "
          f"(chance {100*chance:.1f}%) -- real signal, far too weak to classify")

    # Honest look at the unmatched majority.
    un = [r for i, r in enumerate(recs) if i not in matched]
    ma = [recs[i] for i in matched]
    if un and ma:
        zu = sorted(r["z"] for r in un)
        zm = sorted(r["z"] for r in ma)
        print(f"\nZ profile  matched: median {zm[len(zm)//2]:.0f}   unmatched: median {zu[len(zu)//2]:.0f}")
        print("A large Z gap means the unmatched records are NOT all resource nodes.")
        mc = C.Counter(_cell(m[1], m[2]) for m in markers)
        rc = C.Counter(_cell(r["x"], r["y"]) for r in recs)
        print("\nrecord:marker ratio in the best-explored 270k cells")
        for k, n in mc.most_common(6):
            print(f"  cell {str(k):10} markers={n:5}  records={rc.get(k,0):6}  {rc.get(k,0)/n:5.1f}x")


def _cell(x, y):
    return (int((x + 52656) // 270000), int((y + 52066) // 270000))


if __name__ == "__main__":
    main()
