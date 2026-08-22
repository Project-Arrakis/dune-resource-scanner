// Command dune-resource-scanner scans a running Dune Awakening server
// process's memory to find world positions of raw resources that have no
// database-backed position (ore/stone/fiber actors), and cross-checks
// database-tracked resource fields (spice, the unidentified field_kind_id=0
// resource) against their live in-memory positions.
//
// It never writes to the target process — /proc/<pid>/mem is opened
// read-only, matching the design constraint that this tool run as a
// scheduled host job, never a continuous privileged sidecar.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"dune-resource-scanner/internal/memscan"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// result is one found actor, enriched with the seed label/value that led to
// it (seed mode only — empty/zero in proximity mode).
type result struct {
	memscan.ActorInfo
	Label string `json:"label,omitempty"`
	Value int32  `json:"value,omitempty"`
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("dune-resource-scanner", flag.ContinueOnError)
	pid := fs.Int("pid", 0, "target process ID (required)")
	mode := fs.String("mode", "seed", "scan mode: seed | proximity")
	seeds := fs.String("seeds", "", "seed mode: comma-separated label=int32value pairs, e.g. spice-small=5000,mystery=60000")
	near := fs.String("near", "", "proximity mode: x,y of a known nearby position")
	tolerance := fs.Float64("tolerance", 5.0, "proximity mode: match tolerance in world units")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *pid == 0 {
		return fmt.Errorf("-pid is required")
	}

	memPath := fmt.Sprintf("/proc/%d/mem", *pid)
	mapsPath := fmt.Sprintf("/proc/%d/maps", *pid)

	mapsFile, err := os.Open(mapsPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", mapsPath, err)
	}
	defer mapsFile.Close()

	regions, err := memscan.ParseMaps(mapsFile)
	if err != nil {
		return fmt.Errorf("parse %s: %w", mapsPath, err)
	}

	memFile, err := os.Open(memPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", memPath, err)
	}
	defer memFile.Close()

	exeRegions := memscan.MainModuleRegions(regions)
	heapRegions := memscan.HeapLikeRegions(regions)
	if len(heapRegions) == 0 {
		return fmt.Errorf("no heap-like (anonymous writable) regions found in %s", mapsPath)
	}

	isExe := func(addr uint64) bool { return regionSetContains(exeRegions, addr) }
	isHeap := func(addr uint64) bool { return regionSetContains(heapRegions, addr) }

	mem := memscan.NewProcMem(memFile)

	var results []result
	switch *mode {
	case "seed":
		pairs, err := parseSeeds(*seeds)
		if err != nil {
			return err
		}
		results = scanSeeds(memFile, mem, heapRegions, pairs, isExe, isHeap)
	case "proximity":
		nx, ny, err := parseNear(*near)
		if err != nil {
			return err
		}
		results = scanProximity(memFile, mem, heapRegions, nx, ny, *tolerance, isExe, isHeap)
	default:
		return fmt.Errorf("unknown -mode %q (want seed or proximity)", *mode)
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func regionSetContains(regions []memscan.Region, addr uint64) bool {
	for _, r := range regions {
		if r.Contains(addr) {
			return true
		}
	}
	return false
}

type seedPair struct {
	Label string
	Value int32
}

func parseSeeds(s string) ([]seedPair, error) {
	if s == "" {
		return nil, fmt.Errorf("-seeds is required in seed mode")
	}
	var pairs []seedPair
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid -seeds entry %q, want label=value", part)
		}
		v, err := strconv.ParseInt(kv[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid seed value in %q: %w", part, err)
		}
		pairs = append(pairs, seedPair{Label: kv[0], Value: int32(v)})
	}
	return pairs, nil
}

func parseNear(s string) (float64, float64, error) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("-near is required in proximity mode, want x,y")
	}
	x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid -near x: %w", err)
	}
	y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid -near y: %w", err)
	}
	return x, y, nil
}

// scanSeeds finds each seed value in the heap, then walks backward from each
// hit (candidate BaseValue field) to the owning actor's base address and
// validates it.
func scanSeeds(r io.ReaderAt, mem memscan.MemReader, heap []memscan.Region, pairs []seedPair, isExe, isHeap func(uint64) bool) []result {
	var results []result
	for _, region := range heap {
		buf, err := memscan.ReadRegion(r, region)
		if err != nil {
			continue // region may have been unmapped between maps snapshot and read; skip
		}
		for _, p := range pairs {
			for _, hit := range memscan.FindInt32LE(buf, region.Start, p.Value) {
				if hit < memscan.DefaultOffsets.BaseValue {
					continue
				}
				actorAddr := hit - memscan.DefaultOffsets.BaseValue
				if info, ok := memscan.ValidateActor(mem, actorAddr, memscan.DefaultOffsets, isExe, isHeap); ok {
					results = append(results, result{ActorInfo: info, Label: p.Label, Value: p.Value})
				}
			}
		}
	}
	return results
}

// scanProximity finds (X,Y) transform hits near a known position, then
// resolves each hit back to its owning actor by finding a pointer reference
// to the root component and walking that back to the actor base address.
//
// This does exactly two sequential passes over the heap, regardless of how
// many hits are found: a naive implementation that re-scans the whole heap
// for every transform hit is O(hits x heap size) and does not scale once
// there are dozens of nearby actors (as expected when surveying a whole
// island rather than one known point) against a multi-gigabyte heap.
func scanProximity(r io.ReaderAt, mem memscan.MemReader, heap []memscan.Region, nearX, nearY, tolerance float64, isExe, isHeap func(uint64) bool) []result {
	// Pass 1: find every candidate RootComponent address in one sweep.
	rootComponents := map[uint64]bool{}
	for _, region := range heap {
		buf, err := memscan.ReadRegion(r, region)
		if err != nil {
			continue
		}
		for _, txHit := range memscan.FindNearbyXY(buf, region.Start, nearX, nearY, tolerance) {
			if txHit < memscan.DefaultOffsets.Transform {
				continue
			}
			rootComponents[txHit-memscan.DefaultOffsets.Transform] = true
		}
	}
	if len(rootComponents) == 0 {
		return nil
	}

	// Pass 2: find every reference to any of those RootComponent addresses
	// in one more sweep, then validate each as a real actor.
	var results []result
	for _, region := range heap {
		buf, err := memscan.ReadRegion(r, region)
		if err != nil {
			continue
		}
		for _, ref := range memscan.FindPointerReferencesMulti(buf, region.Start, rootComponents) {
			if ref.Addr < memscan.DefaultOffsets.RootComponent {
				continue
			}
			actorAddr := ref.Addr - memscan.DefaultOffsets.RootComponent
			if info, ok := memscan.ValidateActor(mem, actorAddr, memscan.DefaultOffsets, isExe, isHeap); ok {
				results = append(results, result{ActorInfo: info})
			}
		}
	}
	return results
}
