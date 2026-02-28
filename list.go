package ts

import (
	"iter"
	"unsafe"
)

// List is supposed to be embedded inside of your Thing type.
// List is a circular intrusive linked list of Things.
// In layman's terms: it's a list that wraps around and tracks ThingRef's of the items in the list.
type List[Thing any] struct {
	things        *Things[Thing]
	isInitialized bool
	owner         ThingRef
	offset        uintptr // offset of this list within the Thing struct.

	// the first Thing in the list
	first ThingRef
	// the next Thing in the list. If we are the last thing in the list then next==first
	next ThingRef
	// the Thing before this Thing.
	// If this Thing is the first, then prev is the last Thing in the list.
	prev ThingRef
}

func (curr *List[Thing]) Append(things ...ThingRef) {
	for _, t := range things {
		curr.append(t)
	}
}

// Each iterates over the List and returns each ThingRef + Thing
func (curr *List[Thing]) Each() iter.Seq2[ThingRef, *Thing] {
	return func(yield func(ThingRef, *Thing) bool) {
		if !curr.isInitialized {
			if logger != nil {
				logger.Warn("Range over uninitialized list", "file", getParentCaller(0))
			}
			return
		}
		idx := 0
		current := curr.first
		for {
			if !yield(current, curr.things.get(current)) {
				return
			}
			current = curr.getListDataFromThing(current).next
			idx++
			if current == curr.first {
				break
			}
		}
	}
}

// PopSelf removes current Thing from List.
func (curr *List[Thing]) PopSelf() {
	if curr.owner == NilRef {
		if logger != nil {
			logger.Warn("Tried to Pop from uninitialized list", "file", getParentCaller(0))
		}
		return
	}
	if curr.first == NilRef {
		if logger != nil {
			logger.Warn("Tried to Pop from empty list", "file", getParentCaller(0))
		}
		return
	}

	next := curr.getListDataFromThing(curr.next)
	currRef := next.prev
	// single element case
	if curr.next == currRef {
		*curr = List[Thing]{}
		return
	}

	first := curr.getListDataFromThing(curr.first)
	prev := curr.getListDataFromThing(curr.prev)

	// pop current
	prev.next = curr.next
	next.prev = curr.prev

	// if removing first element, move head
	if currRef == curr.first {
		curr.first = curr.next
	}

	// if removing last element, update first.prev
	if currRef == first.prev {
		first.prev = curr.prev
	}
	// clear this node
	*curr = List[Thing]{}

}

// InsertNext inserts the Thing after the this Thing.
// It does not do anything if list is empty or uninitialized.
func (curr *List[Thing]) InsertNext(newThingRef ThingRef) {
	if newThingRef == NilRef {
		if logger != nil {
			logger.Warn("Tried to insert NilRef into list", "file", getParentCaller(0))
		}
	}
	if curr.owner == NilRef {
		if logger != nil {
			logger.Warn("Tried to Insert into uninitialized list", "file", getParentCaller(0))
		}
	}
	if curr.first == NilRef {
		if logger != nil {
			logger.Warn("Tried to Insert into empty list", "file", getParentCaller(0))
		}
	}

	newThing := curr.getListDataFromThing(newThingRef)
	// must be popped from list before Thing is deleted.
	curr.things.insideLists[newThingRef] = append(curr.things.insideLists[newThingRef], newThing)
	next := curr.getListDataFromThing(curr.next)
	currThingRef := next.prev
	// insert
	newThing.next = curr.next
	newThing.prev = currThingRef
	next.prev = newThingRef
	curr.next = newThingRef

	// check if inserted after last element
	first := curr.getListDataFromThing(curr.first)
	if curr.next == curr.first { // inserted at end, update circle
		first.prev = newThingRef
	}
}

// Count counts the number of elements in the List
func (curr *List[Thing]) Count() int {
	if !curr.isInitialized {
		if logger != nil {
			logger.Warn("Attempt to Count uninitialized list", "file", getParentCaller(0))
		}
		return 0
	}
	count := 0
	for range curr.Each() {
		count++
	}
	return count
}

// Init initializes the List. It must be called before adding Things.
func (curr *List[Thing]) Init(self ThingRef, things *Things[Thing]) {
	if self == NilRef {
		return
	}
	if curr.owner != NilRef {
		if logger != nil {
			logger.Error("Cannot Initialize a if Thing is already part of this list field. Add a new List field and initialize that instead.", "file", getParentCaller(0))
		}
		return
	}
	curr.isInitialized = true
	curr.owner = self
	curr.things = things
	owner := things.get(self)

	// compute and store the offset of this List field inside the owner struct
	curr.offset = uintptr(unsafe.Pointer(curr)) - uintptr(unsafe.Pointer(owner))

	// validate offset: the field must fit inside the owner object
	ownerSize := unsafe.Sizeof(*owner)
	listSize := unsafe.Sizeof(*curr)
	if curr.offset+listSize > ownerSize {
		*curr = List[Thing]{} // uninitialize
		if logger != nil {
			logger.Error("Incorrect owner ThingRef passed", "file", getParentCaller(0))
			return
		}
	}
}

// Owner returns the Owner of the list
func (curr *List[Thing]) Owner() ThingRef {
	if curr.owner == NilRef {
		if logger != nil {
			logger.Warn("Tried to get Owner of uninitialized list", "caller", getParentCaller(0))
		}
	}
	return curr.owner
}

// First returns the First thing inside the List.
// Thing is guaranteed to be not nil.
func (curr *List[Thing]) First() *Thing {
	if curr.owner == NilRef {
		if logger != nil {
			logger.Warn("Tried to get First Thing in uninitialized list", "caller", getParentCaller(0))
		}
	}
	return curr.things.get(curr.first)
}

// First returns the Previous thing inside the List.
// Thing is guaranteed to be not nil.
func (curr *List[Thing]) Prev() *Thing {
	if curr.owner == NilRef {
		if logger != nil {
			logger.Warn("Tried to get Previous Thing in uninitialized list", "caller", getParentCaller(0))
		}
	}
	return curr.things.get(curr.prev)
}

// First returns the Next thing inside the List.
// Thing is guaranteed to be not nil.
func (curr *List[Thing]) Next() *Thing {
	if curr.owner == NilRef {
		if logger != nil {
			logger.Warn("Tried to get Next Thing in uninitialized list", "caller", getParentCaller(0))
		}
	}
	return curr.things.get(curr.next)
}

// First returns the Last thing inside the List.
// Thing is guaranteed to be not nil.
func (curr *List[Thing]) Last() *Thing {
	if curr.owner == NilRef {
		if logger != nil {
			logger.Warn("Tried to get Last Thing in uninitialized list", "caller", getParentCaller(0))
		}
	}
	// first -> prev == last
	last := curr.getListDataFromThing(curr.first).prev
	return curr.things.get(last)
}

// get List from this Thing.
func (curr *List[Thing]) getListDataFromThing(thingRef ThingRef) *List[Thing] {
	thing := curr.things.get(thingRef)
	// add the stored offset to the Thing pointer to get pointer to the embedded List field
	fieldPtr := unsafe.Add(unsafe.Pointer(thing), curr.offset)
	return (*List[Thing])(fieldPtr)
}
func (curr *List[Thing]) append(newThingRef ThingRef) {
	if !curr.isInitialized || !curr.things.isNotNil(newThingRef) || !curr.things.isAlive(newThingRef) {
		logger.Warn("Append to uninitialized list", "file", getParentCaller(1))
		return
	}
	// must be popped from list before deletion

	newThing := curr.getListDataFromThing(newThingRef)
	curr.things.insideLists[newThingRef] = append(curr.things.insideLists[newThingRef], newThing)
	newThing.things = curr.things
	newThing.offset = curr.offset
	newThing.owner = curr.owner

	// if list was empty
	if curr.first == NilRef && curr.next == NilRef && curr.prev == NilRef {
		// The only thing in the list. Links to itself.
		newThing.first = newThingRef
		newThing.next = newThingRef
		newThing.prev = newThingRef

		// update Head of the list
		curr.first = newThingRef
		curr.next = newThingRef
		curr.prev = newThingRef
		return
	}

	// add NewThing between first and last
	firstThing := curr.getListDataFromThing(curr.first)
	// Last is just First->Prev
	last := firstThing.prev
	lastThing := curr.getListDataFromThing(last)

	// Last <-> New <-> First
	newThing.first = curr.first
	newThing.prev = last
	newThing.next = curr.first

	// update neighbors
	lastThing.next = newThingRef
	firstThing.prev = newThingRef
}
