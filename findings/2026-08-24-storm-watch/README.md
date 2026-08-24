# Post-storm scan: hardened, tested, and actually scheduled

`CONTINUATION-PROMPT.md`'s "run a scan at 05:30" was written as a single fixed-time shot,
and was actually never scheduled -- the setup work was done but I got pulled into a live
investigation thread and dropped the last step. Caught when asked directly "what can we do
to ensure it runs successfully." Rebuilt properly rather than just scheduling the original
plan as-is.

## Why a single fixed-time shot was the wrong design

"The seed will change around 0400" is an estimate, not a guarantee. A one-shot job at 05:30
is fragile against the storm running early, running late, or -- least obviously -- landing
on the *same* seed by chance, in which case a single check would report a false negative
with no way to notice or retry.

## What runs instead

`storm-watch.sh`, fired by cron every 10 minutes across a window (04:00-07:59, bounded to
2026-08-25 specifically so it does not become a permanent daily job), rather than once at a
guessed time.

- **Idempotent.** A persisted state file (`~/scan-findings/last-scanned-seed.txt`,
  initialised to `2`, the baseline observed continuously since 2026-08-21) means most
  firings do one cheap DB query and exit. It only runs the full scan the *first* time it
  sees the seed actually change, and only marks that seed as handled *after* a successful
  scan -- a failed attempt leaves the state file alone, so the next firing 10 minutes later
  retries automatically rather than silently giving up.
- **No hardcoded PID**, anywhere -- `post-storm-scan.sh` resolves `DeepDesert_1` fresh each
  time, with retries (10 attempts, 30s apart) in case the map is still mid-respawn when the
  window opens.
- **Deployed to persistent storage**, not `/tmp` -- confirmed via `findmnt` earlier this
  session that `/tmp` on this host is `tmpfs`, and this job needed to survive ~17 hours
  unattended. Binary and scripts live under `~/scan-findings/bin/`.

## What actually went wrong building this, and how each was caught

Two real bugs, both caught by testing before trusting, not by inspection:

1. **A Python string-escaping mistake corrupted the deployed script.** Wrote the first
   version of `storm-watch.sh` by generating it through a Python string with bash's
   `'"'"'`-style quote-escaping embedded in it -- that syntax means nothing to Python, so the
   literal characters `'"'"'` were written straight into the file instead of a single quote.
   The script *looked* fine in the editor; it broke immediately on first real execution
   (`line 20: $2: unbound variable`). Rewritten via a plain heredoc, no nested escaping.
2. **An awk field-index off-by-one** in the original permission-bit check reused elsewhere
   this session (`^..w` checking position 3 instead of position 2) would have silently
   misclassified regions had it shipped uncaught -- caught during the memory-coverage
   investigation below, not this script, but the same mechanical error was worth naming
   here as a pattern: hand-written awk field/character-position math is exactly the kind of
   thing that looks obviously right and is worth a throwaway local test before trusting.

## Verification before trusting it for 17 hours

Deployed a temporary every-minute cron entry first and watched it fire five consecutive
times, correctly:

```
--- storm-watch fired: 2026-08-24T19:06:01+00:00 ---
current DeepDesert seed: 2
last-scanned seed: 2
no change -- skipping
[... 4 more identical, clean firings ...]
```

This also validated something that does not hold by default: `dune` resolved correctly and
`dune database sql` ran successfully under cron's minimal environment, which is a real,
common failure mode (cron's `PATH` is deliberately sparse compared to an interactive login
shell) that would otherwise only have been discovered at 04:00 with nobody there to fix it.

Only after this passed was the temporary entry replaced with the real windowed job.

## Files

- `storm-watch.sh` -- the cron-fired wrapper (idempotency, seed check).
- `post-storm-scan.sh` -- the actual scan (PID resolution with retries, markers, census).
