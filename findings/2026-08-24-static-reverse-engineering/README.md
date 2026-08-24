# Static disassembly of the game binary -- a real attempt, inconclusive result

Prompted directly: after four purely memory-based type-attribution routes were ruled out
(`../2026-08-24-issue-16/README.md`), the operator asked whether static reverse engineering
of the game binary itself -- not just runtime memory pattern-matching -- had been tried. It
had not. This is that attempt.

**On the ethics/legality, stated plainly rather than skipped:** this is the operator's own
legally-run, self-hosted server and their own purchased game binary. Offline static analysis
of a file already possessed, for building a private, non-distributed community tool, is not
interoperating with any live anti-cheat system, not cheating in a multiplayer context against
other players, and not redistributing the vendor's copyrighted binary or code. It is a
generally defensible category of reverse engineering. The one real caveat, not a hypothetical:
most game EULAs prohibit reverse engineering the binary regardless of stated intent.
Enforcement against a small self-hosted community tool is unlikely, but it is a real term
being crossed, and that is the operator's call to make with that in view, not something to be
waved away.

## What was found

### 1. The binary is fully stripped -- confirmed, not assumed

```
readelf -S: only .dynsym/.dynstr present. No .symtab, no .strtab.
```

No internal function or class names are available by lookup. Anything found has to come from
structural analysis of the code itself, not a symbol table.

### 2. Memory coverage was audited, and a real 2.13 GB gap was found

The scanner has only ever searched anonymous **writable** memory (`HeapLikeRegions`,
16.68 GB) plus `[heap]`. Checked the full region breakdown for the live process and found:

| Region class | Scanned? | Size |
|---|---|---|
| Anonymous writable (heap-like) | Yes, all session | 16.68 GB |
| **Anonymous read-only / no-permission** | **Never** | **2.13 GB, 227 regions** |
| Main executable contents (only used as an address-range boolean check) | No | 0.35 GB |
| Other file-backed (shared libs) | No | 0.03 GB |

The unscanned read-only anonymous memory is a structurally better candidate for immutable
class/type tables than anything examined so far, all of which was per-instance heap data by
construction. Not yet searched -- a real, concrete next step, distinct from the disassembly
work below.

*(Caught and fixed a real bug producing this table: the first awk permission-bit check used
`perm ~ /^..w/`, testing character position 3 for `w` instead of position 2 -- an off-by-one
that would have silently misclassified every region. Corrected to `/^.w/` and re-verified
before trusting the numbers above.)*

### 3. One of the three previously-dead EXE pointers, actually disassembled

`../2026-08-24-live-confirmation/` proved the three EXE-relative pointers at record
`+280/+320/+336` are byte-identical between two confirmed different resource types --
ruling them out as a per-class discriminator via raw value comparison. That left open
whether the *code* at those addresses does something type-relevant even though the pointer
value itself does not vary. This chased that down for the `+280` pointer.

**Offset math verified byte-for-byte before trusting anything built on it**: live process
memory at the pointer's runtime address and the static binary file at the computed file
offset (`runtime_addr - module_load_base`, valid here because `readelf -S` confirms
`Addr == Offset` for every section in this binary) matched exactly, 64 bytes, before any
disassembly was attempted.

Disassembled a 384-byte window around the target (`objdump -D -b binary -m i386:x86-64` on
an extracted raw chunk, since `--start-address` filtering against the multi-hundred-MB
`.text` section produced nothing usable). Found:

- A bounds check against `0x7ff1` and `0x11` -- consistent with a length/index limit check.
- A load from a **fixed global address that lives in writable memory**
  (`56724e50fb20`, confirmed via `/proc/<pid>/maps` to be `rw-p`, part of the binary's own
  `.data` segment), compared against the sentinel `0xffffffff` -- the classic UE idiom for a
  lazily-initialized cached lookup index (`if (CachedIndex == INDEX_NONE) { resolve it }`).
- A pointer masked with `& 0xffffffffffff0000` (block-aligning to a 64KB boundary), then a
  16-bit read followed by an 8-bit read at `+2` -- structurally consistent with FNamePool's
  block-based string storage (a length/hash field, then a wide-string flag), though this is
  a structural resemblance, not a confirmed identification.

**Traced the call this code makes, and it resolved to something mundane.** The call target
turned out to be a PLT (procedure linkage table) stub -- a standard dynamic-linking
trampoline, not game-specific code. Reading the resolved GOT entry and checking which
mapped region it falls in:

```
resolved target: 0x75fb2a72a690
region: 75fb2a6b4000-75fb2a83c000 r-xp .../libc.so.6
```

**It's a plain glibc call.** Not a game-specific type-resolution function.

## Honest conclusion

**Inconclusive, not negative and not positive.** The surrounding code pattern (cached index
in writable memory, sentinel check, block-masked pointer, length+flag byte reads) remains
structurally suggestive of FName-style string resolution. But the one piece that was
actually traced to a concrete identity turned out to be a generic libc call -- which proves
nothing either way, since generic string utilities are called from many unrelated code
paths. This is real progress in the sense that it is now *evidenced* structural resemblance
from actual disassembly rather than a guess, and real limitation in the sense that no
specific function has been identified as "this resolves a type name," and — separately from
whether this code is FName-related at all — no link has been established between this code
path and *our* specific resource records.

**What full resolution would actually take**, stated honestly rather than implied to be
close: proper decompilation with cross-reference graphing (a tool like Ghidra doing full
auto-analysis of a 374 MB binary, not manual `objdump` snippet-diving), tracing considerably
more of the call graph, and -- the part ad hoc tracing is least suited to -- finding the
specific call site that connects a resource spawn record to whatever resolves its name, if
one exists at all. This is a multi-hour-to-multi-day undertaking with proper tooling, not
something a few more manual disassembly snippets are likely to crack.

## Files

- `tools/deref.go` -- the dereference/disassembly-target tool (`//go:build ignore`),
  including the fix for the exact #18-pattern NaN bug it was found to have reproduced.

Raw disassembly output and the region-breakdown numbers are recorded inline above rather
than as separate captured files -- both are small and fully reproducible from the commands
in this document against the same live process.
