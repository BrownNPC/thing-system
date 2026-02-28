# Thing System
> A set of helpers to represent Things inside your application


### Features:
- Null Safety + Memory Safety
> Impossible to crash your program by accident
- Zero Allocations
> Things live inside 1 array, allocated upfront
- Performance
> Because zero allocations == "blazingly fast"
- Logs your mistakes
> If you mess something up, your program won't crash, but it'll log exactly where in the code you messed up
- Customizable Logs
> uses the stdlib slog.Logger, you can make it log to a file, to a discord server or disable logs entirely

### Everything in action:

```go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	ts "github.com/BrownNPC/thing-system"
	"go.abhg.dev/log/silog"
)

type Vector2 struct{ X, Y float64 }

type Kind int

const (
	KindNil = iota
	KindPlayer
	KindItem
)

// Thing is something inside our application. All the data every Thing will use lives here.
type Thing struct {
	Kind     Kind // What kind of thing is it?
	Health   int32
	ItemID   int32
	Position Vector2

	// List of Things (intrusive list)
	// Used to represent Things of KindItem inside Player's Inventory.
	Inventory ts.List[Thing]
	Blocks    [16 * 16]uint8
}

func main() {
	// Allocate 10k things in memory.
	// after this, we won't allocate any memory.
	things := ts.NewThings[Thing](10_000)

	// create a Thing to represent Player.
	// Guaranteed to never allocate memory!
	var Plr ts.ThingRef = things.New(Thing{
		Kind:     KindPlayer,
		Position: Vector2{20, 20},
	})

	// Get returns a pointer to Thing, it's GUARANTEED to never be nil.
	// Storing the pointer for later use is not safe. Always use ThingRef to access Things.
	things.Get(Plr).Health -= 1
	// initialize list
	things.Get(Plr).Inventory.Init(Plr, things)

	// Create a Thing that represents Items inside our Player's inventory
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
		Inventory.First(). // Helpers for iterating list. Also null-safe!
		Inventory.Next().Inventory.PopSelf()

	// Loop over all the things inside player inventory
	for ref, thing := range things.Get(Plr).Inventory.Each() {
		fmt.Printf("Inventory: Kind:%v ItemID:%v", thing.Kind, thing.ItemID)

		// Delete:
		// invalidates the Ref, so this Thing can be reused later.
		// Auto removes ref from Lists (like Plr.Inventory)
		// NOTE: Deleting while looping is unsafe.
		defer things.Del(ref) // defer to delete after the loop

		fmt.Printf("Queued %v for deletion. That means it's Active:%v as of now\n", ref.String(), things.IsActive(ref)) // IsActive=true
	}
	// scary!
	var thingThatDoesNotExist ts.ThingRef
	// DOES NOT CRASH! Only logs your mistake (customizable)
	things.Get(thingThatDoesNotExist).Inventory.Append(item1)
	// things.Get on an invalid Ref just returns a zero value Thing{}
	// LOGs:
	// 9:44AM WRN Derefence of NilRef.  file=/home/user/project/example.go:89
	// 9:44AM WRN Append to uninitialized list  file=/home/user/project/example.go:89

	// But I want to disable logs >:)
	ts.SetLogger(nil) // easy!

	// No wait, I wanna log to a file!
	{
		f, _ := os.Create(time.Now().Format(time.DateTime) + "-log.txt") // create file
		defer f.Close()                                                  // save it
		// create slog.Handler
		handler := silog.NewHandler(f, nil) // same handler that's used internally.
		ts.SetLogger(slog.New(handler))     // logs to the file instead.
	}

	// Actually, I wanna log to a JSON file now
	{
		f, _ := os.Create(time.Now().Format(time.DateTime) + "-log.json") // create JSON file
		defer f.Close()                                                   // save it
		// JSON handler :D
		handler := slog.NewJSONHandler(f, nil)
		ts.SetLogger(slog.New(handler)) // logs to the JSON file instead.
	}

}
```
