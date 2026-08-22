package memscan

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Region is one mapped memory region from /proc/<pid>/maps.
type Region struct {
	Start, End uint64
	Perms      string
	Pathname   string
}

// Contains reports whether addr falls within [Start, End).
func (r Region) Contains(addr uint64) bool {
	return addr >= r.Start && addr < r.End
}

// ParseMaps parses the contents of /proc/<pid>/maps.
func ParseMaps(r io.Reader) ([]Region, error) {
	var regions []Region
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("malformed maps line: %q", line)
		}
		addrRange := strings.SplitN(fields[0], "-", 2)
		if len(addrRange) != 2 {
			return nil, fmt.Errorf("malformed address range: %q", fields[0])
		}
		start, err := strconv.ParseUint(addrRange[0], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid start address %q: %w", addrRange[0], err)
		}
		end, err := strconv.ParseUint(addrRange[1], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid end address %q: %w", addrRange[1], err)
		}
		pathname := ""
		if len(fields) >= 6 {
			pathname = strings.Join(fields[5:], " ")
		}
		regions = append(regions, Region{
			Start:    start,
			End:      end,
			Perms:    fields[1],
			Pathname: pathname,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return regions, nil
}

// FilterExecutable returns the regions with execute permission.
func FilterExecutable(regions []Region) []Region {
	var out []Region
	for _, r := range regions {
		if len(r.Perms) >= 3 && r.Perms[2] == 'x' {
			out = append(out, r)
		}
	}
	return out
}

// FilterByPathname returns the regions whose Pathname exactly matches name.
func FilterByPathname(regions []Region, name string) []Region {
	var out []Region
	for _, r := range regions {
		if r.Pathname == name {
			out = append(out, r)
		}
	}
	return out
}

// MainModuleRegions returns every region sharing the same backing file as
// the process's main executable image -- text, rodata, and data segments
// are all part of one module, and a vtable pointer can land in any of them
// (relocated vtables/RTTI commonly live in the read-only rodata segment,
// not the executable-permission text segment), so validating a vtable
// pointer against only the exec-permission region misses real vtables.
// The main executable is identified as the first region with execute
// permission and a backing file.
func MainModuleRegions(regions []Region) []Region {
	var mainPath string
	for _, r := range regions {
		if len(r.Perms) >= 3 && r.Perms[2] == 'x' && r.Pathname != "" {
			mainPath = r.Pathname
			break
		}
	}
	if mainPath == "" {
		return nil
	}
	return FilterByPathname(regions, mainPath)
}

// HeapLikeRegions returns every region that could hold heap-allocated
// objects: the classic glibc [heap], plus every anonymous (no backing file)
// writable mapping. Engines with their own allocator (confirmed live
// against the Dune Awakening server: [heap] is only a few MB, while real
// game-object allocations live in dozens of large anonymous rw mmap
// regions) put real actor data outside [heap] entirely, so scanning
// [heap] alone misses almost everything.
func HeapLikeRegions(regions []Region) []Region {
	var out []Region
	for _, r := range regions {
		if len(r.Perms) < 2 || r.Perms[1] != 'w' {
			continue
		}
		if r.Pathname == "[heap]" || r.Pathname == "" {
			out = append(out, r)
		}
	}
	return out
}
