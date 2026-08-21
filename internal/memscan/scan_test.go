package memscan

import (
	"encoding/binary"
	"math"
	"testing"
)

func le32(v int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	return b
}

func le64(bits uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, bits)
	return b
}

func leF64(f float64) []byte {
	return le64(math.Float64bits(f))
}

func TestFindInt32LE_FindsAlignedMatch(t *testing.T) {
	buf := append(append([]byte{0, 0, 0, 0}, le32(5000)...), []byte{0, 0, 0, 0}...)
	hits := FindInt32LE(buf, 0x1000, 5000)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d: %v", len(hits), hits)
	}
	if hits[0] != 0x1004 {
		t.Fatalf("expected hit at 0x1004, got %#x", hits[0])
	}
}

func TestFindInt32LE_IgnoresUnalignedCoincidence(t *testing.T) {
	// Value appears at a byte offset that isn't 4-byte aligned relative to base.
	buf := append([]byte{0xFF}, le32(5000)...)
	hits := FindInt32LE(buf, 0x1000, 5000)
	if len(hits) != 0 {
		t.Fatalf("expected 0 aligned hits, got %d: %v", len(hits), hits)
	}
}

func TestFindInt32LE_FindsMultipleHits(t *testing.T) {
	buf := append(append(le32(5000), le32(0)...), le32(5000)...)
	hits := FindInt32LE(buf, 0, 5000)
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d: %v", len(hits), hits)
	}
	if hits[0] != 0 || hits[1] != 8 {
		t.Fatalf("unexpected hit addresses: %v", hits)
	}
}

func TestFindNearbyXY_FindsMatchWithinTolerance(t *testing.T) {
	// X, Y pair close to a known base position, at 8-byte-aligned offset 16.
	buf := make([]byte, 16)
	buf = append(buf, leF64(-611736.30)...) // X, within 1.0 of -611736.35
	buf = append(buf, leF64(-700183.50)...) // Y, within 1.0 of -700183.46

	hits := FindNearbyXY(buf, 0x2000, -611736.35, -700183.46, 1.0)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d: %v", len(hits), hits)
	}
	if hits[0] != 0x2010 {
		t.Fatalf("expected hit at 0x2010, got %#x", hits[0])
	}
}

func TestFindNearbyXY_RejectsOutsideTolerance(t *testing.T) {
	buf := append(leF64(-611736.35), leF64(-705000.00)...) // Y far outside tolerance
	hits := FindNearbyXY(buf, 0, -611736.35, -700183.46, 1.0)
	if len(hits) != 0 {
		t.Fatalf("expected 0 hits, got %d: %v", len(hits), hits)
	}
}

func TestFindPointerReferences_FindsAlignedMatch(t *testing.T) {
	target := uint64(0x7f0a12345678)
	buf := append(append(make([]byte, 8), le64(target)...), make([]byte, 8)...)
	hits := FindPointerReferences(buf, 0x3000, target)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d: %v", len(hits), hits)
	}
	if hits[0] != 0x3008 {
		t.Fatalf("expected hit at 0x3008, got %#x", hits[0])
	}
}

func TestFindPointerReferences_IgnoresUnalignedCoincidence(t *testing.T) {
	target := uint64(0x7f0a12345678)
	buf := append([]byte{0xAB}, le64(target)...)
	hits := FindPointerReferences(buf, 0, target)
	if len(hits) != 0 {
		t.Fatalf("expected 0 aligned hits, got %d: %v", len(hits), hits)
	}
}
