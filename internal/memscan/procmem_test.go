package memscan

import (
	"bytes"
	"errors"
	"testing"
)

type failingReaderAt struct{}

func (failingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	return 0, errors.New("simulated read failure")
}

func TestProcMem_ReadU64ReturnsLittleEndianValue(t *testing.T) {
	buf := make([]byte, 16)
	copy(buf[8:], le64(0x1122334455667788))
	pm := NewProcMem(bytes.NewReader(buf))

	v, err := pm.ReadU64(8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 0x1122334455667788 {
		t.Fatalf("unexpected value: %#x", v)
	}
}

func TestProcMem_ReadF64ReturnsFloatBits(t *testing.T) {
	buf := leF64(-611736.35)
	pm := NewProcMem(bytes.NewReader(buf))

	v, err := pm.ReadF64(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != -611736.35 {
		t.Fatalf("unexpected value: %v", v)
	}
}

func TestProcMem_ReadI32ReturnsLittleEndianValue(t *testing.T) {
	buf := le32(5000)
	pm := NewProcMem(bytes.NewReader(buf))

	v, err := pm.ReadI32(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 5000 {
		t.Fatalf("unexpected value: %d", v)
	}
}

func TestProcMem_ReadU64PropagatesReadError(t *testing.T) {
	pm := NewProcMem(failingReaderAt{})
	if _, err := pm.ReadU64(0); err == nil {
		t.Fatal("expected error to propagate from underlying reader")
	}
}

func TestReadRegion_ReadsFullRegionBytes(t *testing.T) {
	data := bytes.Repeat([]byte{0xAB}, 64)
	region := Region{Start: 0, End: 64}

	buf, err := ReadRegion(bytes.NewReader(data), region)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buf) != 64 {
		t.Fatalf("expected 64 bytes, got %d", len(buf))
	}
	for i, b := range buf {
		if b != 0xAB {
			t.Fatalf("unexpected byte at %d: %#x", i, b)
		}
	}
}

func TestReadRegion_ReadsFromCorrectOffset(t *testing.T) {
	data := make([]byte, 32)
	copy(data[16:], []byte{1, 2, 3, 4})
	region := Region{Start: 16, End: 20}

	buf, err := ReadRegion(bytes.NewReader(data), region)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(buf, []byte{1, 2, 3, 4}) {
		t.Fatalf("unexpected bytes: %v", buf)
	}
}
