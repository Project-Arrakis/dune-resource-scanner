#!/usr/bin/env python3
"""Generate docs/index.html from findings/README.md.

findings/README.md is the single source of truth. This renders it as a
standalone, styled page for GitHub Pages so the two cannot drift: CI regenerates
and fails if the committed HTML does not match.

  tools/build-findings-page.py           # write docs/index.html
  tools/build-findings-page.py --check   # exit 1 if it is out of date

Requires markdown-it-py.
"""
from __future__ import annotations

import argparse
import html
import pathlib
import re
import sys

try:
    from markdown_it import MarkdownIt
except ImportError:
    sys.exit("markdown-it-py is required: pip install markdown-it-py")

ROOT = pathlib.Path(__file__).resolve().parent.parent
SOURCE = ROOT / "findings" / "README.md"
TARGET = ROOT / "docs" / "index.html"
BLOB = "https://github.com/Project-Arrakis/dune-resource-scanner/blob/main/findings/"

TITLE = "Deep Desert Scanner Findings"
DESCRIPTION = ("What the Dune: Awakening resource-scanner project has established, "
               "disproved, and left open, with measured evidence for each claim.")

# Status markers carry real state, so they are rendered as chips rather than left
# as bare emoji. Keys are the markers used in findings/README.md.
CHIPS = {
    "✅": ("c-ok", "established"),
    "❌": ("c-no", "disproved"),
    "⚠️": ("c-warn", "bounded"),
    "❓": ("c-open", "open"),
}

STYLE = """
:root{
  --ground:#FBFAF7; --raise:#FFFFFF; --ink:#1A1714; --body:#3A332C;
  --muted:#6E655C; --rule:#E3DDD4; --rule-soft:#EFEAE2;
  --accent:#B4541E; --accent-soft:#F3E4D8;
  --ok:#2F6B4F; --ok-bg:#E4EFE8; --no:#A32E2E; --no-bg:#F6E3E1;
  --warn:#8A6612; --warn-bg:#F4EBD6; --open:#44557A; --open-bg:#E3E7F0;
  --shadow:0 1px 2px rgba(26,23,20,.05),0 8px 24px -16px rgba(26,23,20,.25);
}
@media (prefers-color-scheme:dark){
  :root:not([data-theme="light"]){
    --ground:#14120F; --raise:#1D1A16; --ink:#F1ECE4; --body:#D3CBC0;
    --muted:#9A9086; --rule:#332E27; --rule-soft:#262219;
    --accent:#E2884B; --accent-soft:#3A2717;
    --ok:#7FC49E; --ok-bg:#1B2C23; --no:#E58A82; --no-bg:#331E1C;
    --warn:#D9B45E; --warn-bg:#2E2716; --open:#9FB1D6; --open-bg:#1D2331;
    --shadow:0 1px 2px rgba(0,0,0,.4),0 8px 24px -16px rgba(0,0,0,.7);
  }
}
:root[data-theme="dark"]{
  --ground:#14120F; --raise:#1D1A16; --ink:#F1ECE4; --body:#D3CBC0;
  --muted:#9A9086; --rule:#332E27; --rule-soft:#262219;
  --accent:#E2884B; --accent-soft:#3A2717;
  --ok:#7FC49E; --ok-bg:#1B2C23; --no:#E58A82; --no-bg:#331E1C;
  --warn:#D9B45E; --warn-bg:#2E2716; --open:#9FB1D6; --open-bg:#1D2331;
  --shadow:0 1px 2px rgba(0,0,0,.4),0 8px 24px -16px rgba(0,0,0,.7);
}
*{box-sizing:border-box}
body{margin:0;background:var(--ground);color:var(--body);
  font-family:"IBM Plex Sans",system-ui,-apple-system,"Segoe UI",sans-serif;
  font-size:16px;line-height:1.65;-webkit-font-smoothing:antialiased}
.wrap{max-width:58rem;margin:0 auto;padding:clamp(2rem,5vw,4.5rem) clamp(1.1rem,4vw,2.5rem) 5rem}
.masthead{padding-bottom:1.5rem;border-bottom:2px solid var(--ink);margin-bottom:2.6rem}
.eyebrow{font-family:"IBM Plex Mono",ui-monospace,monospace;font-size:.7rem;letter-spacing:.14em;
  text-transform:uppercase;color:var(--accent);margin:0 0 .7rem}
h1{font-family:"IBM Plex Serif",Georgia,serif;font-weight:600;color:var(--ink);
  font-size:clamp(2rem,5.5vw,3rem);line-height:1.1;margin:0;letter-spacing:-.015em;text-wrap:balance}
h2{font-family:"IBM Plex Serif",Georgia,serif;font-weight:600;color:var(--ink);font-size:1.55rem;
  line-height:1.2;margin:3rem 0 .9rem;letter-spacing:-.01em;text-wrap:balance;
  padding-top:1.4rem;border-top:1px solid var(--rule)}
h3{font-family:"IBM Plex Serif",Georgia,serif;font-weight:600;color:var(--ink);font-size:1.2rem;
  line-height:1.25;margin:2.2rem 0 .7rem;text-wrap:balance}
h4{font-family:"IBM Plex Sans",sans-serif;font-weight:600;color:var(--ink);font-size:.95rem;
  margin:1.6rem 0 .5rem;letter-spacing:.02em}
p,li{max-width:68ch}
p{margin:0 0 1rem}
ul,ol{margin:0 0 1.1rem;padding-left:1.3rem}
li{margin:.3rem 0}
li::marker{color:var(--muted)}
hr{border:none;border-top:1px solid var(--rule);margin:2.4rem 0}
blockquote{margin:0 0 1.1rem;padding:.85rem 1.05rem;background:var(--accent-soft);
  border-left:3px solid var(--accent);border-radius:0 2px 2px 0}
blockquote p:last-child{margin:0}
strong{color:var(--ink);font-weight:600}
a{color:var(--accent);text-underline-offset:2px}
a:hover{text-decoration-thickness:2px}
a:focus-visible{outline:2px solid var(--accent);outline-offset:2px;border-radius:1px}
code{font-family:"IBM Plex Mono",ui-monospace,monospace;font-size:.86em;background:var(--rule-soft);
  padding:.1rem .3rem;border-radius:2px;color:var(--ink)}
pre{background:var(--raise);border:1px solid var(--rule);border-radius:2px;padding:.9rem 1rem;
  overflow-x:auto;margin:0 0 1.1rem}
pre code{background:none;padding:0;font-size:.83rem;line-height:1.6}
.scroller{overflow-x:auto;border:1px solid var(--rule);border-radius:2px;background:var(--raise);
  margin:0 0 1.3rem;box-shadow:var(--shadow)}
table{border-collapse:collapse;width:100%;font-size:.89rem}
th,td{padding:.55rem .9rem;text-align:left;border-bottom:1px solid var(--rule-soft);vertical-align:top}
th{color:var(--ink);font-weight:600;font-size:.75rem;letter-spacing:.05em;text-transform:uppercase;
  white-space:nowrap}
td{font-variant-numeric:tabular-nums}
tbody tr:last-child td{border-bottom:none}
.chip{display:inline-block;font-family:"IBM Plex Mono",ui-monospace,monospace;font-size:.66rem;
  letter-spacing:.08em;text-transform:uppercase;padding:.16rem .42rem;border-radius:2px;
  white-space:nowrap;font-weight:500}
.c-ok{background:var(--ok-bg);color:var(--ok)}
.c-no{background:var(--no-bg);color:var(--no)}
.c-warn{background:var(--warn-bg);color:var(--warn)}
.c-open{background:var(--open-bg);color:var(--open)}
footer{margin-top:3.5rem;padding-top:1.3rem;border-top:1px solid var(--rule);
  color:var(--muted);font-size:.85rem}
@media (prefers-reduced-motion:reduce){*{animation:none!important;transition:none!important}}
"""


def render_markdown(text: str) -> str:
    md = MarkdownIt("commonmark").enable("table").enable("strikethrough")
    return md.render(text)


def absolutise_links(body: str) -> str:
    """Repo-relative links do not resolve on a standalone page; point them at GitHub."""
    def fix(m: re.Match) -> str:
        href = m.group(1)
        if re.match(r"^(https?:|mailto:|#)", href):
            return m.group(0)
        return f'href="{BLOB}{href.lstrip("./")}"'
    return re.sub(r'href="([^"]+)"', fix, body)


def chipify(body: str) -> str:
    """Status markers encode real state -- render them as chips, not bare emoji."""
    for marker, (cls, label) in CHIPS.items():
        body = body.replace(marker, f'<span class="chip {cls}">{label}</span>')
    return body


def wrap_tables(body: str) -> str:
    """Tables must scroll inside their own container, never the page body."""
    return body.replace("<table>", '<div class="scroller"><table>').replace(
        "</table>", "</table></div>")


def build() -> str:
    text = SOURCE.read_text()
    # The H1 becomes the masthead, so drop it from the flowed body.
    text = re.sub(r"^# .*?\n", "", text, count=1)
    body = wrap_tables(chipify(absolutise_links(render_markdown(text))))
    return f"""<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="description" content="{html.escape(DESCRIPTION, quote=True)}">
<meta name="color-scheme" content="light dark">
<meta property="og:title" content="{html.escape(TITLE, quote=True)}">
<meta property="og:description" content="{html.escape(DESCRIPTION, quote=True)}">
<meta property="og:type" content="article">
<title>{html.escape(TITLE)}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&family=IBM+Plex+Sans:wght@400;500;600&family=IBM+Plex+Serif:wght@400;500;600&display=swap">
<style>{STYLE}</style>
</head>
<body>
<div class="wrap">
<header class="masthead">
<p class="eyebrow">Findings index &middot; generated from source</p>
<h1>{html.escape(TITLE)}</h1>
</header>
{body}<footer>
<p>Generated from <a href="{BLOB}README.md">findings/README.md</a> by
<code>tools/build-findings-page.py</code>. Edit the Markdown, never this page.
Operational detail &mdash; hosts, addresses and player identifiers &mdash; is
deliberately omitted.</p>
</footer>
</div>
</body>
</html>
"""


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--check", action="store_true",
                    help="exit 1 if docs/index.html is out of date")
    args = ap.parse_args()
    generated = build()
    if args.check:
        current = TARGET.read_text() if TARGET.exists() else ""
        if current != generated:
            print("docs/index.html is out of date -- run tools/build-findings-page.py",
                  file=sys.stderr)
            return 1
        print("findings page: up to date")
        return 0
    TARGET.parent.mkdir(exist_ok=True)
    TARGET.write_text(generated)
    print(f"wrote {TARGET.relative_to(ROOT)} ({len(generated):,} bytes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
