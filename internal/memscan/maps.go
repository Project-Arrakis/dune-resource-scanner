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
