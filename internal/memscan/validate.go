package memscan

import "math"

// WorldBound is the maximum absolute X/Y coordinate magnitude considered
// plausible for an in-world actor position.
const WorldBound = 1_250_000.0

// ValidTransform reports whether (x, y, z) is a plausible in-world actor
// position: no NaN component, X and Y within WorldBound, and not the exact
// origin (which a null/uninitialized actor reads as).
func ValidTransform(x, y, z float64) bool {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) {
		return false
	}
	if math.Abs(x) > WorldBound || math.Abs(y) > WorldBound {
		return false
	}
	if x == 0 && y == 0 && z == 0 {
		return false
	}
	return true
}
