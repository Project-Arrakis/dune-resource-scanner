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
		if math.Abs(x-nearX) > tolerance {
			continue
		}
		y := math.Float64frombits(binary.LittleEndian.Uint64(buf[off+8 : off+16]))
		if math.Abs(y-nearY) > tolerance {
			continue
		}
		hits = append(hits, baseAddr+uint64(off))
	}
	return hits
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
