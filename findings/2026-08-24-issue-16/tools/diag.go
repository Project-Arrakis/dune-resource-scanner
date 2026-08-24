//go:build ignore
// +build ignore

// Command diag is a throwaway investigation tool for issue #16. It measures
// where resource-node actors are lost between the raw (X,Y) transform match
// and a validated actor, marker by marker, calling the real memscan library
// so the numbers reflect the shipped code path. Not part of the shipped tool.
package main

import (
	"bufio"
	"encoding/binary"
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

type xyHit struct {
	Addr uint64
	X, Y float64
}

func main() {
	pid := flag.Int("pid", 0, "target pid")
	near := flag.String("near", "", "x,y")
	tol := flag.Float64("tolerance", 15000, "per-axis half width")
	mkFile := flag.String("markers", "", "csv: type,x,y")
	offsetProbe := flag.Bool("offset-probe", false, "also resolve transform/rootComponent offsets offset-agnostically")
	maxDelta := flag.Uint64("maxdelta", 4096, "offset-probe: how far back from a transform hit to look for an object base")
	maxK := flag.Uint64("maxk", 4096, "offset-probe: how far back from a reference to look for an actor base")
	maxProbeHits := flag.Int("maxprobehits", 4000, "offset-probe: refuse to run if pass 1 exceeds this many hits")
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

	exe := memscan.MainModuleRegions(regions)
	heap := memscan.HeapLikeRegions(regions)
	isExe := func(a uint64) bool { return inSet(exe, a) }
	isHeap := func(a uint64) bool { return inSet(heap, a) }
	mem := memscan.NewProcMem(memF)

	var heapBytes uint64
	for _, r := range heap {
		heapBytes += r.End - r.Start
	}
	fmt.Printf("regions: exe=%d heap-like=%d (%.1f GB)\n", len(exe), len(heap), float64(heapBytes)/1e9)

	// ---- PASS 1 ----
	var hits []xyHit
	var readErrs int
	var readErrBytes uint64
	streamRegions(memF, heap, &readErrs, &readErrBytes, func(base uint64, buf []byte) {
		for _, a := range memscan.FindNearbyXY(buf, base, nx, ny, *tol) {
			off := a - base
			x := math.Float64frombits(binary.LittleEndian.Uint64(buf[off : off+8]))
			y := math.Float64frombits(binary.LittleEndian.Uint64(buf[off+8 : off+16]))
			hits = append(hits, xyHit{a, x, y})
		}
	})
	fmt.Printf("pass1: read-error regions=%d (%.2f GB skipped)\n", readErrs, float64(readErrBytes)/1e9)
	fmt.Printf("pass1: raw XY hits=%d distinct positions=%d\n", len(hits), countDistinct(hits))
	reportCoverage("PASS-1 (raw XY transform present in scanned memory)", markers, hits)

	// ---- PASS 2 ----
	rcs := map[uint64]bool{}
	rcOfHit := map[uint64]xyHit{}
	for _, h := range hits {
		if h.Addr < memscan.DefaultOffsets.Transform {
			continue
		}
		rc := h.Addr - memscan.DefaultOffsets.Transform
		rcs[rc] = true
		rcOfHit[rc] = h
	}
	refsPerRC := map[uint64]int{}
	stages := map[string]int{}
	var validated []xyHit
	validRC := map[uint64]bool{}
	streamRegions(memF, heap, new(int), new(uint64), func(base uint64, buf []byte) {
		for _, ref := range memscan.FindPointerReferencesMulti(buf, base, rcs) {
			refsPerRC[ref.Target]++
			if ref.Addr < memscan.DefaultOffsets.RootComponent {
				continue
			}
			actor := ref.Addr - memscan.DefaultOffsets.RootComponent
			st := stagedValidate(mem, actor, isExe, isHeap)
			stages[st]++
			if st == "ok" && !validRC[ref.Target] {
				validRC[ref.Target] = true
				validated = append(validated, rcOfHit[ref.Target])
			}
		}
	})
	withRef := 0
	for rc := range rcs {
		if refsPerRC[rc] > 0 {
			withRef++
		}
	}
	fmt.Printf("\npass2: rootComponent candidates=%d  with>=1 pointer ref=%d (%.1f%%)  validated=%d\n",
		len(rcs), withRef, 100*float64(withRef)/float64(maxi(1, len(rcs))), len(validRC))
	fmt.Printf("pass2: ValidateActor stage outcomes: ")
	var keys []string
	for k := range stages {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s=%d ", k, stages[k])
	}
	fmt.Println()
	reportCoverage("PASS-2 (fully validated actor)", markers, validated)

	if !*offsetProbe {
		return
	}
	if len(hits) > *maxProbeHits {
		fmt.Printf("\noffset-probe SKIPPED: %d hits exceeds -maxprobehits=%d (would need %d map entries)\n",
			len(hits), *maxProbeHits, uint64(len(hits))*(*maxDelta/8))
		return
	}
	offsetProbeRun(memF, mem, heap, hits, markers, *maxDelta, *maxK, isExe, isHeap)
}

// offsetProbeRun answers, without assuming Transform=384 or RootComponent=576:
// for each transform hit, what points into the object containing it, and at
// what offsets does a valid actor chain resolve?
func offsetProbeRun(f *os.File, mem memscan.MemReader, heap []memscan.Region, hits []xyHit, markers []marker, maxDelta, maxK uint64, isExe, isHeap func(uint64) bool) {
	type cand struct {
		hitIdx int
		delta  uint64
	}
	pointees := map[uint64][]cand{}
	for i, h := range hits {
		for d := uint64(0); d <= maxDelta; d += 8 {
			if h.Addr < d {
				break
			}
			pointees[h.Addr-d] = append(pointees[h.Addr-d], cand{i, d})
		}
	}
	fmt.Printf("\n=== OFFSET PROBE: %d hits -> %d candidate object-base addresses ===\n", len(hits), len(pointees))

	targets := map[uint64]bool{}
	for p := range pointees {
		targets[p] = true
	}
	type resolution struct {
		tOff, rcOff uint64
		actor, cls  uint64
		x, y, z     float64
		hitIdx      int
	}
	var res []resolution
	seen := map[[2]uint64]bool{}
	streamRegions(f, heap, new(int), new(uint64), func(base uint64, buf []byte) {
		for _, ref := range memscan.FindPointerReferencesMulti(buf, base, targets) {
			for _, c := range pointees[ref.Target] {
				for k := uint64(0); k <= maxK; k += 8 {
					if ref.Addr < k {
						break
					}
					actor := ref.Addr - k
					if seen[[2]uint64{actor, ref.Target}] {
						continue
					}
					vt, err := mem.ReadU64(actor)
					if err != nil || !isExe(vt) {
						continue
					}
					cp, err := mem.ReadU64(actor + memscan.DefaultOffsets.ClassPrivate)
					if err != nil || !isHeap(cp) {
						continue
					}
					cvt, err := mem.ReadU64(cp)
					if err != nil || !isExe(cvt) {
						continue
					}
					x, e1 := mem.ReadF64(ref.Target + c.delta)
					y, e2 := mem.ReadF64(ref.Target + c.delta + 8)
					z, e3 := mem.ReadF64(ref.Target + c.delta + 16)
					if e1 != nil || e2 != nil || e3 != nil || !memscan.ValidTransform(x, y, z) {
						continue
					}
					seen[[2]uint64{actor, ref.Target}] = true
					res = append(res, resolution{c.delta, k, actor, cp, x, y, z, c.hitIdx})
				}
			}
		}
	})
	fmt.Printf("offset probe: %d resolutions\n", len(res))
	combo := map[[2]uint64]int{}
	for _, r := range res {
		combo[[2]uint64{r.tOff, r.rcOff}]++
	}
	type kv struct {
		k [2]uint64
		n int
	}
	var list []kv
	for k, n := range combo {
		list = append(list, kv{k, n})
	}
	sort.Slice(list, func(a, b int) bool { return list[a].n > list[b].n })
	fmt.Println("most common (transformOffset, rootComponentOffset) combinations:")
	for j, e := range list {
		if j >= 15 {
			fmt.Printf("  ... %d more\n", len(list)-15)
			break
		}
		fmt.Printf("  transformOff=%-6d rootComponentOff=%-6d n=%d\n", e.k[0], e.k[1], e.n)
	}
	// per-marker: does any resolution land on it, and with what offsets
	fmt.Println("\nper-marker resolution (nearest resolution within 100uu):")
	for _, m := range markers {
		best := math.Inf(1)
		var br resolution
		ok := false
		for _, r := range res {
			d := math.Hypot(r.x-m.X, r.y-m.Y)
			if d < best {
				best, br, ok = d, r, true
			}
		}
		if ok && best <= 100 {
			fmt.Printf("  %-20s (%8.0f,%9.0f) d=%6.2f tOff=%-5d rcOff=%-5d class=%#x\n", m.T, m.X, m.Y, best, br.tOff, br.rcOff, br.cls)
		} else {
			fmt.Printf("  %-20s (%8.0f,%9.0f) UNRESOLVED (nearest %.0f)\n", m.T, m.X, m.Y, best)
		}
	}
}

func stagedValidate(mem memscan.MemReader, addr uint64, isExe, isHeap func(uint64) bool) string {
	vt, err := mem.ReadU64(addr)
	if err != nil {
		return "err-vtable-read"
	}
	if !isExe(vt) {
		return "rej-vtable"
	}
	cp, err := mem.ReadU64(addr + memscan.DefaultOffsets.ClassPrivate)
	if err != nil {
		return "err-class-read"
	}
	if !isHeap(cp) {
		return "rej-classprivate"
	}
	cvt, err := mem.ReadU64(cp)
	if err != nil {
		return "err-classvtable-read"
	}
	if !isExe(cvt) {
		return "rej-classvtable"
	}
	rc, err := mem.ReadU64(addr + memscan.DefaultOffsets.RootComponent)
	if err != nil {
		return "err-rc-read"
	}
	if !isHeap(rc) {
		return "rej-rootcomponent"
	}
	x, e1 := mem.ReadF64(rc + memscan.DefaultOffsets.Transform)
	y, e2 := mem.ReadF64(rc + memscan.DefaultOffsets.Transform + 8)
	z, e3 := mem.ReadF64(rc + memscan.DefaultOffsets.Transform + 16)
	if e1 != nil || e2 != nil || e3 != nil {
		return "err-transform-read"
	}
	if !memscan.ValidTransform(x, y, z) {
		return "rej-transform"
	}
	if _, err := mem.ReadI32(addr + memscan.DefaultOffsets.BaseValue); err != nil {
		return "err-basevalue-read"
	}
	return "ok"
}

func reportCoverage(title string, markers []marker, hits []xyHit) {
	fmt.Printf("\n=== %s ===\n", title)
	byType := map[string][]float64{}
	for _, m := range markers {
		best := math.Inf(1)
		for _, h := range hits {
			d := math.Hypot(h.X-m.X, h.Y-m.Y)
			if d < best {
				best = d
			}
		}
		byType[m.T] = append(byType[m.T], best)
	}
	var types []string
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)
	totN, totHit := 0, 0
	fmt.Printf("%-22s %4s %10s %8s\n", "type", "n", "min-dist", "<=100uu")
	for _, t := range types {
		ds := byType[t]
		sort.Float64s(ds)
		n := 0
		for _, d := range ds {
			if d <= 100 {
				n++
			}
		}
		totN += len(ds)
		totHit += n
		fmt.Printf("%-22s %4d %10.1f %8d\n", t, len(ds), ds[0], n)
	}
	fmt.Printf("%-22s %4d %10s %8d  (%.1f%%)\n", "TOTAL", totN, "", totHit, 100*float64(totHit)/float64(maxi(1, totN)))
}

func streamRegions(f *os.File, regions []memscan.Region, errN *int, errBytes *uint64, fn func(base uint64, buf []byte)) {
	buf := make([]byte, chunk+16)
	for _, r := range regions {
		size := r.End - r.Start
		var failed bool
		for off := uint64(0); off < size; off += chunk {
			n := uint64(chunk) + 16
			if off+n > size {
				n = size - off
			}
			b := buf[:n]
			if _, err := f.ReadAt(b, int64(r.Start+off)); err != nil {
				failed = true
				continue
			}
			fn(r.Start+off, b)
		}
		if failed {
			*errN++
			*errBytes += size
		}
	}
}

func inSet(rs []memscan.Region, a uint64) bool {
	for _, r := range rs {
		if r.Contains(a) {
			return true
		}
	}
	return false
}

func countDistinct(h []xyHit) int {
	m := map[[2]float64]bool{}
	for _, v := range h {
		m[[2]float64{v.X, v.Y}] = true
	}
	return len(m)
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

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}
