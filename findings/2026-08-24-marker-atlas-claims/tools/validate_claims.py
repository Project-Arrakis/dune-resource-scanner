#!/usr/bin/env python3
"""Validate third-party claims about dune.markers against a live export.

  validate_claims.py <marker_counts.csv>

marker_counts.csv is produced by:
  COPY (SELECT (m.marker).marker_type, count(*) FROM dune.markers m GROUP BY 1)
  TO STDOUT WITH (FORMAT csv, HEADER true);

The point of this tool is NOT to declare claimed counts wrong. Counts differ
legitimately between servers for two independent reasons, and the tool exists to
separate what is checkable from what is not:

  1. Exploration. `dune.markers` is discovery-driven for `long_range=false` rows,
     so a busier server has strictly more of them. HaggaBasin is authored and
     never resets, so its counts accumulate indefinitely.
  2. Coriolis seed. DeepDesert is procedurally re-rolled each storm, so DD node
     counts are a property of the live seed, not of the game. Two servers on
     different seeds legitimately differ in both directions.

Only `long_range` content (revealed at range, complete without exploration) can
be compared as fact. Everything else is a snapshot.
"""
import csv, sys

# Claimed counts, from the third-party document (2026-08-24).
CLAIMS = {
    "Cave": 116, "Ecolab": 19, "Sietch": 8, "TitaniumOre": 178, "TitaniumPickup": 130,
    "ErythriteOre": 95, "ErythritePickup": 8, "StravidiumOre": 66, "StravidiumPickup": 29,
    "JasmiumOre": 18, "AzuriteOre": 3067, "AzuritePickup": 1834, "RhyoliteOre": 4761,
    "RhyolitePickup": 2920, "MagnetiteOre": 620, "MagnetitePickup": 310, "BauxiteOre": 556,
    "BauxitePickup": 280, "BasaltOre": 1031, "BasaltPickup": 490, "DolomiteRock": 699,
    "DolomitePickup": 340, "Shipwreck": 30, "BrittleBush": 1372, "PrimroseField": 1199,
    "SaguaroSeed": 23, "EnemyCamp": 446, "EnemyOutpost": 72, "EnemyLaborOutpost": 14,
    "Hazard_Quicksand": 180, "Hazard_Radiation": 56, "Hazard_Drumsand": 40,
}
CLAIMED_TOTAL = 23413

# Verified 2026-08-24: 100% of rows of these types carry long_range=true, so they
# are revealed at range and are complete without exploration.
LONG_RANGE = {"Cave", "Ecolab", "Sietch", "Shipwreck", "EnemyCamp", "EnemyOutpost",
              "EnemyLaborOutpost", "TaxiService", "TradingPost"}


def main():
    real = {r["marker_type"]: int(r["count"]) for r in csv.DictReader(open(sys.argv[1]))}
    total = sum(real.values())
    print(f"claimed total {CLAIMED_TOTAL}   actual {total}   ratio {CLAIMED_TOTAL/total:.2f}x\n")

    for group, label in ((True, "long_range — server-independent, CHECKABLE as fact"),
                         (False, "discovery-driven — a per-server, per-seed snapshot, NOT checkable")):
        rows = [(k, v) for k, v in CLAIMS.items() if (k in LONG_RANGE) == group]
        print(f"=== {label} ===")
        print(f"{'type':<20}{'claimed':>9}{'actual':>9}{'ratio':>9}")
        worst = 0.0
        for k, c in sorted(rows, key=lambda kv: -kv[1]):
            a = real.get(k)
            if a is None:
                # Absent here does not mean the type is invalid -- it may simply
                # be undiscovered on this server. JasmiumOre is the known case:
                # Hagga-only and never located here (CONTINUATION.md section 11).
                print(f"{k:<20}{c:>9}{'absent':>9}   not present here; may be undiscovered")
                continue
            r = c / a
            worst = max(worst, r if r > 1 else 1 / r)
            print(f"{k:<20}{c:>9}{a:>9}{r:>8.2f}x")
        print(f"  worst deviation in this group: {worst:.2f}x\n")


if __name__ == "__main__":
    main()
