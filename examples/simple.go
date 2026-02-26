package main

import (
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

	Inventory ts.List[Thing]
	Blocks    [16 * 16]uint8
}

func main() {

	things := ts.NewThings[Thing](10_000)
	Plr := things.New(Thing{
		Kind:     KindPlayer,
		Position: Vector2{20, 20},
	})

	things.Get(Plr).Health -= 1
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
	things.Get(Plr).Inventory.Append(item1, item2, item3)
}
