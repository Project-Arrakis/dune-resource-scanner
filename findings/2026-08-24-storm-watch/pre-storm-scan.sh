#!/usr/bin/env bash
# pre-storm-scan.sh -- run manually, once, while the DeepDesert seed is still
# the long-lived baseline (2), to capture a same-format "before" snapshot to
# diff against whatever post-storm-scan.sh captures after the seed changes.
#
# Deliberately mirrors post-storm-scan.sh's output format exactly (seeds.txt,
# markers.csv, census.jsonl) so the two are directly comparable -- same
# columns, same census binary, same query. The only difference is this one
# resolves the PID once with no retry loop, since there is no respawn to wait
# out before the storm has happened.
set -euo pipefail

BASE="$HOME/scan-findings"
BIN="$BASE/bin/census"
OUT="$BASE/pre-storm-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT"
LOG="$OUT/run.log"
exec > >(tee -a "$LOG") 2>&1

echo "=== pre-storm baseline scan starting: $(date -u -Iseconds) (local: $(date)) ==="

echo
echo "--- seed check ---"
dune database sql "SELECT unnest(map_names) AS m, unnest(map_seeds) AS s FROM dune.debug_get_coriolis_seeds();" \
  | tee "$OUT/seeds.txt"
SEED="$(awk -F'|' '/DeepDesert[ ]*\|/ && $1 ~ /^ *DeepDesert *$/ {gsub(/ /,"",$2); print $2}' "$OUT/seeds.txt")"
echo "DeepDesert seed at scan time: ${SEED:-<not parsed, see seeds.txt>}"
if [ "${SEED:-}" != "2" ]; then
  echo "WARNING: seed is not the expected baseline (2) -- the storm may already"
  echo "have happened. This snapshot would then NOT be a valid pre-storm baseline."
fi

echo
echo "--- resolving DeepDesert_1 PID (never hardcoded; matched by RSS to skip the"
echo "    /bin/sh wrapper, which shares a truncated comm with the binary) ---"
PID="$(ps -eo pid=,rss=,args= | awk '
  /DuneSandboxServer-Linux-Shipping/ && $0 ~ /[ ]DeepDesert_1([ ]|$)/ && $2 > 100000 { print $1; exit }
')"
if [ -z "$PID" ]; then
  echo "FATAL: no DeepDesert_1 process found."
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
echo "=== pre-storm baseline scan complete: $(date -u -Iseconds) ==="
ls -la "$OUT"
echo "$OUT" > "$BASE/last-prestorm-dir.txt"
