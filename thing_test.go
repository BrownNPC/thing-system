package ts

import (
	"testing"
)
func init(){
	SetLogger(nil)
}

func TestNewThingsBasicLayout(t *testing.T) {
	th := NewThings[int](3)

	if th == nil {
		t.Fatal("NewThings returned nil")
	}

	// NewThings adds 1 for nil slot
	if int(th.maxThings) != 4 {
		t.Fatalf("expected maxThings 4, got %v", th.maxThings)
	}
	if len(th.things) != 4 {
		t.Fatalf("expected things length 4, got %v", len(th.things))
	}
	if len(th.used) != 4 {
		t.Fatalf("expected used length 4, got %v", len(th.used))
	}
}

func TestNewGetDeleteAndGeneration(t *testing.T) {
	th := NewThings[int](3)

	// create a thing
	ref := th.New(42)
	if ref == nilRef {
		t.Fatal("New returned nilRef unexpectedly")
	}
	if !th.IsNotNil(ref) {
		t.Fatalf("IsNotNil should be true for new ref %v", ref)
	}

	// Get should return pointer to stored value
	ptr := th.Get(ref)
	if ptr == nil {
		t.Fatal("Get returned nil pointer")
	}
	if *ptr != 42 {
		t.Fatalf("expected value 42, got %v", *ptr)
	}

	// Delete it
	oldGeneration := ref.generation
	th.Delete(ref)

	if th.IsNotNil(ref) {
		t.Fatalf("IsNotNil should be false after Delete for ref %v", ref)
	}
	// After deletion generation at index should have incremented
	if th.generations[ref.idx] == oldGeneration {
		t.Fatalf("expected generation to change after delete (was %v, now %v)", oldGeneration, th.generations[ref.idx])
	}

	// Old ref should not be alive
	if th.isAlive(ref) {
		t.Fatalf("old ref should not be alive after delete: %v", ref)
	}

	// Creating a new thing should eventually reuse an index (but with new generation)
	ref2 := th.New(99)
	if ref2 == nilRef {
		t.Fatal("New returned nilRef unexpectedly when capacity should be available")
	}
	if ref2.idx != ref.idx && ref2.idx >= th.maxThings {
		t.Fatalf("unexpected index for reused thing: %v", ref2)
	}
	if th.isAlive(ref2) == false {
		t.Fatalf("new ref should be alive: %v", ref2)
	}
}

func TestNewExhaustion(t *testing.T) {
	// Only one usable slot (max 1 -> internal size 2)
	th := NewThings[int](1)
	r1 := th.New(1)
	if r1 == nilRef {
		t.Fatal("first New returned nilRef")
	}
	// second New should return nilRef because capacity exhausted
	r2 := th.New(2)
	if r2 != nilRef {
		t.Fatalf("expected nilRef when out of capacity, got %v", r2)
	}
}

func TestIsNotNilAndBounds(t *testing.T) {
	th := NewThings[int](1)

	// zero value equals nilRef
	if th.IsNotNil(nilRef) {
		t.Fatal("nilRef should be considered nil")
	}

	// out of bounds ref
	out := ThingRef{idx: th.maxThings + 1, generation: 0}
	if th.IsNotNil(out) {
		t.Fatalf("out of bounds ref should be nil: %v", out)
	}
}

func TestGetParentCaller(t *testing.T) {
	call := GetParentCaller()
	if call == "" {
		t.Fatal("GetParentCaller returned empty string")
	}
	// Typically returns "file:line" format. Just ensure contains colon.
	foundColon := false
	for _, ch := range call {
		if ch == ':' {
			foundColon = true
			break
		}
	}
	if !foundColon {
		t.Fatalf("GetParentCaller returned unexpected format: %q", call)
	}
}

func TestEachAndFilterBehaviorIntended(t *testing.T) {
	// These tests express the intended semantics: Each should iterate over all active things,
	// and Filter should collect matching things. If the implementation has bugs, these tests will fail.
	th := NewThings[int](3)

	refs := make([]ThingRef, 0, 3)
	for i := 1; i <= 3; i++ {
		r := th.New(i)
		if r == nilRef {
			t.Fatalf("unexpected nilRef when adding item %d", i)
		}
		refs = append(refs, r)
	}

	// Count items yielded by Each
	count := 0
	for _, p := range th.Each() {
		if p == nil {
			t.Fatalf("Each yielded nil pointer")
		}
		count++
	}
	if count != 3 {
		t.Fatalf("Each: expected to iterate 3 items, got %d (implementation may be faulty)", count)
	}

	// Filter: select items with Value%2==1
	col := th.Filter(func(s *int) bool {
		return *s%2 == 1
	})
	if len(col) != 2 {
		t.Fatalf("Filter: expected 2 odd items, got %d", len(col))
	}
	// check values
	got := make(map[int]bool)
	for _, p := range col {
		got[*p] = true
	}
	if !got[1] || !got[3] {
		t.Fatalf("Filter returned unexpected items: %v", got)
	}
}
