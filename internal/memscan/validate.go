package memscan

import "math"

// WorldBound is the maximum absolute coordinate magnitude considered
// plausible for an in-world actor position, on every axis. Live data on
// dune-dev reaches |x| = 1,130,162, so this leaves real positions comfortably
// inside the bound while rejecting garbage.
const WorldBound = 1_250_000.0

// minMagnitude is the smallest absolute non-zero coordinate treated as real.
// The world is measured in centimetres, so a sub-micrometre component is not
// a position -- it is uninitialized memory. Denormalized doubles (~1e-310)
// are the observed form: effectively zero, but not == 0, so the exact-origin
// guard alone never fired on them.
const minMagnitude = 1e-6

// ValidTransform reports whether (x, y, z) is a plausible in-world actor
// position: every component finite (no NaN, no Inf), within WorldBound on
// every axis including Z, no implausibly-tiny non-zero magnitude, and not the
// exact origin (which a null/uninitialized actor reads as).
func ValidTransform(x, y, z float64) bool {
	for _, v := range [3]float64{x, y, z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
		if math.Abs(v) > WorldBound {
			return false
		}
		if v != 0 && math.Abs(v) < minMagnitude {
			return false
		}
	}
	if x == 0 && y == 0 && z == 0 {
		return false
	}
	return true
}
