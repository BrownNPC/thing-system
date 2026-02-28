package ts

import (
	"fmt"
	"iter"
	"log/slog"
	"os"
	"runtime"

	"github.com/lmittmann/tint"
)

var logger *slog.Logger = slog.New(tint.NewHandler(os.Stderr, nil))

type ThingRef struct {
	idx, generation uint32
}

func (ref ThingRef) String() string {
	if ref == NilRef {
		return "ThingRef(NIL)"
	}
	return fmt.Sprintf("Thing(%v.%v)", ref.idx, ref.generation)
}

var NilRef = ThingRef{}

// Things is responsible for the creation, deletion, and reuse of a Thing.
type Things[Thing any] struct {
	maxThings   uint32
	activeThings uint //number of things that are active
	things      []Thing // index 0 is nil (zero)
	used        []bool
	generations []uint32
	insideLists map[ThingRef][]*List[Thing]
}

// NewThings allocates memory for all the Things upfront. It's also responsible for the creation, deletion, and reuse of a Thing
// Things will not log anything if logger is nil.
func NewThings[Thing any](maxThings uint,_...Thing) *Things[Thing] {
	maxThings += 1 // thing on index 0 is nil.
	return &Things[Thing]{
		maxThings:   uint32(maxThings),
		things:      make([]Thing, maxThings),
		used:        make([]bool, maxThings),
		generations: make([]uint32, maxThings),
		insideLists: make(map[ThingRef][]*List[Thing]),
	}
}

// New creates a new Thing and returns the ThingRef
func (things *Things[Thing]) New(thing Thing) ThingRef {
	ref := things.findEmpty()
	things.used[ref.idx] = true
	things.things[ref.idx] = thing
	things.activeThings++
	return ref
}

// Del marks the Thing available for reuse.
func (things *Things[Thing]) Del(ref ThingRef) {
	if things.IsActive(ref) {
		for _, list := range things.insideLists[ref] {
			// if not already popped
			if (*list != List[Thing]{}) {
				list.PopSelf()
			}
		}
		delete(things.insideLists, ref)

		things.used[ref.idx] = false
		things.generations[ref.idx] += 1
		// zero it out (set to nil)
		things.things[ref.idx] = things.things[0]
		things.activeThings--
	} else {
		if logger != nil {
			logger.Warn("Tried to Delete inactive Thing", "file", getParentCaller(0))
		}
	}
}

// Get returns a pointer to the Thing behind the ThingRef.
// It is guaranteed to never be nil.
// You should NEVER store the pointer returned by Get for safety reasons.
// It's recommended to call Get every time you want to modify a field.
//
//	✅ Correct
//	things.Get(Plr).Health -= 1
//	things.Get(Plr).Invincible = true // Safe
//
//	✅ Safe
//	player := things.Get(Plr) // Not recommended but safe
//	player.Health -= 1
//
//	❌ Unsafe
//	someGlobalVariable.player = things.Get(Plr) // reference stored for later use
//	someGlobalVariable.player.Health -= 1 // Unsafe
func (things *Things[Thing]) Get(ref ThingRef) *Thing {
	if things.isNotNil(ref) && things.isAlive(ref) {
		return &things.things[ref.idx]
	}
	if logger != nil {
		logger.Warn("Derefence of NilRef.", "file", getParentCaller(0))
	}
	var z Thing
	return &z
}

// get is the same as Get but does not trigger a log.
func (things *Things[Thing]) get(ref ThingRef) *Thing {
	if things.isNotNil(ref) && things.isAlive(ref) {
		return &things.things[ref.idx]
	}
	var z Thing
	return &z
}

// Each iterates over all the active Things.
//
// The pointers should not be stored, only modified.
func (things *Things[Thing]) Each() iter.Seq2[ThingRef, *Thing] {
	return func(yield func(ThingRef, *Thing) bool) {
		remaining := things.activeThings
		for id, used := range things.used {
			if remaining==0{
				break
			}
			if used {
	   			remaining--
				if !yield(
					ThingRef{idx: uint32(id),
						generation: things.generations[id]},
					&things.things[id]) {
					break
				}
			}
		}
	}
}

// SetLogger sets the logger used for warnings.
// Passing nil disables the logger.
func SetLogger(log *slog.Logger) {
	logger = log
}

// IsActive returns true if ref is in use.
func (things *Things[Thing]) IsActive(ref ThingRef) bool {
	return things.isNotNil(ref) && things.isAlive(ref)
}

// isNotNil checks if the ref is a NilRef, or out of bounds.
func (things *Things[Thing]) isNotNil(ref ThingRef) bool {
	if ref.idx > 0 && ref.idx < things.maxThings {
		return true
	}
	return false
}

// findEmpty finds an unsed slot.
func (things *Things[Thing]) findEmpty() ThingRef {
	for i := 1; i < len(things.used); i++ {
		if !things.used[i] {
			return ThingRef{uint32(i), things.generations[i]}
		}
	}
	if logger != nil {
		logger.Error("Ran out of memory, allocate more things in NewThings()", "file", getParentCaller(1))
	}
	return NilRef
}

// isAlive checks if a ref is in use and the generation is not old.
func (things *Things[Thing]) isAlive(ref ThingRef) bool {
	dead := things.used[ref.idx] == false || ref.generation != things.generations[ref.idx]
	return !dead
}

// used for loggin
func getParentCaller(skip int) string {
	_, file, line, _ := runtime.Caller(2 + skip)
	return fmt.Sprintf("%v:%v", file, line)
}
