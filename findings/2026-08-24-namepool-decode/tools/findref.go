//go:build ignore
// +build ignore

// findref.go -- search all memory for 8-byte-aligned pointers referencing a
// specific target address (a string found by scanro.go), to see whether
// anything treats it as live FString/FName data rather than incidental bytes
// inside a larger allocation.
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"

	"dune-resource-scanner/internal/memscan"
)

const chunk = 64 << 20

func main() {
	pid := os.Args[1]
	target, err := strconv.ParseUint(os.Args[2], 0, 64)
	if err != nil {
		fmt.Println("bad target address:", err)
		os.Exit(1)
	}

	mapsF, _ := os.Open(fmt.Sprintf("/proc/%s/maps", pid))
	regions, _ := memscan.ParseMaps(mapsF)
	mapsF.Close()
	memF, err := os.Open(fmt.Sprintf("/proc/%s/mem", pid))
	if err != nil {
		fmt.Println("open mem:", err)
		os.Exit(1)
	}
	defer memF.Close()
	heap := memscan.HeapLikeRegions(regions)

	fmt.Printf("searching for 8-byte-aligned references to %#x across %d heap-like regions\n", target, len(heap))
	found := 0
	buf := make([]byte, chunk+8)
	for _, r := range heap {
		size := r.End - r.Start
		for off := uint64(0); off < size; off += chunk {
			n := uint64(chunk) + 8
			if off+n > size {
				n = size - off
			}
			b := buf[:n]
			if _, err := memF.ReadAt(b, int64(r.Start+off)); err != nil {
				continue
			}
			limit := len(b) - 8
			for p := 0; p <= limit; p += 8 {
				v := binary.LittleEndian.Uint64(b[p : p+8])
				if v == target {
					fmt.Printf("  REF at %#x\n", r.Start+off+uint64(p))
					found++
				}
			}
		}
	}
	fmt.Printf("total references: %d\n", found)
}
