package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// A scan that finds nothing must encode as a JSON array ("[]"), not "null".
// Consumers of this tool's output (the census pipeline) parse it as a JSON
// array unconditionally; "null" breaks that parse. See sessions/CONTINUATION-PROMPT.md.

func TestScanSeedsEmptyHeapReturnsEmptySliceNotNil(t *testing.T) {
	results := scanSeeds(nil, nil, nil, nil, alwaysFalse, alwaysFalse)

	if results == nil {
		t.Fatalf("scanSeeds returned nil, want non-nil empty slice")
	}
	if len(results) != 0 {
		t.Fatalf("scanSeeds returned %d results, want 0", len(results))
	}
}

func TestScanProximityEmptyHeapReturnsEmptySliceNotNil(t *testing.T) {
	results := scanProximity(nil, nil, nil, 0, 0, 5.0, alwaysFalse, alwaysFalse)

	if results == nil {
		t.Fatalf("scanProximity returned nil, want non-nil empty slice")
	}
	if len(results) != 0 {
		t.Fatalf("scanProximity returned %d results, want 0", len(results))
	}
}

func TestEmptyResultsEncodeAsJSONArray(t *testing.T) {
	results := scanSeeds(nil, nil, nil, nil, alwaysFalse, alwaysFalse)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(results); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	got := buf.String()
	want := "[]\n"
	if got != want {
		t.Fatalf("encoded empty results as %q, want %q", got, want)
	}
}

func alwaysFalse(uint64) bool { return false }
