#!/usr/bin/env bash
# post-storm-scan.sh -- fired once by systemd-run after the DD Coriolis storm is
# expected to have regenerated the map (~0400 storm, this runs ~0530).
#
# Never assumes a PID: the DeepDesert_1 process that exists when this fires is
# guaranteed to be a different PID than any process seen earlier this session,
# because a map regeneration respawns the server process. PID lookup matches on
# RSS (>100MB) to skip the /bin/sh wrapper, which shares a truncated `comm`
# with the binary it launches -- plain `pgrep -f DeepDesert_1` also matches an
# ssh/bash command line containing that string.
#
# Self-contained deliberately: everything this script needs (this script
# itself and the census binary) lives under ~/scan-findings/, not /tmp, because
# /tmp on this host is tmpfs and this job is scheduled ~18h in advance -- a
# reboot in that window would silently wipe anything staged there.
set -euo pipefail

BASE="$HOME/scan-findings"
BIN="$BASE/bin/census"
OUT="$BASE/post-storm-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT"
LOG="$OUT/run.log"
exec > >(tee -a "$LOG") 2>&1

echo "=== post-storm scan starting: $(date -u -Iseconds) (local: $(date)) ==="

echo
echo "--- seed check ---"
dune database sql "SELECT unnest(map_names) AS m, unnest(map_seeds) AS s FROM dune.debug_get_coriolis_seeds();" \
  | tee "$OUT/seeds.txt"
SEED="$(awk -F'|' '/DeepDesert[ ]*\|/ && $1 ~ /^ *DeepDesert *$/ {gsub(/ /,"",$2); print $2}' "$OUT/seeds.txt")"
echo "DeepDesert seed at scan time: ${SEED:-<not parsed, see seeds.txt>}"
if [ "${SEED:-}" = "2" ]; then
  echo "NOTE: seed is still 2, the baseline observed continuously since session 2"
  echo "(2026-08-21). Either the storm has not happened yet, ran late, or"
  echo "regenerated back to the same seed. Do not assume this scan is against a"
  echo "genuinely new layout without checking this line first."
fi

echo
echo "--- resolving DeepDesert_1 PID (never hardcoded; retries in case the map"
echo "    is still mid-respawn) ---"
PID=""
for i in $(seq 1 10); do
  PID="$(ps -eo pid=,rss=,args= | awk '
    /DuneSandboxServer-Linux-Shipping/ && $0 ~ /[ ]DeepDesert_1([ ]|$)/ && $2 > 100000 { print $1; exit }
  ')"
  [ -n "$PID" ] && break
  echo "  DeepDesert_1 not found (attempt $i/10), retrying in 30s..."
  sleep 30
done
if [ -z "$PID" ]; then
  echo "FATAL: no DeepDesert_1 process found after 10 retries (5 min). Aborting."
  exit 1
fi
echo "DeepDesert_1 PID resolved: $PID"
# Redact args: the launch command line carries a live Funcom ServiceAuthToken
# (see findings/2026-08-22-technical-findings-report.md sec 11.1) -- never
# capture full `args` into a log that leaves this host.
ps -o pid,etime,rss,comm -p "$PID"

echo
echo "--- capturing markers ---"
dune database sql "$(cat <<'SQL'
COPY (
  SELECT (m.marker).marker_type, (m.marker).x, (m.marker).y, (m.marker).z, m.long_range, m.area_id
  FROM dune.markers m JOIN dune.map_names n ON n.map_name_id = m.map_name_id
  WHERE n.map_name = 'DeepDesert'
) TO STDOUT WITH (FORMAT csv, HEADER true);
SQL
)" > "$OUT/markers.csv"
echo "markers captured: $(($(wc -l < "$OUT/markers.csv") - 1))"

echo
echo "--- running census against PID $PID ---"
/usr/bin/time -f "elapsed=%es maxrss=%MkB" sudo timeout 600 "$BIN" -pid "$PID" -out "$OUT/census.jsonl"

echo
echo "=== post-storm scan complete: $(date -u -Iseconds) ==="
ls -la "$OUT"
