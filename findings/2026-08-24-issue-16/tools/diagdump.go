//go:build ignore
// +build ignore

// Command diagdump is a throwaway investigation tool for issue #16. It locates
// a node's transform in memory and dumps the annotated qword layout around it,
// so the containing object's real shape (vtable position, pointer fields) can
// be read directly instead of inferred from assumed offsets.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"dune-resource-scanner/internal/memscan"
)

const chunk = 64 << 20

func main() {
	pid := flag.Int("pid", 0, "target pid")
	near := flag.String("near", "", "x,y")
	tol := flag.Float64("tolerance", 5, "per-axis half width")
	back := flag.Uint64("back", 1024, "bytes to dump before the transform")
	fwd := flag.Uint64("fwd", 128, "bytes to dump after the transform")
	maxHits := flag.Int("maxhits", 6, "dump at most this many hits")
	flag.Parse()

	nx, ny := parseXY(*near)
	mapsF, err := os.Open(fmt.Sprintf("/proc/%d/maps", *pid))
	must(err)
	regions, err := memscan.ParseMaps(mapsF)
	must(err)
	mapsF.Close()
	memF, err := os.Open(fmt.Sprintf("/proc/%d/mem", *pid))
	must(err)
	defer memF.Close()

	exe := memscan.MainModuleRegions(regions)
	heap := memscan.HeapLikeRegions(regions)
	all := regions
	mem := memscan.NewProcMem(memF)

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
	fmt.Printf("found %d transform-shaped hits near (%.0f,%.0f) tol=%.0f\n\n", len(hits), nx, ny, *tol)

	for i, h := range hits {
		if i >= *maxHits {
			fmt.Printf("... %d more hits not dumped\n", len(hits)-*maxHits)
			break
		}
		x, _ := mem.ReadF64(h)
		y, _ := mem.ReadF64(h + 8)
		z, _ := mem.ReadF64(h + 16)
		fmt.Printf("========== hit %d: addr=%#x pos=(%.3f, %.3f, %.3f) region=%s ==========\n",
			i, h, x, y, z, regionOf(all, h))
		start := h - *back
		for a := start; a <= h+*fwd; a += 8 {
			v, err := mem.ReadU64(a)
			if err != nil {
				continue
			}
			delta := int64(a) - int64(h)
			fmt.Printf("  %+6d  %#018x  %s\n", delta, v, annotate(v, exe, heap, all))
		}
		fmt.Println()
	}
}

func annotate(v uint64, exe, heap, all []memscan.Region) string {
	var tags []string
	if inSet(exe, v) {
		tags = append(tags, "EXE-PTR(vtable?)")
	} else if inSet(heap, v) {
		tags = append(tags, "HEAP-PTR")
	} else if r := regionOf(all, v); r != "" {
		tags = append(tags, "PTR->"+r)
	}
	f := math.Float64frombits(v)
	if !math.IsNaN(f) && !math.IsInf(f, 0) && math.Abs(f) > 1e-3 && math.Abs(f) < 1e7 {
		tags = append(tags, fmt.Sprintf("f64=%.3f", f))
	}
	lo := int32(uint32(v))
	hi := int32(uint32(v >> 32))
	if v < 1<<32 && v != 0 {
		tags = append(tags, fmt.Sprintf("u32=%d", uint32(v)))
	}
	f1 := math.Float32frombits(uint32(v))
	f2 := math.Float32frombits(uint32(v >> 32))
	if !math.IsNaN(float64(f1)) && !math.IsInf(float64(f1), 0) && math.Abs(float64(f1)) > 1e-3 && math.Abs(float64(f1)) < 1e7 &&
		!math.IsNaN(float64(f2)) && !math.IsInf(float64(f2), 0) && math.Abs(float64(f2)) < 1e7 {
		tags = append(tags, fmt.Sprintf("f32x2=(%.2f,%.2f)", f1, f2))
	}
	_ = lo
	_ = hi
	if len(tags) == 0 {
		return ""
	}
	return strings.Join(tags, " ")
}

func regionOf(rs []memscan.Region, a uint64) string {
	for _, r := range rs {
		if r.Contains(a) {
			if r.Pathname != "" {
				return r.Pathname + "(" + r.Perms + ")"
			}
			return "anon(" + r.Perms + ")"
		}
	}
	return ""
}

func inSet(rs []memscan.Region, a uint64) bool {
	for _, r := range rs {
		if r.Contains(a) {
			return true
		}
	}
	return false
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

var _ = binary.LittleEndian
