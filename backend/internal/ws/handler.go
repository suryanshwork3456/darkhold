package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// DEV ONLY: allows any origin. Restrict to your frontend's address before going live.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS returns the HTTP handler that upgrades a request to a WebSocket.
func ServeWS(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade failed: %v", err)
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, sendBufferSize),
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}