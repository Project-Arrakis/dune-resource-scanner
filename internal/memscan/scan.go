package memscan

import (
	"encoding/binary"
	"math"
)

// FindInt32LE scans buf for 4-byte-aligned occurrences of value as a
// little-endian int32. Returns the absolute address (baseAddr + offset) of
// each match.
func FindInt32LE(buf []byte, baseAddr uint64, value int32) []uint64 {
	var hits []uint64
	target := uint32(value)
	for off := 0; off+4 <= len(buf); off += 4 {
		if binary.LittleEndian.Uint32(buf[off:off+4]) == target {
			hits = append(hits, baseAddr+uint64(off))
		}
	}
	return hits
}

// FindNearbyXY scans buf for 8-byte-aligned float64 pairs (X, Y) where X is
// within tolerance of nearX and the immediately following float64 Y is
// within tolerance of nearY. Returns the absolute address of each matching
// X.
func FindNearbyXY(buf []byte, baseAddr uint64, nearX, nearY, tolerance float64) []uint64 {
	var hits []uint64
	for off := 0; off+16 <= len(buf); off += 8 {
		x := math.Float64frombits(binary.LittleEndian.Uint64(buf[off : off+8]))
		if !withinTolerance(x, nearX, tolerance) {
			continue
		}
		y := math.Float64frombits(binary.LittleEndian.Uint64(buf[off+8 : off+16]))
		if !withinTolerance(y, nearY, tolerance) {
			continue
		}
		hits = append(hits, baseAddr+uint64(off))
	}
	return hits
}

// withinTolerance reports whether v is within tolerance of target. It is
// written as a positive range test rather than the negation of an
// out-of-range test so that NaN falls out instead of through: every
// comparison involving NaN is false, so `math.Abs(v-target) > tolerance` is
// false for NaN and a `continue` guarding on it never fires. Arbitrary
// memory is full of byte patterns that decode as NaN -- 16 bytes of 0xFF
// (two int64 -1 values) is the most common -- and those matched every target
// at every tolerance, producing ~17M spurious hits in a single live scan.
// See issue #18.
func withinTolerance(v, target, tolerance float64) bool {
	d := v - target
	return d >= -tolerance && d <= tolerance
}

// FindPointerReferences scans buf for 8-byte-aligned occurrences of target
// as a raw little-endian uint64 pointer value. Returns the absolute address
// of each match.
func FindPointerReferences(buf []byte, baseAddr uint64, target uint64) []uint64 {
	var hits []uint64
	for off := 0; off+8 <= len(buf); off += 8 {
		if binary.LittleEndian.Uint64(buf[off:off+8]) == target {
			hits = append(hits, baseAddr+uint64(off))
		}
	}
	return hits
}

// PointerRef is one 8-byte-aligned occurrence of a pointer value found by
// FindPointerReferencesMulti.
type PointerRef struct {
	Addr   uint64 // absolute address of the reference itself
	Target uint64 // the target pointer value found there
}

// FindPointerReferencesMulti scans buf once for 8-byte-aligned occurrences
// of any value in targets, as raw little-endian uint64 pointer values. This
// lets a caller search for many target addresses in a single pass over a
// large region, instead of one pass per target.
func FindPointerReferencesMulti(buf []byte, baseAddr uint64, targets map[uint64]bool) []PointerRef {
	var refs []PointerRef
	for off := 0; off+8 <= len(buf); off += 8 {
		v := binary.LittleEndian.Uint64(buf[off : off+8])
		if targets[v] {
			refs = append(refs, PointerRef{Addr: baseAddr + uint64(off), Target: v})
		}
	}
	return refs
}
