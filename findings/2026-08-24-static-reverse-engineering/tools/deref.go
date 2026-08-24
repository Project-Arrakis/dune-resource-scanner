//go:build ignore
// +build ignore

// deref.go -- dereference record offsets +280/+320/+336 for two known,
// different-typed, live-confirmed nodes and compare. If these are true
// per-class vtables, the two different types should show different
// addresses, and the DATA at those addresses should differ meaningfully;
// if they're a shared/generic vtable, the two nodes will match exactly.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"

	"dune-resource-scanner/internal/memscan"
)

const chunk = 64 << 20

func findRecordBase(memF *os.File, heap []memscan.Region, tx, ty float64) (uint64, bool) {
	buf := make([]byte, chunk+16)
	for _, r := range heap {
		size := r.End - r.Start
		for off := uint64(0); off < size; off += chunk {
			n := uint64(chunk) + 16
			if off+n > size {
				n = size - off
			}
			b := buf[:n]
			if _, err := memF.ReadAt(b, int64(r.Start+off)); err != nil {
				continue
			}
			base := r.Start + off
			limit := len(b) - 24
			for p := 0; p <= limit; p += 8 {
				x := math.Float64frombits(binary.LittleEndian.Uint64(b[p : p+8]))
				// Positive range test, not a negated one: math.Abs(NaN-tx) > 2 is FALSE
				// for NaN, so a negated guard lets NaN through for any target. This is
				// the exact #18 bug, reproduced here in a hand-rolled copy of the same
				// search -- see internal/memscan/scan.go's withinTolerance.
				dx := x - tx
				if !(dx >= -2 && dx <= 2) {
					continue
				}
				y := math.Float64frombits(binary.LittleEndian.Uint64(b[p+8 : p+16]))
				dy := y - ty
				if !(dy >= -2 && dy <= 2) {
					continue
				}
				if p < 256 {
					continue
				}
				rb := p - 256
				sig := binary.LittleEndian.Uint64(b[rb+8 : rb+16])
				if sig != 0x0000000100000001 {
					continue
				}
				fmt.Printf("  candidate: p=%d x=%.1f y=%.1f recordAddr=%#x\n", p, x, y, base+uint64(rb))
				return base + uint64(rb), true
			}
		}
	}
	return 0, false
}

func main() {
	pid := os.Args[1]
	label := os.Args[2]
	tx, err1 := strconv.ParseFloat(os.Args[3], 64)
	ty, err2 := strconv.ParseFloat(os.Args[4], 64)
	if err1 != nil || err2 != nil {
		fmt.Printf("FATAL: could not parse target coords %q,%q: %v %v\n", os.Args[3], os.Args[4], err1, err2)
		os.Exit(1)
	}
	fmt.Printf("target: label=%s x=%.1f y=%.1f\n", label, tx, ty)

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
	exe := memscan.MainModuleRegions(regions)
	mem := memscan.NewProcMem(memF)

	base, ok := findRecordBase(memF, heap, tx, ty)
	if !ok {
		fmt.Printf("%s: record not found near (%.0f,%.0f)\n", label, tx, ty)
		return
	}
	fmt.Printf("=== %s: record base %#x ===\n", label, base)

	// Also dump further into the +0 handle's target than the census's 256-byte
	// capture ever looked -- that object (a bounding box in the first 256
	// bytes) was never examined past that point.
	p0, err := mem.ReadU64(base)
	if err == nil {
		fmt.Printf("  +0 heap handle -> %#x, dumping 768 bytes\n", p0)
		var fb [768]byte
		if _, err := memF.ReadAt(fb[:], int64(p0)); err == nil {
			for i := 256; i < 768; i += 8 {
				v := binary.LittleEndian.Uint64(fb[i : i+8])
				fmt.Printf("    [+%3d] %#018x\n", i, v)
			}
		} else {
			fmt.Printf("    deref failed: %v\n", err)
		}
	}

	for _, off := range []uint64{280, 320, 336} {
		ptr, err := mem.ReadU64(base + off)
		if err != nil {
			fmt.Printf("  +%-4d read error: %v\n", off, err)
			continue
		}
		inExe := false
		for _, r := range exe {
			if r.Contains(ptr) {
				inExe = true
			}
		}
		fmt.Printf("  +%-4d -> %#x  (in main module: %v)\n", off, ptr, inExe)
		if ptr == 0 {
			continue
		}
		// Dereference: dump the first 64 bytes at the target address.
		var buf [64]byte
		if _, err := memF.ReadAt(buf[:], int64(ptr)); err != nil {
			fmt.Printf("       deref failed: %v\n", err)
			continue
		}
		for i := 0; i < 64; i += 8 {
			v := binary.LittleEndian.Uint64(buf[i : i+8])
			fmt.Printf("       [%+3d] %#018x\n", i, v)
		}
	}
}
