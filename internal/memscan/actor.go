package memscan

// MemReader reads scalar values from a process's virtual address space.
type MemReader interface {
	ReadU64(addr uint64) (uint64, error)
	ReadF64(addr uint64) (float64, error)
	ReadI32(addr uint64) (int32, error)
}

// Offsets are the empirically re-derived byte offsets used to validate and
// read an actor's shape in memory.
type Offsets struct {
	ClassPrivate  uint64 // UObject-level: vtable-adjacent ClassPrivate pointer
	RootComponent uint64 // AActor-level: root scene component pointer
	Transform     uint64 // offset within the root component: 3 consecutive float64s (X/Y/Z)
	BaseValue     uint64 // offset within the actor: a "full/base" numeric field
}

// DefaultOffsets are the offsets confirmed against the live game process.
var DefaultOffsets = Offsets{
	ClassPrivate:  16,
	RootComponent: 576,
	Transform:     384,
	BaseValue:     1440,
}

// ActorInfo is the result of a successfully validated actor.
type ActorInfo struct {
	Addr      uint64
	X, Y, Z   float64
	BaseValue int32
}

// ValidateActor checks whether addr points to a real actor matching the
// expected UObject/AActor memory shape: a vtable pointer into the
// executable's mapped range, a ClassPrivate pointer into the heap whose own
// vtable is also in the executable range, a RootComponent pointer into the
// heap, and a plausible world-position Transform. isExe and isHeap report
// whether an address falls in the executable's or the heap's mapped ranges,
// respectively.
func ValidateActor(mem MemReader, addr uint64, off Offsets, isExe, isHeap func(uint64) bool) (ActorInfo, bool) {
	vtable, err := mem.ReadU64(addr)
	if err != nil || !isExe(vtable) {
		return ActorInfo{}, false
	}

	classPrivate, err := mem.ReadU64(addr + off.ClassPrivate)
	if err != nil || !isHeap(classPrivate) {
		return ActorInfo{}, false
	}

	classVtable, err := mem.ReadU64(classPrivate)
	if err != nil || !isExe(classVtable) {
		return ActorInfo{}, false
	}

	rootComponent, err := mem.ReadU64(addr + off.RootComponent)
	if err != nil || !isHeap(rootComponent) {
		return ActorInfo{}, false
	}

	x, err := mem.ReadF64(rootComponent + off.Transform)
	if err != nil {
		return ActorInfo{}, false
	}
	y, err := mem.ReadF64(rootComponent + off.Transform + 8)
	if err != nil {
		return ActorInfo{}, false
	}
	z, err := mem.ReadF64(rootComponent + off.Transform + 16)
	if err != nil {
		return ActorInfo{}, false
	}
	if !ValidTransform(x, y, z) {
		return ActorInfo{}, false
	}

	baseValue, err := mem.ReadI32(addr + off.BaseValue)
	if err != nil {
		return ActorInfo{}, false
	}

	return ActorInfo{Addr: addr, X: x, Y: y, Z: z, BaseValue: baseValue}, true
}
