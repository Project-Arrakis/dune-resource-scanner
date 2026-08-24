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

## Session 2 (2026-08-24, later): +320 and +336 disassembled; still inconclusive, but further

Prompted directly again: after the DB/console schema was fully enumerated and closed off as
a type-attribution avenue (see `../2026-08-24-storm-watch/pre-storm-baseline/README.md`),
the operator asked whether memory/code/console had genuinely been exhausted. It had not --
only `+280` of the three EXE-relative pointers had ever actually been disassembled. This
picks up `+320` and `+336`.

**Finding the real binary required one extra step not needed before: the game server runs
inside a Docker container** (`docker-4b9d8...` per `/proc/<pid>/cgroup`), so the path
recorded in `/proc/<pid>/maps` (`/home/dune/server/DuneSandbox/Binaries/Linux/...`) does not
exist on the host's own filesystem -- it's the path as the process sees it inside its own
mount namespace. Reached it via `/proc/<pid>/root/<that path>`, which resolves through the
process's namespace regardless of which namespace the caller is in. Confirmed correct: file
size (374,143,544 bytes) matches this document's earlier "374 MB binary" note.

### Binary-strings search: negative, and it rules out static XREF entirely

Before disassembling, checked the obvious shortcut: do the resource class names
(`TitaniumOre`, `BP_StravidiumOre_...`, etc.) exist as literal strings anywhere in the
binary file itself, separate from the runtime-only FNamePool heap block? Searched with
`strings -a -n 6` across the whole 374 MB file (966,449 strings extracted) for every mineral
name and for generic patterns (`_Ore_`, `_Pickup_`, `_Spawner_C`). **Zero matches, of any
kind.** These names are not baked into the executable's static data at all -- they must come
from a separate asset/pak file loaded at runtime, which is also why the FNamePool block
housing them was only ever found in heap memory, never in the file. This closes the
static-string-XREF approach for good; there is nothing in the binary to cross-reference
against.

### +320: confirmed to be a genuine, well-formed C++ vtable

Dereferencing `+320` against the live, confirmed `StravidiumOre` record gives an array of
7 pointers, 6 of them stepping by exactly 16 bytes (`...b990, b9a0, b9b0, b9c0, b9d0, b9e0`).
Disassembled two of the slots. Both show the same recognizable shape: a scalar-deleting
destructor (`mov esi,0x18; call <alloc-free-helper>` -- freeing an 0x18/24-byte object) and
a trivial `xor eax,eax; ret` no-op getter, each padded to a 16-byte boundary with `int3`
filler between them -- the standard MSVC/Itanium compiler pattern for a run of default/mostly
-unimplemented virtual functions. This **confirms** (not just "structurally resembles") a
genuine vtable at `+320`. It does not help with type attribution though: a vtable this
generic, with real default implementations doing nothing type-specific, is exactly what
you'd expect from a shared/common base-class interface -- consistent with, not contradicting,
the standing "no class identity in these records" theory.

### +336: a real function, not a stub -- and it goes one step further than +280 did

Unlike `+320`'s trivial destructor thunks, `+336` targets a substantial function with a real
stack frame (`sub rsp,0x158`). Traced it:

1. Calls the **same PLT-resolved libc function** `+280` was already traced to (same GOT
   target, `...2a690`) -- independent corroboration that this whole area funnels through one
   shared, generic runtime utility, not type-specific code.
2. Takes that call's return value (`eax`) and uses it directly as an **array index**:
   `rcx = eax; rcx *= 0x70` (112-byte stride).
3. Loads a RIP-relative global (computed precisely: instruction vaddr + length + displacement
   = `0x56724e615e80`), which live process inspection shows is **not in the file-backed
   binary at all** -- it lands in the anonymous `rw-p` region immediately following the
   binary's own `.data` segment (`56724e528000-56724e98a000 00:00 0`), i.e. this is BSS,
   allocated and populated only at runtime. Read live: it holds a pointer to a real,
   currently-mapped heap array at `0x75f8b6ee0000`.
4. Reads a single **byte at `entry+0x10`** and a **pointer at `entry+0x50`** from the indexed
   112-byte struct -- the shape of an ID-to-metadata lookup (a category/flag byte, plus a
   pointer to something else, possibly a name).

This is a real step further than the `+280` trace ever got -- that one stopped at "it's a
libc call, proves nothing." This one shows the libc call's *return value* feeding a genuine
lookup table.

**Where it stopped, and why that's a real boundary, not an early quit:** dumped the first 40
entries (indices 0-39) of the live table at `0x75f8b6ee0000`. Every single one is identical
and unpopulated -- `byte+0x10 == 0x04`, `ptr+0x50 == 0` (null) -- for all 40. That's
consistent with either an under-populated/default region of a larger pool, or with sampling
the wrong index range entirely. **The actual index used for any specific record's lookup is
the return value of a function call that was never observed executing** -- getting it
requires single-stepping or breakpointing the live process at the call site, which requires a
debugger. `gdb` is not installed on this host (checked directly: `radare2`, `r2`, `rizin`,
`gdb`, `ghidra` all absent; only `objdump`/`readelf`/`nm`/`strings`). This is the same
boundary the Session 1 conclusion already predicted -- "proper decompilation... tracing
considerably more of the call graph... is a multi-hour-to-multi-day undertaking with proper
tooling" -- reached concretely rather than assumed. Installing `gdb` (available via `apt`,
not yet done -- a real host-state change worth flagging rather than doing silently) would be
the next unlock, with no guarantee the call site fires on a predictable trigger even with a
breakpoint set.

## Files

- `tools/deref.go` -- the dereference/disassembly-target tool (`//go:build ignore`),
  including the fix for the exact #18-pattern NaN bug it was found to have reproduced.
- `tools/dump_table.py` -- reads the live `+336` lookup table given its runtime base pointer;
  session-2-specific, not general-purpose (the base pointer is hardcoded from one live run).

Raw disassembly output and the region-breakdown numbers are recorded inline above rather
than as separate captured files -- both are small and fully reproducible from the commands
in this document against the same live process.
