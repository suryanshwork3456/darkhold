package ws

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"darkhold/internal/types"
)

// FakeSource pretends to be the matching engine so the WebSocket layer
// can be built and tested alone. It satisfies EventSource.
type FakeSource struct {
	mu     sync.Mutex
	mids   map[string]int64 // current middle price per item, in paise
	events chan types.Event
	seq    int
}

func NewFakeSource(itemIDs []string) *FakeSource {
	f := &FakeSource{
		mids:   make(map[string]int64),
		events: make(chan types.Event, 64),
	}
	for i, id := range itemIDs {
		f.mids[id] = int64(100000 + i*50000) // Rs 1000, 1500, 2000...
	}
	go f.run()
	return f
}

func (f *FakeSource) Events() <-chan types.Event { return f.events }

func (f *FakeSource) Snapshot(itemID string) types.BookSnapshot {
	f.mu.Lock()
	mid, ok := f.mids[itemID]
	f.mu.Unlock()
	if !ok {
		return types.BookSnapshot{ItemID: itemID, Bids: []types.PriceLevel{}, Asks: []types.PriceLevel{}}
	}
	return buildBook(itemID, mid)
}

func (f *FakeSource) run() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		f.mu.Lock()
		ids := make([]string, 0, len(f.mids))
		for id := range f.mids {
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			f.mu.Unlock()
			continue
		}
		id := ids[rand.Intn(len(ids))]
		f.mids[id] += int64(rand.Intn(2001) - 1000) // random walk +-Rs 10
		if f.mids[id] < 10000 {
			f.mids[id] = 10000
		}
		mid := f.mids[id]
		f.mu.Unlock()

		var ev types.Event
		if rand.Intn(10) < 3 { // 30% trades, 70% book updates
			f.seq++
			ev = types.Event{
				Type:   eventTrade,
				ItemID: id,
				Trade: &types.Trade{
					ID:       fmt.Sprintf("fake-trade-%d", f.seq),
					ItemID:   id,
					Price:    mid,
					Quantity: int64(rand.Intn(5) + 1),
					At:       time.Now(),
				},
			}
		} else {
			snap := buildBook(id, mid)
			ev = types.Event{Type: eventBookUpdate, ItemID: id, Snapshot: &snap}
		}

		select {
		case f.events <- ev:
		default: // buffer full, drop like the real bus does
		}
	}
}

// buildBook makes 5 bid levels and 5 ask levels around the mid price.
func buildBook(itemID string, mid int64) types.BookSnapshot {
	snap := types.BookSnapshot{ItemID: itemID, Bids: []types.PriceLevel{}, Asks: []types.PriceLevel{}}
	for i := int64(0); i < 5; i++ {
		snap.Bids = append(snap.Bids, types.PriceLevel{Price: mid - 500 - i*500, Quantity: rand.Int63n(20) + 1}) // highest first
		snap.Asks = append(snap.Asks, types.PriceLevel{Price: mid + 500 + i*500, Quantity: rand.Int63n(20) + 1}) // lowest first
	}
	return snap
}