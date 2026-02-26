package ts_test

import (
	"fmt"

	ts "github.com/BrownNPC/thing-system"
)

type Vector2 struct{ X, Y float64 }

type Kind int

const (
	KindNil = iota
	KindPlayer
	KindItem
)

type Thing struct {
	Kind     Kind
	Health   int32
	ItemID   int32
	Position Vector2

	// List of Things (intrusive list)
	Inventory ts.List[Thing]
	Blocks    [16 * 16]uint8
}

func Example() {
	// Allocate 10k things in memory.
	things := ts.NewThings[Thing](10_000)

	var Plr ts.ThingRef = things.New(Thing{
		Kind:     KindPlayer,
		Position: Vector2{20, 20},
	})
	// Storing result from Things.Get() is safe but not recommended.
	things.Get(Plr).Health -= 1
	// initialize list
	things.Get(Plr).Inventory.Init(Plr, things)

	item1 := things.New(Thing{
		Kind:   KindItem,
		ItemID: 1,
	})
	item2 := things.New(Thing{
		Kind:   KindItem,
		ItemID: 2,
	})
	item3 := things.New(Thing{
		Kind:   KindItem,
		ItemID: 3,
	})

	// Append items to Player inventory
	things.Get(Plr).Inventory.Append(item1, item2, item3)

	fmt.Println("Items in player inventory:", things.Get(Plr).Inventory.Count()) // 3

	// remove 2nd Item from Inventory
	things.Get(Plr).
		Inventory.First().
		Inventory.Next().Inventory.PopSelf()

	// Loop over all the things inside player inventory
	for ref, thing := range things.Get(Plr).Inventory.Each() {
		fmt.Printf("Inventory: Kind:%v ItemID:%v", thing.Kind, thing.ItemID)

		things.Del(ref)                                                              // delete thing marks for reuse, and ref becomes useless
		fmt.Printf("Deleted ref:%v IsActive:%v", ref.String(), things.IsActive(ref)) // false
	}
}
