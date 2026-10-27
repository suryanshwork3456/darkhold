package ws

import "darkhold/internal/types"

// EventSource is anything that can feed the hub: the real engine or the fake.
type EventSource interface {
	Events() <-chan types.Event
	Snapshot(itemID string) types.BookSnapshot
}

type subscription struct {
	client *Client
	itemID string
}

// Hub tracks who is connected and who watches which item.
// Only the goroutine running Run() touches clients and subs, so no mutex is needed.
type Hub struct {
	source      EventSource
	clients     map[*Client]bool
	subs        map[string]map[*Client]bool // itemID -> watchers
	register    chan *Client
	unregister  chan *Client
	subscribe   chan subscription
	unsubscribe chan subscription
}

func NewHub(source EventSource) *Hub {
	return &Hub{
		source:      source,
		clients:     make(map[*Client]bool),
		subs:        make(map[string]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		subscribe:   make(chan subscription),
		unsubscribe: make(chan subscription),
	}
}

// Run is the hub's main loop. Start it with: go hub.Run()
func (h *Hub) Run() {
	events := h.source.Events()
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true

		case c := <-h.unregister:
			h.removeClient(c)

		case s := <-h.subscribe:
			h.addSubscription(s)

		case s := <-h.unsubscribe:
			if watchers, ok := h.subs[s.itemID]; ok {
				delete(watchers, s.client)
				if len(watchers) == 0 {
					delete(h.subs, s.itemID)
				}
			}

		case ev, ok := <-events:
			if !ok {
				events = nil // source closed; a nil channel blocks forever, so skip it
				continue
			}
			if msg, ok := encodeEvent(ev); ok {
				h.broadcast(ev.ItemID, msg)
			}
		}
	}
}

func (h *Hub) addSubscription(s subscription) {
	if !h.clients[s.client] {
		return // not registered (or already removed)
	}
	if h.subs[s.itemID] == nil {
		h.subs[s.itemID] = make(map[*Client]bool)
	}
	h.subs[s.itemID][s.client] = true

	// Give the new watcher the current book right away.
	snap := h.source.Snapshot(s.itemID)
	snap.ItemID = s.itemID
	h.sendTo(s.client, encodeBook(msgSnapshot, snap))
}

func (h *Hub) broadcast(itemID string, msg []byte) {
	for c := range h.subs[itemID] {
		h.sendTo(c, msg)
	}
}

// sendTo never blocks. If the client's buffer is full it is too slow, so drop it.
func (h *Hub) sendTo(c *Client, msg []byte) {
	select {
	case c.send <- msg:
	default:
		h.removeClient(c)
	}
}

// removeClient is safe to call twice. Only the hub closes c.send, and only once.
func (h *Hub) removeClient(c *Client) {
	if !h.clients[c] {
		return
	}
	delete(h.clients, c)
	for itemID, watchers := range h.subs {
		delete(watchers, c)
		if len(watchers) == 0 {
			delete(h.subs, itemID)
		}
	}
	close(c.send)
}