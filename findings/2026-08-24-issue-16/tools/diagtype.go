//go:build ignore
// +build ignore

// Command diagtype is a throwaway investigation tool for issue #16. Pass 1
// already locates 100% of known nodes; what is missing is a resource TYPE for
// each hit. This searches the bytes around each hit for a field whose value
// partitions the known markers by their type -- i.e. a type discriminator --
// without assuming any particular record layout.
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

type hit struct {
	addr    uint64
	x, y, z float64
}

func main() {
	pid := flag.Int("pid", 0, "target pid")
	near := flag.String("near", "", "x,y box centre")
	tol := flag.Float64("tolerance", 15000, "per-axis half width")
	mkFile := flag.String("markers", "", "csv: type,x,y")
	recSig := flag.Bool("recsig", false, "keep only hits matching the observed 384-byte spawn-record signature")
	lo := flag.Int64("lo", -2048, "lowest byte offset from hit to test")
	hi := flag.Int64("hi", 512, "highest byte offset from hit to test")
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

	var hits []hit
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
			for _, a := range memscan.FindNearbyXY(b, r.Start+off, nx, ny, *tol) {
				o := a - (r.Start + off)
				z := math.Float64frombits(le64(b[o+16 : o+24]))
				if math.Abs(z) > 100000 || math.IsNaN(z) {
					continue // keep the terrain-Z copy, not the pre-trace sentinel
				}
				hits = append(hits, hit{a,
					math.Float64frombits(le64(b[o : o+8])),
					math.Float64frombits(le64(b[o+8 : o+16])), z})
			}
		}
	}
	fmt.Printf("plausible-Z hits: %d\n", len(hits))
	if *recSig {
		var keep []hit
		for _, h := range hits {
			if h.addr < 256 {
				continue
			}
			base := h.addr - 256
			p0, e0 := mem.ReadU64(base)
			p1, e1 := mem.ReadU64(base + 8)
			if e0 != nil || e1 != nil {
				continue
			}
			if !inHeap(heap, p0) || p1 != 0x0000000100000001 {
				continue
			}
			keep = append(keep, h)
		}
		hits = keep
		fmt.Printf("hits matching spawn-record signature (base=hit-256, +0 heap ptr, +8 == 0x100000001): %d\n", len(hits))
	}

	// For each marker, keep every hit within 100uu (a node may have several copies).
	type mh struct {
		m    marker
		hits []hit
	}
	var mhs []mh
	for _, m := range markers {
		var hs []hit
		for _, h := range hits {
			if math.Hypot(h.x-m.X, h.y-m.Y) <= 100 {
				hs = append(hs, h)
			}
		}
		if len(hs) > 0 {
			mhs = append(mhs, mh{m, hs})
		}
	}
	fmt.Printf("markers with >=1 hit: %d/%d\n", len(mhs), len(markers))
	copies := map[int]int{}
	for _, e := range mhs {
		copies[len(e.hits)]++
	}
	var ck []int
	for k := range copies {
		ck = append(ck, k)
	}
	sort.Ints(ck)
	fmt.Print("copies per marker: ")
	for _, k := range ck {
		fmt.Printf("%dx:%d ", k, copies[k])
	}
	fmt.Println()

	// Score each candidate offset by how cleanly its value partitions types.
	type score struct {
		off        int64
		pureTypes  int
		collisions int
		distinct   int
		readable   int
	}
	var scores []score
	for off := *lo; off <= *hi; off += 8 {
		valuesByType := map[string]map[uint64]int{}
		readable := 0
		for _, e := range mhs {
			set := map[uint64]bool{}
			for _, h := range e.hits {
				a := uint64(int64(h.addr) + off)
				v, err := mem.ReadU64(a)
				if err != nil {
					continue
				}
				set[v] = true
			}
			if len(set) == 0 {
				continue
			}
			readable++
			if valuesByType[e.m.T] == nil {
				valuesByType[e.m.T] = map[uint64]int{}
			}
			for v := range set {
				valuesByType[e.m.T][v]++
			}
		}
		// A type counts as resolved only if some value is shared by EVERY marker
		// of that type AND appears for no marker of any other type. A value like
		// 0 or a common vtable is present everywhere and must not score.
		owner := map[uint64]map[string]bool{}
		distinct := map[uint64]bool{}
		for t, vals := range valuesByType {
			for v := range vals {
				distinct[v] = true
				if owner[v] == nil {
					owner[v] = map[string]bool{}
				}
				owner[v][t] = true
			}
		}
		pure := 0
		for t, vals := range valuesByType {
			nT := 0
			for _, e := range mhs {
				if e.m.T == t {
					nT++
				}
			}
			if nT < 2 {
				continue
			}
			for v, c := range vals {
				if c == nT && len(owner[v]) == 1 {
					pure++
					break
				}
			}
		}
		coll := 0
		for _, ts := range owner {
			if len(ts) > 1 {
				coll++
			}
		}
		scores = append(scores, score{off, pure, coll, len(distinct), readable})
	}
	sort.Slice(scores, func(a, b int) bool {
		if scores[a].pureTypes != scores[b].pureTypes {
			return scores[a].pureTypes > scores[b].pureTypes
		}
		return scores[a].collisions < scores[b].collisions
	})
	fmt.Println("\ntop candidate type-discriminator offsets (pure = types whose every marker shares one value):")
	fmt.Printf("%8s %6s %11s %9s %9s\n", "offset", "pure", "collisions", "distinct", "readable")
	for i, s := range scores {
		if i >= 20 {
			break
		}
		fmt.Printf("%8d %6d %11d %9d %9d\n", s.off, s.pureTypes, s.collisions, s.distinct, s.readable)
	}

	// Detail for the single best offset.
	if len(scores) > 0 && scores[0].pureTypes > 0 {
		best := scores[0].off
		fmt.Printf("\n=== values at offset %d, by marker type ===\n", best)
		byType := map[string]map[uint64]int{}
		for _, e := range mhs {
			for _, h := range e.hits {
				v, err := mem.ReadU64(uint64(int64(h.addr) + best))
				if err != nil {
					continue
				}
				if byType[e.m.T] == nil {
					byType[e.m.T] = map[uint64]int{}
				}
				byType[e.m.T][v]++
			}
		}
		var ts []string
		for t := range byType {
			ts = append(ts, t)
		}
		sort.Strings(ts)
		for _, t := range ts {
			var vs []string
			type kv struct {
				v uint64
				n int
			}
			var l []kv
			for v, n := range byType[t] {
				l = append(l, kv{v, n})
			}
			sort.Slice(l, func(a, b int) bool { return l[a].n > l[b].n })
			for i, e := range l {
				if i >= 4 {
					vs = append(vs, fmt.Sprintf("+%d more", len(l)-4))
					break
				}
				vs = append(vs, fmt.Sprintf("%#x(%d)", e.v, e.n))
			}
			fmt.Printf("  %-22s %s\n", t, strings.Join(vs, " "))
		}
	}
}

func inHeap(rs []memscan.Region, a uint64) bool {
	for _, r := range rs {
		if r.Contains(a) {
			return true
		}
	}
	return false
}

func le64(b []byte) uint64 {
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
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
