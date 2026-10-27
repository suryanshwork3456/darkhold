package ws

import (
	"strings"
	"testing"
	"time"

	"darkhold/internal/types"
)

type stubSource struct{ ch chan types.Event }

func (s *stubSource) Events() <-chan types.Event { return s.ch }
func (s *stubSource) Snapshot(id string) types.BookSnapshot {
	return types.BookSnapshot{ItemID: id, Bids: []types.PriceLevel{{Price: 100, Quantity: 1}}}
}

func newTestClient() *Client { return &Client{send: make(chan []byte, 8)} }

func mustReceive(t *testing.T, c *Client) string {
	t.Helper()
	select {
	case msg := <-c.send:
		return string(msg)
	case <-time.After(time.Second):
		t.Fatal("expected a message but got none")
		return ""
	}
}

func mustNotReceive(t *testing.T, c *Client) {
	t.Helper()
	select {
	case msg := <-c.send:
		t.Fatalf("unexpected message: %s", msg)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestSubscribeAndBroadcast(t *testing.T) {
	src := &stubSource{ch: make(chan types.Event, 8)}
	h := NewHub(src)
	go h.Run()

	c1, c2 := newTestClient(), newTestClient()
	h.register <- c1
	h.register <- c2
	h.subscribe <- subscription{client: c1, itemID: "item-1"}
	h.subscribe <- subscription{client: c2, itemID: "item-2"}

	if msg := mustReceive(t, c1); !strings.Contains(msg, `"snapshot"`) {
		t.Fatalf("c1 should get a snapshot first, got %s", msg)
	}
	mustReceive(t, c2)

	snap := types.BookSnapshot{ItemID: "item-1"}
	src.ch <- types.Event{Type: eventBookUpdate, ItemID: "item-1", Snapshot: &snap}

	if msg := mustReceive(t, c1); !strings.Contains(msg, `"book_update"`) {
		t.Fatalf("c1 should get book_update, got %s", msg)
	}
	mustNotReceive(t, c2) // c2 watches item-2 only
}

func TestSlowClientIsDropped(t *testing.T) {
	src := &stubSource{ch: make(chan types.Event, 8)}
	h := NewHub(src)
	go h.Run()

	slow := &Client{send: make(chan []byte, 1)} // tiny buffer, nobody reads it
	h.register <- slow
	h.subscribe <- subscription{client: slow, itemID: "item-1"} // snapshot fills the buffer

	snap := types.BookSnapshot{ItemID: "item-1"}
	src.ch <- types.Event{Type: eventBookUpdate, ItemID: "item-1", Snapshot: &snap}

	// The hub should close the slow client's channel instead of blocking.
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-slow.send:
			if !ok {
				return // closed, as expected
			}
		case <-deadline:
			t.Fatal("slow client was not dropped")
		}
	}
}