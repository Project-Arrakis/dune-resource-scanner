package memscan

import (
	"fmt"
	"math"
	"testing"
)

// fakeMem is an in-memory MemReader test double: address -> raw 8-byte value.
type fakeMem map[uint64]uint64

func (m fakeMem) ReadU64(addr uint64) (uint64, error) {
	v, ok := m[addr]
	if !ok {
		return 0, fmt.Errorf("no data at %#x", addr)
	}
	return v, nil
}

func (m fakeMem) ReadF64(addr uint64) (float64, error) {
	v, err := m.ReadU64(addr)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(v), nil
}

func (m fakeMem) ReadI32(addr uint64) (int32, error) {
	v, err := m.ReadU64(addr)
	if err != nil {
		return 0, err
	}
	return int32(uint32(v)), nil
}

const (
	testActorAddr     = uint64(0x100000)
	testClassPrivate  = uint64(0x710000)
	testRootComponent = uint64(0x720000)
)

func testExe(addr uint64) bool  { return addr >= 0x400000 && addr < 0x500000 }
func testHeap(addr uint64) bool { return addr >= 0x700000 && addr < 0x800000 }

// newValidActorMem builds a fakeMem representing one real, valid actor at
// testActorAddr, matching the vtable/ClassPrivate/RootComponent/Transform
// chain ValidateActor is expected to walk.
func newValidActorMem() fakeMem {
	m := fakeMem{}
	m[testActorAddr] = 0x450000                                       // vtable -> exe
	m[testActorAddr+DefaultOffsets.ClassPrivate] = testClassPrivate   // -> heap
	m[testClassPrivate] = 0x460000                                    // class's own vtable -> exe
	m[testActorAddr+DefaultOffsets.RootComponent] = testRootComponent // -> heap
	m[testRootComponent+DefaultOffsets.Transform] = math.Float64bits(-611736.35)
	m[testRootComponent+DefaultOffsets.Transform+8] = math.Float64bits(-700183.46)
	m[testRootComponent+DefaultOffsets.Transform+16] = math.Float64bits(500.0)
	m[testActorAddr+DefaultOffsets.BaseValue] = uint64(uint32(5000))
	return m
}

func TestValidateActor_AcceptsRealActor(t *testing.T) {
	mem := newValidActorMem()
	info, ok := ValidateActor(mem, testActorAddr, DefaultOffsets, testExe, testHeap)
	if !ok {
		t.Fatal("expected a well-formed actor to validate")
	}
	if info.Addr != testActorAddr {
		t.Errorf("unexpected Addr: %#x", info.Addr)
	}
	if info.X != -611736.35 || info.Y != -700183.46 || info.Z != 500.0 {
		t.Errorf("unexpected position: %+v", info)
	}
	if info.BaseValue != 5000 {
		t.Errorf("unexpected BaseValue: %d", info.BaseValue)
	}
}

func TestValidateActor_RejectsVtableOutsideExe(t *testing.T) {
	mem := newValidActorMem()
	mem[testActorAddr] = 0x999999 // not in exe range
	if _, ok := ValidateActor(mem, testActorAddr, DefaultOffsets, testExe, testHeap); ok {
		t.Fatal("expected rejection: vtable pointer not in executable range")
	}
}

func TestValidateActor_RejectsClassPrivateOutsideHeap(t *testing.T) {
	mem := newValidActorMem()
	mem[testActorAddr+DefaultOffsets.ClassPrivate] = 0x999999
	if _, ok := ValidateActor(mem, testActorAddr, DefaultOffsets, testExe, testHeap); ok {
		t.Fatal("expected rejection: ClassPrivate pointer not in heap range")
	}
}

func TestValidateActor_RejectsClassVtableOutsideExe(t *testing.T) {
	mem := newValidActorMem()
	mem[testClassPrivate] = 0x999999
	if _, ok := ValidateActor(mem, testActorAddr, DefaultOffsets, testExe, testHeap); ok {
		t.Fatal("expected rejection: class's own vtable not in executable range")
	}
}

func TestValidateActor_RejectsRootComponentOutsideHeap(t *testing.T) {
	mem := newValidActorMem()
	mem[testActorAddr+DefaultOffsets.RootComponent] = 0x999999
	if _, ok := ValidateActor(mem, testActorAddr, DefaultOffsets, testExe, testHeap); ok {
		t.Fatal("expected rejection: RootComponent pointer not in heap range")
	}
}

func TestValidateActor_RejectsInvalidTransform(t *testing.T) {
	mem := newValidActorMem()
	mem[testRootComponent+DefaultOffsets.Transform] = math.Float64bits(0)
	mem[testRootComponent+DefaultOffsets.Transform+8] = math.Float64bits(0)
	mem[testRootComponent+DefaultOffsets.Transform+16] = math.Float64bits(0)
	if _, ok := ValidateActor(mem, testActorAddr, DefaultOffsets, testExe, testHeap); ok {
		t.Fatal("expected rejection: exact-origin transform")
	}
}

func TestValidateActor_RejectsUnmappedRead(t *testing.T) {
	mem := newValidActorMem()
	delete(mem, testActorAddr)
	if _, ok := ValidateActor(mem, testActorAddr, DefaultOffsets, testExe, testHeap); ok {
		t.Fatal("expected rejection: unmapped/unreadable address")
	}
}
