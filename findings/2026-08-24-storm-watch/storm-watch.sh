#!/usr/bin/env bash
# storm-watch.sh -- fired repeatedly by cron across a window, not once at a
# fixed time. "The seed will change around 0400" is an estimate, not a
# guarantee, so a single fixed-time shot is fragile against the storm running
# early, late, or landing on the same seed by chance. This checks the seed
# every firing and only actually scans once, the first time it sees a real
# change -- idempotent via a persisted state file, so repeated firings after
# a successful scan are nearly free (one query, then exit).
set -euo pipefail

BASE="$HOME/scan-findings"
STATE="$BASE/last-scanned-seed.txt"
LOG="$BASE/storm-watch.log"
BASELINE_SEED="2"   # the seed observed continuously since 2026-08-21

exec >> "$LOG" 2>&1
echo "--- storm-watch fired: $(date -u -Iseconds) ---"

SEEDS_OUT="$(dune database sql "SELECT unnest(map_names) AS m, unnest(map_seeds) AS s FROM dune.debug_get_coriolis_seeds();")"
SEED="$(printf '%s\n' "$SEEDS_OUT" | awk -F'|' '$1 ~ /^ *DeepDesert *$/ {gsub(/ /,"",$2); print $2}')"
echo "current DeepDesert seed: ${SEED:-<unparsed>}"

LAST="$(cat "$STATE" 2>/dev/null || echo "$BASELINE_SEED")"
echo "last-scanned seed: $LAST"

if [ "${SEED:-$LAST}" = "$LAST" ]; then
  echo "no change -- skipping"
  exit 0
fi

echo "SEED CHANGED ($LAST -> $SEED) -- running full post-storm scan"
if "$BASE/bin/post-storm-scan.sh"; then
  echo "$SEED" > "$STATE"
  echo "scan succeeded, recorded seed $SEED as scanned"
else
  echo "scan FAILED -- state file not updated, next firing will retry"
  exit 1
fi
