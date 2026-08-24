//go:build ignore
// +build ignore

// namepool.go -- decode a live FNamePool-style block.
//
// Format, empirically derived and verified against 58 consecutive real
// entries with zero decode failures on 2026-08-24:
//
//	[2-byte LE header][Length raw ASCII bytes][1 pad byte if Length is odd]
//
// header = (Length << 6) | ProbeHash6 -- the low 6 bits are a comparison
// hash used for fast interning lookups, not part of the string. Entries are
// packed back-to-back with 2-byte alignment (the single pad byte after an
// odd-length string exists purely to keep the next header aligned).
//
// Blocks are 64KB-aligned; a valid entry pointer masked with
// & 0xffffffffffff0000 gives the block's base address.
package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// addrToOffset converts a process virtual address to the signed offset
// io.ReaderAt.ReadAt requires, with an explicit bounds check rather than a
// bare cast. x86-64 Linux userspace addresses are always well under
// math.MaxInt64, so this never actually fails for a real address -- the
// check exists to make that guarantee explicit and verified, not assumed.
func addrToOffset(addr uint64) int64 {
	if addr > math.MaxInt64 {
		panic(fmt.Sprintf("address %#x exceeds int64 range", addr))
	}
	return int64(addr)
}

func decodeBlock(data []byte, start int) (entries []struct {
	Off  int
	Name string
}, failedAt int) {
	i := start
	for i < len(data)-4 {
		hdr := int(data[i]) | int(data[i+1])<<8
		length := hdr >> 6
		if length == 0 || length > 512 {
			return entries, i
		}
		sStart := i + 2
		if sStart+length > len(data) {
			return entries, i
		}
		raw := data[sStart : sStart+length]
		clean := true
		for _, b := range raw {
			if b < 32 || b >= 127 {
				clean = false
				break
			}
		}
		if !clean {
			return entries, i
		}
		entries = append(entries, struct {
			Off  int
			Name string
		}{i, string(raw)})
		next := sStart + length
		if length%2 == 1 {
			next++
		}
		i = next
	}
	return entries, i
}

func main() {
	pid := os.Args[1]
	blockAddr, err := strconv.ParseUint(os.Args[2], 0, 64)
	if err != nil {
		fmt.Println("bad block address:", err)
		os.Exit(1)
	}
	blockAddr &= 0xffffffffffff0000 // block-align

	memF, err := os.Open(fmt.Sprintf("/proc/%s/mem", pid))
	if err != nil {
		fmt.Println("open mem:", err)
		os.Exit(1)
	}
	defer memF.Close()

	const blockSize = 1 << 16
	buf := make([]byte, blockSize)
	if _, err := memF.ReadAt(buf, addrToOffset(blockAddr)); err != nil {
		fmt.Println("read block:", err)
		os.Exit(1)
	}

	// The block's first entry is very likely mid-string (block boundary does
	// not imply entry boundary), so try every small starting offset and keep
	// whichever gives the longest unbroken clean run -- that is almost
	// certainly the true entry alignment.
	bestStart, bestLen := 0, -1
	for s := 0; s < 64; s++ {
		entries, _ := decodeBlock(buf, s)
		if len(entries) > bestLen {
			bestLen = len(entries)
			bestStart = s
		}
	}
	entries, failedAt := decodeBlock(buf, bestStart)
	fmt.Printf("block %#x: best start offset %d, decoded %d entries cleanly, stopped at buffer offset %d\n",
		blockAddr, bestStart, len(entries), failedAt)
	for _, e := range entries {
		fmt.Printf("  [%#x] %q\n", blockAddr+uint64(e.Off), e.Name)
	}
}
