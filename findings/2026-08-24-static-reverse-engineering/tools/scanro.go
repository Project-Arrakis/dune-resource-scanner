//go:build ignore
// +build ignore

// scanro.go -- search the previously-unscanned read-only/no-permission
// anonymous memory regions for resource type name strings. If found,
// records each hit's exact runtime address so a follow-up pass can search
// for pointers referencing it.
package main

import (
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

	// The complement of HeapLikeRegions: anonymous, but NOT writable.
	var roAnon []memscan.Region
	for _, r := range regions {
		if r.Pathname != "" {
			continue
		}
		if len(r.Perms) < 2 || r.Perms[1] != 'w' {
			roAnon = append(roAnon, r)
		}
	}
	var total uint64
	for _, r := range roAnon {
		total += r.End - r.Start
	}
	fmt.Printf("scanning %d read-only/no-perm anonymous regions, %.2f GB\n", len(roAnon), float64(total)/1e9)

	found := 0
	buf := make([]byte, chunk+256)
	for _, r := range roAnon {
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
					i := indexFrom(b, nb, idx)
					if i < 0 {
						break
					}
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

func indexFrom(haystack, needle []byte, from int) int {
	if from >= len(haystack) {
		return -1
	}
	i := indexBytes(haystack[from:], needle)
	if i < 0 {
		return -1
	}
	return from + i
}

func indexBytes(h, n []byte) int {
	if len(n) == 0 || len(n) > len(h) {
		return -1
	}
	for i := 0; i+len(n) <= len(h); i++ {
		match := true
		for j := range n {
			if h[i+j] != n[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
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
