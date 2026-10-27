package ws

import (
	"encoding/json"

	"darkhold/internal/types"
)

// Event type names. They must match what the engine emits in types.Event.Type.
const (
	eventBookUpdate = "book_update"
	eventTrade      = "trade"
	msgSnapshot     = "snapshot"
)

// ClientMessage is what the browser sends to us.
// Example: {"action":"subscribe","itemId":"item-1"}
type ClientMessage struct {
	Action string `json:"action"` // "subscribe" or "unsubscribe"
	ItemID string `json:"itemId"`
}

// level is a price level in lowercase JSON. Prices are in paise (int64).
type level struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
}

type bookMessage struct {
	Type   string  `json:"type"` // "snapshot" or "book_update"
	ItemID string  `json:"itemId"`
	Bids   []level `json:"bids"` // highest price first
	Asks   []level `json:"asks"` // lowest price first
}

type tradeMessage struct {
	Type     string `json:"type"` // "trade"
	ItemID   string `json:"itemId"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity"`
	At       int64  `json:"at"` // unix milliseconds
}

// toLevels always returns a non-nil slice, so JSON gives [] and never null.
func toLevels(in []types.PriceLevel) []level {
	out := make([]level, 0, len(in))
	for _, p := range in {
		out = append(out, level{Price: p.Price, Quantity: p.Quantity})
	}
	return out
}

// encodeBook builds a snapshot or book_update message.
func encodeBook(msgType string, snap types.BookSnapshot) []byte {
	data, _ := json.Marshal(bookMessage{
		Type:   msgType,
		ItemID: snap.ItemID,
		Bids:   toLevels(snap.Bids),
		Asks:   toLevels(snap.Asks),
	})
	return data
}

// encodeTrade builds a public trade message. Buyer and seller IDs are
// deliberately left out, because every watcher receives this.
func encodeTrade(t *types.Trade) []byte {
	data, _ := json.Marshal(tradeMessage{
		Type:     eventTrade,
		ItemID:   t.ItemID,
		Price:    t.Price,
		Quantity: t.Quantity,
		At:       t.At.UnixMilli(),
	})
	return data
}

// encodeEvent converts an engine event into bytes for browsers.
// The bool is false when the event should be ignored.
func encodeEvent(ev types.Event) ([]byte, bool) {
	switch ev.Type {
	case eventBookUpdate:
		if ev.Snapshot == nil {
			return nil, false
		}
		return encodeBook(eventBookUpdate, *ev.Snapshot), true
	case eventTrade:
		if ev.Trade == nil {
			return nil, false
		}
		return encodeTrade(ev.Trade), true
	}
	return nil, false
}