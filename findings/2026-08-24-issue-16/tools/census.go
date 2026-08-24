//go:build ignore
// +build ignore

// Command census scans the whole world (not a box) for spawn-record-shaped
// node entries and emits them as JSON for offline analysis against the full
// dune.markers set. Investigation tool for issue #16; not part of the build.
//
//	go build -o census findings/2026-08-24-issue-16/tools/census.go
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"

	"dune-resource-scanner/internal/memscan"
)

const chunk = 64 << 20

// overlap must exceed the largest record-relative offset we read behind a hit,
// so a record straddling a chunk boundary is still fully readable in-buffer.
const overlap = 1024

// recSigConst is the constant observed at record+8 in the layout dump.
const recSigConst = 0x0000000100000001

// fieldOffsets covers the whole 384-byte record, so the offline
// type-discriminator search never needs a re-scan to test another offset.
var fieldOffsets = func() []int {
	var o []int
	for i := 0; i < 384; i += 8 {
		o = append(o, i)
	}
	return o
}()

// followWords is how many qwords to capture from whatever the record's +0
// pointer points at. That pointer is per-record rather than per-type, so it
// is a handle to a definition object -- the type is likely inside it.
const followWords = 32

type record struct {
	Addr    uint64            `json:"addr"`
	Base    uint64            `json:"base"`
	X       float64           `json:"x"`
	Y       float64           `json:"y"`
	Z       float64           `json:"z"`
	Strict  bool              `json:"strict"`
	Fields  map[string]uint64 `json:"fields"`
	HeapPtr bool              `json:"heap_ptr"`
	Follow  []uint64          `json:"follow,omitempty"`
}

func main() {
	pid := flag.Int("pid", 0, "target pid")
	out := flag.String("out", "", "output JSON path (required)")
	zMax := flag.Float64("zmax", 100000, "max plausible |z|")
	follow := flag.Bool("follow", false, "also capture the first 32 qwords at whatever the record's +0 pointer points at")
	relaxed := flag.Bool("relaxed", false, "also emit records matching only the heap-pointer test, without the +8 constant")
	maxRecords := flag.Int("maxrecords", 2000000, "abort if more than this many records match; guards against an unbounded working set on a host that also runs the live game server")
	flag.Parse()
	if *pid == 0 || *out == "" {
		fmt.Fprintln(os.Stderr, "need -pid and -out")
		os.Exit(2)
	}

	mapsF, err := os.Open(fmt.Sprintf("/proc/%d/maps", *pid))
	must(err)
	regions, err := memscan.ParseMaps(mapsF)
	must(err)
	mapsF.Close()
	memF, err := os.Open(fmt.Sprintf("/proc/%d/mem", *pid))
	must(err)
	defer memF.Close()

	heap := memscan.HeapLikeRegions(regions)
	sorted := append([]memscan.Region(nil), heap...)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a].Start < sorted[b].Start })
	isHeap := func(a uint64) bool {
		i := sort.Search(len(sorted), func(i int) bool { return sorted[i].End > a })
		return i < len(sorted) && sorted[i].Contains(a)
	}

	outF, err := os.Create(*out)
	must(err)
	defer outF.Close()
	w := bufio.NewWriterSize(outF, 1<<20)
	enc := json.NewEncoder(w)

	var raw, strictN, relaxedN, emitted uint64
	buf := make([]byte, chunk+overlap)
	abort := false
	for _, r := range heap {
		if abort {
			break
		}
		size := r.End - r.Start
		for off := uint64(0); off < size && !abort; off += chunk {
			n := uint64(chunk) + overlap
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
				if !plausibleXY(x) {
					continue
				}
				y := math.Float64frombits(binary.LittleEndian.Uint64(b[p+8 : p+16]))
				if !plausibleXY(y) {
					continue
				}
				z := math.Float64frombits(binary.LittleEndian.Uint64(b[p+16 : p+24]))
				if !memscan.ValidTransform(x, y, z) {
					continue
				}
				if nearOrigin(x, y) || math.Abs(z) > *zMax {
					continue
				}
				raw++
				if p < 256 {
					continue // record base falls before this buffer; skipped, counted via overlap
				}
				rb := p - 256
				sig := binary.LittleEndian.Uint64(b[rb+8 : rb+16])
				p0 := binary.LittleEndian.Uint64(b[rb : rb+8])
				strict := sig == recSigConst && isHeap(p0)
				loose := isHeap(p0)
				if !strict && !(*relaxed && loose) {
					continue
				}
				if strict {
					strictN++
				} else {
					relaxedN++
				}
				f := map[string]uint64{}
				for _, fo := range fieldOffsets {
					q := rb + fo
					if q >= 0 && q+8 <= len(b) {
						f[fmt.Sprintf("%d", fo)] = binary.LittleEndian.Uint64(b[q : q+8])
					}
				}
				var fw []uint64
				if *follow && isHeap(p0) {
					var fb [followWords * 8]byte
					if _, err := memF.ReadAt(fb[:], int64(p0)); err == nil {
						fw = make([]uint64, followWords)
						for k := 0; k < followWords; k++ {
							fw[k] = binary.LittleEndian.Uint64(fb[k*8 : k*8+8])
						}
					}
				}
				must(enc.Encode(record{
					Addr: base + uint64(p), Base: base + uint64(rb),
					X: x, Y: y, Z: z, Strict: strict, Fields: f, HeapPtr: loose,
					Follow: fw,
				}))
				emitted++
				if emitted >= uint64(*maxRecords) {
					fmt.Fprintf(os.Stderr, "ABORT: hit -maxrecords=%d; the match test is too permissive, tighten it\n", *maxRecords)
					abort = true
					break
				}
			}
		}
	}
	must(w.Flush())
	fmt.Printf("raw plausible-XYZ triples: %d\n", raw)
	fmt.Printf("strict signature records:  %d\n", strictN)
	fmt.Printf("relaxed-only records:      %d\n", relaxedN)
	fmt.Printf("total emitted:             %d (aborted=%v)\n", emitted, abort)
	fmt.Printf("wrote %s\n", *out)
}

// originRadius rejects the near-origin junk that a bare `a >= 1` test lets through.
// A live scan produced 1,709 records (2.0%) inside 1,000 uu of the world origin --
// 567 within 100 uu, 43 at exactly (1,1,1), and 373 at (360,360,0) -- none of which
// are real node positions.
//
// The test is RADIAL, not per-axis. Deep Desert crosses the origin, so a real node
// can legitimately have one small coordinate: 98 of 9,601 real DD markers have
// |x| or |y| below 1,000. None has BOTH below 1,000, so rejecting on distance from
// the origin drops the junk without losing any of them. A per-axis test would have
// discarded up to 98 real nodes.
const originRadius = 1000.0

// plausibleXY is a cheap pre-filter so the hot loop can reject most slots after
// reading only one qword. The authoritative check is memscan.ValidTransform on the
// full triple, which additionally rejects denormals, Inf and the exact origin --
// dropping it here let 13,281 records through with a zero or denormal coordinate.
func plausibleXY(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return math.Abs(v) <= memscan.WorldBound
}

// nearOrigin reports whether (x, y) sits inside originRadius of the world origin.
func nearOrigin(x, y float64) bool {
	return math.Hypot(x, y) < originRadius
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}
