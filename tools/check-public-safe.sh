#!/usr/bin/env bash
# Fail if tracked files contain material that should not be public.
#
# This repository is public. Findings are written against a live game server, so
# it is easy to paste in an operator identifier or an internal address without
# noticing. Run this before publishing anything and in CI.
#
# Patterns are deliberately GENERIC -- writing the literal values here would put
# the very strings we are trying to remove back into a public file.
set -euo pipefail

cd "$(dirname "$0")/.."

fail=0
report() { # <label> <grep-args...>
  local label="$1"; shift
  local hits
  if hits=$(git grep -n -I -E "$@" -- . ':!tools/check-public-safe.sh' 2>/dev/null); then
    printf '\n[FAIL] %s\n%s\n' "$label" "$hits"
    fail=1
  fi
}

# Player identifiers: the game's FLS handles render as Name#12345.
report 'player identifier (Name#NNNN)' '[A-Za-z][A-Za-z0-9_]{2,}#[0-9]{4,}'

# RFC1918 addresses: internal topology.
report 'private IPv4 address' '(^|[^0-9.])(192\.168|10)\.[0-9]{1,3}\.[0-9]{1,3}([^0-9.]|$)'
report 'private IPv4 address (172.16/12)' '(^|[^0-9.])172\.(1[6-9]|2[0-9]|3[01])\.[0-9]{1,3}\.[0-9]{1,3}([^0-9.]|$)'

# Email addresses.
report 'email address' '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'

if [ "$fail" -ne 0 ]; then
  cat <<'MSG'

Replace the values above with placeholders such as <fls-id>, <character>,
<dev-vm-ip> or <scan-host>. Keep host ALIASES (they carry no address and the
operational docs need them); redact what identifies a person or a network.

Note: redaction limits future exposure only. Anything already pushed remains in
git history.
MSG
  exit 1
fi
echo "check-public-safe: clean"
