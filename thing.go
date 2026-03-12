package ts

import (
	"fmt"
	"iter"
	"log/slog"
	"os"
	"runtime"
	"sync"

	"github.com/lmittmann/tint"
)

var logger *slog.Logger = slog.New(tint.NewHandler(os.Stderr, nil))

type ThingRef struct {
	idx, generation uint32
}

func (ref ThingRef) String() string {
	if ref == nilRef {
		return "ThingRef(NIL)"
	}
	return fmt.Sprintf("Thing(%v.%v)", ref.idx, ref.generation)
}

var nilRef = ThingRef{}

// Things is responsible for the creation, deletion, and reuse of a Thing.
// // nil thing will be defaultStateOptional[0]
type Things[Thing any] struct {
	maxThings    uint32
	activeThings uint    //number of things that are active
	things       []Thing // index 0 is nil (zero)
	used         []bool
	generations  []uint32
	insideLists  map[ThingRef][]*List[Thing]

	// []*Thing
	thingPointerPool sync.Pool
}

// NewThings allocates memory for all the Things upfront. It's also responsible for the creation, deletion, and reuse of a Thing.
// Things will not log anything if logger is nil.
//
// NewThings also supports setting the state for the Nil Thing.
// This could be useful for adding a "missing texture" for when you dereference a Nil ThingRef.
// 
// Usage: 
//	ts.Newthings[Thing](1024)
//  OR
//	ts.Newthings(1024, Thing{}) // same as above
//	
//	Set default state for Nil things.
//	ts.Newthings(1024, Thing{Texture: MyTextureForNilThings})
func NewThings[Thing any](maxThings uint, nilThingState_OPTIONAL ...Thing) *Things[Thing] {
	maxThings += 1 // thing on index 0 is nil.
	things :=&Things[Thing]{
		maxThings:   uint32(maxThings),
		things:      make([]Thing, maxThings),
		used:        make([]bool, maxThings),
		generations: make([]uint32, maxThings),
		insideLists: make(map[ThingRef][]*List[Thing]),
	}
	// nil thing will be defaultStateOptional[0]
	if len(nilThingState_OPTIONAL)>1{
		things.things[0] = nilThingState_OPTIONAL[0]
	}
	return things
}

// New creates a new Thing and returns the ThingRef
func (things *Things[Thing]) New(thing Thing) ThingRef {
	ref := things.findEmpty()
	if ref != nilRef {
		things.used[ref.idx] = true
		things.things[ref.idx] = thing
		things.activeThings++
	}
	return ref
}

// Delete marks the Thing available for reuse.
// Does not do anything if ref is Nil.
func (things *Things[Thing]) Delete(ref ...ThingRef) {
	for _,ref  := range ref {
		things.del(ref)
	}
}

func (things *Things[Thing]) del(ref ThingRef) {
	if things.IsNotNil(ref) {
		for _, list := range things.insideLists[ref] {
			// if not already popped
			if (*list != List[Thing]{}) {
				list.PopSelf()
			}
		}
		// free map memory
		delete(things.insideLists, ref)

		things.used[ref.idx] = false
		things.generations[ref.idx] += 1
		// zero it out  = things.things[0](set to nil)
		things.things[ref.idx] = things.things[0]
		things.activeThings--
	} else {
		if logger != nil {
			logger.Warn("Tried to Delete inactive Thing", "file", getParentCaller(0))
		}
	}
}

// Get  =t hings.things[0]returns a pointer to the Thing behind the ThingRef.
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
	if things.IsNotNil(ref){
		return &things.things[ref.idx]
	}
	if logger != nil {
		logger.Warn("Derefence of NilRef.", "file", getParentCaller(0))
	}
	var z Thing = things.things[0]
	return &z
}

// get is the same as Get but does not trigger a log.
func (things *Things[Thing]) get(ref ThingRef) *Thing {
	if things.isInBounds(ref) && things.isAlive(ref) {
		return &things.things[ref.idx]
	}
	var z Thing = things.things[0]
	return &z
}

// Each iterates over all the active Things.
//
// The pointers should not be stored, only modified.
func (things *Things[Thing]) Each() iter.Seq2[ThingRef, *Thing] {
	return func(yield func(ThingRef, *Thing) bool) {
		for id := uint(1); id <=things.activeThings; id++{
			if things.used[id]{
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



// Filter takes in filterFunc.
// For every thing the filterFunc returns true, it will be collected
// into the returned slice.
//
// It uses sync.Pool to avoid allocating every frame.
func (things *Things[Thing]) Filter(filterFunc func(t *Thing)bool) []*Thing{
	var collection []*Thing

	// Get collection from Pool.
	if coll := things.thingPointerPool.Get(); coll != nil {
		collection = coll.([]*Thing)
	}else{
		collection = make([]*Thing, 0, things.activeThings)
	}
	defer things.thingPointerPool.Put(collection)
	

    // filter things
	for _, thing := range things.Each() {
		if filterFunc(thing) {
			collection = append(collection, thing)
		}
	}
	return collection
}

// SetLogger sets the logger used for warnings.
// Passing nil disables the logger.
func SetLogger(log *slog.Logger) {
	logger = log
}

// IsNotNil returns true if ref is in use.
func (things *Things[Thing]) IsNotNil(ref ThingRef) bool {
	return things.isInBounds(ref) && things.isAlive(ref)
}

// isInBounds checks if the ref is a NilRef, or out of bounds.
func (things *Things[Thing]) isInBounds(ref ThingRef) bool {
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
	return nilRef
}

// isAlive checks if a ref is in use and the generation is not old.
func (things *Things[Thing]) isAlive(ref ThingRef) bool {
	dead := things.used[ref.idx] == false || ref.generation != things.generations[ref.idx]
	return !dead
}

// Returns line number of the function that called current function. useful for logging.
func GetParentCaller() string {
	return getParentCaller(1)
}

// used for loggin
func getParentCaller(skip int) string {
	_, file, line, _ := runtime.Caller(2 + skip)
	return fmt.Sprintf("%v:%v", file, line)
}
