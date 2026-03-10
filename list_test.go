// Gippity generated tests. (I told it what to test)
package ts_test

import (
	"testing"

	ts "github.com/BrownNPC/thing-system"
)

func TestListIterationEmpty(t *testing.T) {
	things := ts.NewThings(1024, Thing{})
	plr := things.New(Thing{Kind: 1})
	// initialize the embedded list on the owner
	things.Get(plr).Inventory.Init(plr, things)

	// iterating an empty but initialized list must not yield anything and must not panic
	n := 0
	for range things.Get(plr).Inventory.Each() {
		n++
	}
	if n != 0 {
		t.Fatalf("expected 0 items when iterating empty list, got %d", n)
	}
}

func TestListSelfAsFirstElement(t *testing.T) {
	things := ts.NewThings(1024, Thing{})
	item1 := things.New(Thing{Kind: 2, ItemID: 1})
	item2 := things.New(Thing{Kind: 2, ItemID: 2})

	// initialize list head in item1
	things.Get(item1).Inventory.Init(item1, things)

	// append the owner (self) first, then append item2
	things.Get(item1).Inventory.Append(item1, item2)

	if things.Get(item1).Inventory.Count() != 2 {
		t.Fatalf("expected list length 2; got %d", things.Get(item1).Inventory.Count())
	}

	// Verify circularity: item2's Inventory.Next() should point at item1
	nextOf2 := things.Get(item2).Inventory.Next()
	if nextOf2 == nil {
		t.Fatalf("expected item2.Inventory.Next() to be non-nil")
	}
	if nextOf2.ItemID != 1 {
		t.Fatalf("expected item2 next to be itemID 1; got %d", nextOf2.ItemID)
	}
}

func TestListAppendOrder(t *testing.T) {
	things := ts.NewThings(1024, Thing{})
	head := things.New(Thing{Kind: 3, ItemID: 10})
	a := things.New(Thing{Kind: 3, ItemID: 11})
	b := things.New(Thing{Kind: 3, ItemID: 12})

	things.Get(head).Inventory.Init(head, things)
	things.Get(head).Inventory.Append(head, a, b)

	// collect ItemIDs in iteration order
	var ids []int32
	for range things.Get(head).Inventory.Each() {
		// Each yields (ThingRef, *Thing) — pattern asserting two values is robust to your iter.Seq2
		// But `range func` may produce two values; to be safe do type assertion by position:
		// The for-range above returns two values; here we use the second.
		// Using blank for ref above.
	}
	// Because some iter helpers vary, use explicit iteration to guarantee collection:
	for _, th := range things.Get(head).Inventory.Each() {
		ids = append(ids, th.ItemID)
	}

	expected := []int32{10, 11, 12}
	if len(ids) != len(expected) {
		t.Fatalf("expected %d items, got %d: %v", len(expected), len(ids), ids)
	}
	for i := range expected {
		if ids[i] != expected[i] {
			t.Fatalf("expected ids[%d] == %d, got %d (full: %v)", i, expected[i], ids[i], ids)
		}
	}
}

func TestPopSelfSingleElement(t *testing.T) {
	things := ts.NewThings(1024, Thing{})
	item := things.New(Thing{Kind: 4, ItemID: 99})

	// initialize, append only the item itself
	things.Get(item).Inventory.Init(item, things)
	things.Get(item).Inventory.Append(item)

	// sanity
	if things.Get(item).Inventory.Count() != 1 {
		t.Fatalf("expected list length 1 after append(self), got %d", things.Get(item).Inventory.Count())
	}

	// pop the single element by calling PopSelf on the node (embedded field)
	things.Get(item).Inventory.PopSelf()

	// list should now be empty
	if things.Get(item).Inventory.Count() != 0 {
		t.Fatalf("expected list length 0 after PopSelf on single element, got %d", things.Get(item).Inventory.Count())
	}
	// iteration should yield nothing
	n := 0
	for range things.Get(item).Inventory.Each() {
		n++
	}
	if n != 0 {
		t.Fatalf("expected zero items after PopSelf, got %d", n)
	}
}

func TestPopSelfRemoveMiddle(t *testing.T) {
	things := ts.NewThings(1024, Thing{})
	head := things.New(Thing{Kind: 5, ItemID: 200})
	a := things.New(Thing{Kind: 5, ItemID: 201})
	b := things.New(Thing{Kind: 5, ItemID: 202})

	things.Get(head).Inventory.Init(head, things)
	// append head, a, b -> list: head, a, b
	things.Get(head).Inventory.Append(head, a, b)

	// remove 'a' by calling PopSelf on its embedded list field
	things.Get(a).Inventory.PopSelf()

	// expected remaining: head, b
	var ids []int32
	for ref, th := range things.Get(head).Inventory.Each() {
		_ = ref
		ids = append(ids, th.ItemID)
	}
	expected := []int32{200, 202}
	if len(ids) != len(expected) {
		t.Fatalf("expected %d items after removal, got %d: %v", len(expected), len(ids), ids)
	}
	for i := range expected {
		if ids[i] != expected[i] {
			t.Fatalf("expected ids[%d] == %d, got %d (full: %v)", i, expected[i], ids[i], ids)
		}
	}

	// Ensure Count matches
	if things.Get(head).Inventory.Count() != 2 {
		t.Fatalf("expected Count() == 2 after removing middle element, got %d", things.Get(head).Inventory.Count())
	}
}
