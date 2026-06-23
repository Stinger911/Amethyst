// Package notify implements the WebSocket live-update from
// plan_amethyst-web-ui §5: when the file watcher reindexes a genuinely
// external change (one it didn't suppress for the API's own writes — see
// internal/watch's Watcher.Suppress), every connected browser tab is told
// which note path changed, so a note open for viewing can offer to reload.
package notify

import (
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Event is broadcast as JSON to every connected client.
type Event struct {
	Path string `json:"path"`
}

// Hub fans out Broadcast calls to every currently-connected client.
type Hub struct {
	register   chan chan Event
	unregister chan chan Event
	broadcast  chan Event
}

// NewHub creates a Hub and starts its dispatch loop in the background; it
// runs for the lifetime of the process (there's exactly one per server).
func NewHub() *Hub {
	h := &Hub{
		register:   make(chan chan Event),
		unregister: make(chan chan Event),
		broadcast:  make(chan Event),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	clients := make(map[chan Event]struct{})
	for {
		select {
		case ch := <-h.register:
			clients[ch] = struct{}{}
		case ch := <-h.unregister:
			if _, ok := clients[ch]; ok {
				delete(clients, ch)
				close(ch)
			}
		case ev := <-h.broadcast:
			for ch := range clients {
				select {
				case ch <- ev:
				default:
					// Client's buffer is full — drop the event rather than
					// block the whole hub on one slow reader; this is a
					// "something changed, maybe refetch" hint, not a queue
					// that must never lose a message.
				}
			}
		}
	}
}

// Broadcast notifies every connected client that path changed.
func (h *Hub) Broadcast(path string) {
	h.broadcast <- Event{Path: path}
}

// Handler upgrades the request to a WebSocket and streams Broadcast events
// to it until the client disconnects or the request context ends. The
// connection is write-only from the server's side — it never expects to
// read a message from the client beyond protocol-level pings.
func (h *Hub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		ctx := conn.CloseRead(r.Context())

		ch := make(chan Event, 8)
		h.register <- ch
		defer func() { h.unregister <- ch }()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if err := wsjson.Write(ctx, conn, ev); err != nil {
					return
				}
			}
		}
	}
}
