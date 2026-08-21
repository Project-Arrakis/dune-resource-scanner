package memscan

import (
	"math"
	"testing"
)

func TestValidTransform_AcceptsPlausibleInWorldPosition(t *testing.T) {
	if !ValidTransform(-611736.35, -700183.46, 500.0) {
		t.Fatal("expected a real, known in-world position to validate")
	}
}

func TestValidTransform_RejectsNaN(t *testing.T) {
	if ValidTransform(math.NaN(), 0, 0) {
		t.Fatal("expected NaN X to be rejected")
	}
	if ValidTransform(0, math.NaN(), 0) {
		t.Fatal("expected NaN Y to be rejected")
	}
	if ValidTransform(0, 0, math.NaN()) {
		t.Fatal("expected NaN Z to be rejected")
	}
}

func TestValidTransform_RejectsOutOfWorldBounds(t *testing.T) {
	if ValidTransform(1_250_001, 0, 0) {
		t.Fatal("expected X beyond world bound to be rejected")
	}
	if ValidTransform(0, -1_250_001, 0) {
		t.Fatal("expected Y beyond world bound to be rejected")
	}
}

func TestValidTransform_AcceptsExactlyAtWorldBound(t *testing.T) {
	if !ValidTransform(1_250_000, 1_250_000, 0) {
		t.Fatal("expected exactly-at-bound position to be accepted")
	}
}

func TestValidTransform_RejectsExactOrigin(t *testing.T) {
	if ValidTransform(0, 0, 0) {
		t.Fatal("expected exact origin (0,0,0) to be rejected as a null/uninitialized actor")
	}
}
