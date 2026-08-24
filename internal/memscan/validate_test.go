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

// The denormal case from issue #14: a proximity scan reported this as its only
// "valid" hit. The components are denormalized doubles -- effectively zero but
// not == 0, so the exact-origin guard alone does not fire.
func TestValidTransform_RejectsDenormalNearZero(t *testing.T) {
	if ValidTransform(6.7894276969319e-310, 0, 4.8317235615046e-310) {
		t.Fatal("expected denormal near-zero coordinates to be rejected")
	}
}

func TestValidTransform_RejectsImplausiblyTinyMagnitude(t *testing.T) {
	if ValidTransform(1e-7, 1e-7, 1e-7) {
		t.Fatal("expected sub-micrometre coordinates to be rejected")
	}
}

func TestValidTransform_RejectsInf(t *testing.T) {
	for _, tc := range []struct {
		name    string
		x, y, z float64
	}{
		{"x=+Inf", math.Inf(1), 0, 0},
		{"y=-Inf", 0, math.Inf(-1), 0},
		{"z=+Inf", 1000, 1000, math.Inf(1)},
		{"z=-Inf", 1000, 1000, math.Inf(-1)},
	} {
		if ValidTransform(tc.x, tc.y, tc.z) {
			t.Fatalf("expected %s to be rejected", tc.name)
		}
	}
}

// Z was previously unbounded: only X and Y were range-checked, so a garbage Z
// of any finite magnitude passed. See issue #16's "two more scanner defects".
func TestValidTransform_RejectsOutOfWorldZ(t *testing.T) {
	if ValidTransform(1000, 1000, 1_250_001) {
		t.Fatal("expected Z beyond world bound to be rejected")
	}
	if ValidTransform(1000, 1000, -1_250_001) {
		t.Fatal("expected negative Z beyond world bound to be rejected")
	}
}

// Real observed extremes must keep validating. Across dune-dev's full marker
// set (12,667 rows, both maps) and live dune.actors, |z| never exceeds 75,246
// and |x| reaches 1,130,162 -- so a real deep-cave or high-city actor must not
// be rejected by the new guards.
func TestValidTransform_AcceptsRealObservedExtremes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		x, y, z float64
	}{
		{"hagga deepest marker", 100, 100, -21800},
		{"hagga highest marker", 100, 100, 75246},
		{"arrakeen city height", 100, 100, 55240},
		{"deep desert far x", 1130162, -700598, 4000},
		{"z exactly zero with real xy", -611736.35, -700183.46, 0},
	} {
		if !ValidTransform(tc.x, tc.y, tc.z) {
			t.Fatalf("expected %s to be accepted", tc.name)
		}
	}
}
