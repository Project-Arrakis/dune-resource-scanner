//go:build ignore
// +build ignore

// scanro.go -- search the previously-unscanned read-only/no-permission
// anonymous memory regions for resource type name strings. If found,
// records each hit's exact runtime address so a follow-up pass can search
// for pointers referencing it.
package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"dune-resource-scanner/internal/memscan"
)

const chunk = 64 << 20

func main() {
	pid := os.Args[1]
	needles := os.Args[2:]

	mapsF, _ := os.Open(fmt.Sprintf("/proc/%s/maps", pid))
	regions, _ := memscan.ParseMaps(mapsF)
	mapsF.Close()
	memF, err := os.Open(fmt.Sprintf("/proc/%s/mem", pid))
	if err != nil {
		fmt.Println("open mem:", err)
		os.Exit(1)
	}
	defer memF.Close()

	// Both heap-like (writable) and its complement (anonymous, not writable) --
	// text has never been searched for in either; all prior scanning was
	// numeric pattern matching only.
	regionSet := memscan.HeapLikeRegions(regions)
	var roAnon []memscan.Region
	for _, r := range regions {
		if r.Pathname != "" {
			continue
		}
		if len(r.Perms) < 2 || r.Perms[1] != 'w' {
			roAnon = append(roAnon, r)
		}
	}
	regionSet = append(regionSet, roAnon...)
	var total uint64
	for _, r := range regionSet {
		total += r.End - r.Start
	}
	fmt.Printf("scanning %d regions (heap-like + read-only anon), %.2f GB\n", len(regionSet), float64(total)/1e9)

	found := 0
	buf := make([]byte, chunk+256)
	for _, r := range regionSet {
		size := r.End - r.Start
		for off := uint64(0); off < size; off += chunk {
			n := uint64(chunk) + 256
			if off+n > size {
				n = size - off
			}
			b := buf[:n]
			if _, err := memF.ReadAt(b, int64(r.Start+off)); err != nil {
				continue
			}
			for _, needle := range needles {
				nb := []byte(needle)
				idx := 0
				for {
					rel := bytes.Index(b[idx:], nb)
					if rel < 0 {
						break
					}
					i := idx + rel
					addr := r.Start + off + uint64(i)
					ctx := string(b[max0(i-8) : min(len(b), i+len(nb)+24)])
					ctx = strings.Map(func(r rune) rune {
						if r < 32 || r > 126 {
							return '.'
						}
						return r
					}, ctx)
					fmt.Printf("  HIT %-20s addr=%#x  region=%#x-%#x  ctx=%q\n", needle, addr, r.Start, r.End, ctx)
					found++
					idx = i + 1
				}
			}
		}
	}
	fmt.Printf("total hits: %d\n", found)
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
