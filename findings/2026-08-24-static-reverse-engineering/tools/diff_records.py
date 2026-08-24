import json, csv, math, sys

MARKERS = '/home/dune/scan-findings/pre-storm-20260824T195710Z/markers.csv'
BEFORE = '/home/dune/scan-findings/pre-storm-20260824T195710Z/census.jsonl'
AFTER = '/home/dune/scan-findings/re/census-now.jsonl'
TYPES = ('BauxiteOre', 'DolomiteRock', 'DolomitePickup', 'BauxitePickup')
RADIUS = 150.0

def load_census(path):
    recs = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            recs.append(json.loads(line))
    return recs

def nearest(recs, x, y):
    best, bi = 1e18, None
    for i, r in enumerate(recs):
        d = math.hypot(r['x']-x, r['y']-y)
        if d < best:
            best, bi = d, i
    return best, bi

markers = []
with open(MARKERS) as f:
    for r in csv.DictReader(f):
        if r['marker_type'] in TYPES:
            markers.append((r['marker_type'], float(r['x']), float(r['y'])))

print(f'candidate markers: {len(markers)}')
before = load_census(BEFORE)
after = load_census(AFTER)
print(f'before records: {len(before)}  after records: {len(after)}')

diffs_found = 0
for mtype, mx, my in markers:
    db, ib = nearest(before, mx, my)
    da, ia = nearest(after, mx, my)
    if db > RADIUS or da > RADIUS:
        continue
    rb, ra = before[ib], after[ia]
    fb, fa = rb.get('fields', {}), ra.get('fields', {})
    diffkeys = [k for k in fb if fb.get(k) != fa.get(k)]
    if diffkeys:
        diffs_found += 1
        print(f'DIFF at {mtype} ({mx:.0f},{my:.0f}) dist_before={db:.1f} dist_after={da:.1f}')
        for k in sorted(diffkeys, key=lambda x: int(x)):
            print(f'    [{k:>4}] before={fb.get(k)!r}  after={fa.get(k)!r}')
print(f'total markers checked (both matched within {RADIUS}uu): -- see above')
print(f'records with ANY field difference: {diffs_found}')
