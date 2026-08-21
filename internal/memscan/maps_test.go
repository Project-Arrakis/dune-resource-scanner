package memscan

import (
	"strings"
	"testing"
)

func newMapsReader() *strings.Reader {
	return strings.NewReader(sampleMaps)
}

// A trimmed but realistic sample of /proc/<pid>/maps content: one executable
// text segment, one heap segment, and one unrelated read-only mapped file,
// to prove parsing doesn't just grab everything.
const sampleMaps = `55f1a0000000-55f1a1000000 r-xp 00000000 08:01 1234567                    /home/dune/DeepDesertServer
55f1a1000000-55f1a1200000 r--p 01000000 08:01 1234567                    /home/dune/DeepDesertServer
7f0a00000000-7f0a10000000 rw-p 00000000 00:00 0                          [heap]
7f0a20000000-7f0a20010000 r--p 00000000 08:01 7654321                   /usr/lib/libc.so.6
7f0a30000000-7f0a30001000 rw-p 00000000 00:00 0
`

func TestParseMaps_ParsesAllRegions(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 5 {
		t.Fatalf("expected 5 regions, got %d", len(regions))
	}
}

func TestParseMaps_ParsesStartEndPermsAndPathname(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	first := regions[0]
	if first.Start != 0x55f1a0000000 || first.End != 0x55f1a1000000 {
		t.Fatalf("unexpected start/end: %#x-%#x", first.Start, first.End)
	}
	if first.Perms != "r-xp" {
		t.Fatalf("unexpected perms: %q", first.Perms)
	}
	if first.Pathname != "/home/dune/DeepDesertServer" {
		t.Fatalf("unexpected pathname: %q", first.Pathname)
	}
}

func TestParseMaps_HandlesAnonymousRegionWithNoPathname(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := regions[len(regions)-1]
	if last.Pathname != "" {
		t.Fatalf("expected empty pathname for anonymous region, got %q", last.Pathname)
	}
}

func TestRegion_Contains(t *testing.T) {
	r := Region{Start: 0x1000, End: 0x2000}
	if !r.Contains(0x1000) {
		t.Fatal("expected start address to be contained (inclusive)")
	}
	if !r.Contains(0x1fff) {
		t.Fatal("expected last byte before end to be contained")
	}
	if r.Contains(0x2000) {
		t.Fatal("expected end address to be exclusive")
	}
	if r.Contains(0xfff) {
		t.Fatal("expected address before start to not be contained")
	}
}

func TestFilterExecutable_ReturnsOnlyExecRegions(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exe := FilterExecutable(regions)
	if len(exe) != 1 {
		t.Fatalf("expected 1 executable region, got %d", len(exe))
	}
	if exe[0].Pathname != "/home/dune/DeepDesertServer" {
		t.Fatalf("unexpected executable region: %+v", exe[0])
	}
}

func TestFilterByPathname_ReturnsMatchingRegions(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	heap := FilterByPathname(regions, "[heap]")
	if len(heap) != 1 {
		t.Fatalf("expected 1 heap region, got %d", len(heap))
	}
	if heap[0].Start != 0x7f0a00000000 {
		t.Fatalf("unexpected heap region start: %#x", heap[0].Start)
	}
}

// HeapLikeRegions must cover both the classic glibc [heap] AND every
// anonymous read-write mapping, because a game engine with its own
// allocator (confirmed live against dune-dev: [heap] was ~3MB while the
// actual game-object allocations live in dozens of large anonymous rw
// mmap regions up to 4GB each) puts real actor data outside [heap] entirely.
func TestHeapLikeRegions_IncludesNamedHeap(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	heapLike := HeapLikeRegions(regions)
	found := false
	for _, r := range heapLike {
		if r.Pathname == "[heap]" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected [heap] region to be included")
	}
}

func TestHeapLikeRegions_IncludesAnonymousRWRegions(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	heapLike := HeapLikeRegions(regions)
	found := false
	for _, r := range heapLike {
		if r.Start == 0x7f0a30000000 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the anonymous rw region to be included")
	}
}

func TestHeapLikeRegions_ExcludesFileBackedRegions(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	heapLike := HeapLikeRegions(regions)
	for _, r := range heapLike {
		if r.Pathname != "" && r.Pathname != "[heap]" {
			t.Fatalf("expected no file-backed regions, found %+v", r)
		}
	}
}

func TestHeapLikeRegions_ExcludesReadOnlyRegions(t *testing.T) {
	regions, err := ParseMaps(newMapsReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	heapLike := HeapLikeRegions(regions)
	for _, r := range heapLike {
		if len(r.Perms) < 2 || r.Perms[1] != 'w' {
			t.Fatalf("expected only writable regions, found %+v", r)
		}
	}
}
