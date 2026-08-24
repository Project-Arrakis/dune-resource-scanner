//go:build ignore
// +build ignore

// Command diagarray is a throwaway investigation tool for issue #16. It tests
// the hypothesis that resource-node positions live in a large array of
// fixed-stride records rather than in individually pointer-referenced actors,
// by walking outward from one known hit and checking the extracted positions
// against the known marker set.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"dune-resource-scanner/internal/memscan"
)

const chunk = 64 << 20

type marker struct {
	T    string
	X, Y float64
}

func main() {
	pid := flag.Int("pid", 0, "target pid")
	near := flag.String("near", "", "x,y of a known node, to seed the array walk")
	tol := flag.Float64("tolerance", 5, "seed match tolerance")
	stride := flag.Uint64("stride", 384, "record stride in bytes")
	recOff := flag.Uint64("recoff", 256, "offset of the position triple within a record")
	span := flag.Int("span", 20000, "records to walk in each direction")
	mkFile := flag.String("markers", "", "csv: type,x,y")
	flag.Parse()

	nx, ny := parseXY(*near)
	markers := loadMarkers(*mkFile)

	mapsF, err := os.Open(fmt.Sprintf("/proc/%d/maps", *pid))
	must(err)
	regions, err := memscan.ParseMaps(mapsF)
	must(err)
	mapsF.Close()
	memF, err := os.Open(fmt.Sprintf("/proc/%d/mem", *pid))
	must(err)
	defer memF.Close()
	heap := memscan.HeapLikeRegions(regions)
	mem := memscan.NewProcMem(memF)

	// Seed: find transform-shaped hits near the known node.
	var hits []uint64
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
			hits = append(hits, memscan.FindNearbyXY(b, r.Start+off, nx, ny, *tol)...)
		}
	}
	fmt.Printf("seed hits: %d\n", len(hits))

	type found struct {
		x, y, z float64
		addr    uint64
	}
	best := map[string][]found{}
	for _, h := range hits {
		z, err := mem.ReadF64(h + 16)
		if err != nil || math.Abs(z) > 100000 {
			continue // keep only the plausible-Z copy, not the pre-trace sentinel
		}
		if h < *recOff {
			continue
		}
		base := h - *recOff
		region := regionContaining(heap, base)
		if region == nil {
			continue
		}
		var recs []found
		for k := -*span; k <= *span; k++ {
			addr := uint64(int64(base) + int64(k)*int64(*stride))
			if addr < region.Start || addr+*recOff+24 >= region.End {
				continue
			}
			x, e1 := mem.ReadF64(addr + *recOff)
			y, e2 := mem.ReadF64(addr + *recOff + 8)
			z, e3 := mem.ReadF64(addr + *recOff + 16)
			if e1 != nil || e2 != nil || e3 != nil {
				continue
			}
			if !memscan.ValidTransform(x, y, z) || math.Abs(z) > 100000 {
				continue
			}
			recs = append(recs, found{x, y, z, addr})
		}
		key := fmt.Sprintf("%#x", base)
		best[key] = recs
		fmt.Printf("  seed %#x (region %#x-%#x, %.1f MB): %d plausible records extracted\n",
			h, region.Start, region.End, float64(region.End-region.Start)/1e6, len(recs))
	}

	// Score each candidate array against the marker set.
	var keys []string
	for k := range best {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		recs := best[k]
		if len(recs) == 0 {
			continue
		}
		matched := 0
		byType := map[string]int{}
		nType := map[string]int{}
		for _, m := range markers {
			nType[m.T]++
			bestd := math.Inf(1)
			for _, r := range recs {
				d := math.Hypot(r.x-m.X, r.y-m.Y)
				if d < bestd {
					bestd = d
				}
			}
			if bestd <= 100 {
				matched++
				byType[m.T]++
			}
		}
		fmt.Printf("\n=== array at %s: %d records -> %d/%d markers matched (%.1f%%) ===\n",
			k, len(recs), matched, len(markers), 100*float64(matched)/float64(len(markers)))
		var ts []string
		for t := range nType {
			ts = append(ts, t)
		}
		sort.Strings(ts)
		for _, t := range ts {
			fmt.Printf("   %-22s %3d/%3d\n", t, byType[t], nType[t])
		}
	}
}

func regionContaining(rs []memscan.Region, a uint64) *memscan.Region {
	for i := range rs {
		if rs[i].Contains(a) {
			return &rs[i]
		}
	}
	return nil
}

func loadMarkers(p string) []marker {
	f, err := os.Open(p)
	must(err)
	defer f.Close()
	var out []marker
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Split(strings.TrimSpace(sc.Text()), ",")
		if len(parts) < 3 {
			continue
		}
		x, e1 := strconv.ParseFloat(parts[1], 64)
		y, e2 := strconv.ParseFloat(parts[2], 64)
		if e1 != nil || e2 != nil {
			continue
		}
		out = append(out, marker{parts[0], x, y})
	}
	return out
}

func parseXY(s string) (float64, float64) {
	p := strings.SplitN(s, ",", 2)
	x, _ := strconv.ParseFloat(strings.TrimSpace(p[0]), 64)
	y, _ := strconv.ParseFloat(strings.TrimSpace(p[1]), 64)
	return x, y
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}
