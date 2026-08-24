# Session records

Session summaries and continuation prompts for this tool live here, in the repository —
**never in `/tmp` or a scratchpad directory.** A handoff note that does not survive a reboot
is not a handoff note, and `/tmp` on the hosts this work runs against is `tmpfs`.

| File | What it is | Lifecycle |
|---|---|---|
| `CONTINUATION-PROMPT.md` | The current handoff. Paste into a fresh session to resume. | **Overwritten** each session — always describes the present state. |
| `YYYY-MM-DD-findings.md` | A point-in-time record of what one session established. | **Append-only.** Never edited after the fact; historical by design. |

Dated findings files are snapshots. They are accurate for the day they were written and are
deliberately *not* kept current — if a claim in one has since been overturned, the correction
lives in [`../CONTINUATION.md`](../CONTINUATION.md), which is the single living document for
this investigation. Read `CONTINUATION.md` first; read a dated file only when you need to know
what was believed at a particular point.

Machine-readable scan output and raw captures go in [`../findings/`](../findings/), not here.
