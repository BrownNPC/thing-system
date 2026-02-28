package ts_test

import (
	"testing"
	"time"

	ts "github.com/BrownNPC/thing-system"
)

func BenchmarkLoop10kThings(b *testing.B) {
	const thingCount = 10_000
	things := ts.NewThings(thingCount, Thing{})
	for range thingCount {
		things.New(Thing{})
	}

	for b.Loop() {
		for _, thing := range things.Each() {
			thing.Health -= 1
		}
	}
	b.Attr("PerElementCost", (b.Elapsed() / time.Duration(b.N) / time.Duration(thingCount)).String())
}

func BenchmarkLoop10kThingsList(b *testing.B) {
	const thingCount = 10_000
	things := ts.NewThings(thingCount+1, Thing{})
	plr := things.New(Thing{
		Kind: KindPlayer,
	})
	// initialize inventory List
	things.Get(plr).Inventory.Init(plr, things)

	for range thingCount {
		item := things.New(Thing{Kind: KindItem, ItemID: 67})
		things.Get(plr).Inventory.Append(item)
	}

	for b.Loop() {
		for _, thing := range things.Get(plr).Inventory.Each() {
			thing.ItemID += 1
		}
	}
	b.Attr("PerElementCost", (b.Elapsed() / time.Duration(b.N) / time.Duration(thingCount)).String())
}
