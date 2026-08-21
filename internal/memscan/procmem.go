package memscan

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// ProcMem implements MemReader over any io.ReaderAt indexed by virtual
// address — in production, an *os.File opened on /proc/<pid>/mem.
type ProcMem struct {
	r io.ReaderAt
}

// NewProcMem wraps r as a MemReader.
func NewProcMem(r io.ReaderAt) *ProcMem {
	return &ProcMem{r: r}
}

func (p *ProcMem) ReadU64(addr uint64) (uint64, error) {
	var buf [8]byte
	if _, err := p.r.ReadAt(buf[:], int64(addr)); err != nil {
		return 0, fmt.Errorf("read u64 at %#x: %w", addr, err)
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}

func (p *ProcMem) ReadF64(addr uint64) (float64, error) {
	v, err := p.ReadU64(addr)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(v), nil
}

func (p *ProcMem) ReadI32(addr uint64) (int32, error) {
	var buf [4]byte
	if _, err := p.r.ReadAt(buf[:], int64(addr)); err != nil {
		return 0, fmt.Errorf("read i32 at %#x: %w", addr, err)
	}
	return int32(binary.LittleEndian.Uint32(buf[:])), nil
}

// ReadRegion reads an entire region's bytes from r.
func ReadRegion(r io.ReaderAt, region Region) ([]byte, error) {
	size := region.End - region.Start
	buf := make([]byte, size)
	if _, err := r.ReadAt(buf, int64(region.Start)); err != nil {
		return nil, fmt.Errorf("read region %#x-%#x: %w", region.Start, region.End, err)
	}
	return buf, nil
}
