// internal/server/hub.go
package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// upgrader is used to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Note: In a production environment, you would check the origin.
	// For a local dev server, we can skip this check.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	// Registered clients.
	clients map[*websocket.Conn]bool

	// Inbound messages from the clients (not used in this implementation).
	broadcast chan []byte

	// Mutex to protect concurrent access to clients map.
	mu sync.Mutex
}

// newHub creates a new Hub.
func newHub() *Hub {
	return &Hub{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte),
	}
}

// register adds a new client to the hub.
func (h *Hub) register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	log.Println("Live-reload client connected.")
}

// unregister removes a client from the hub.
func (h *Hub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
		conn.Close()
		log.Println("Live-reload client disconnected.")
	}
}

// broadcastMessage sends a message to all registered clients.
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error writing to client: %v", err)
			// On error, assume the client disconnected and remove them.
			client.Close()
			delete(h.clients, client)
		}
	}
}

// serveWs handles WebSocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	hub.register(conn)

	// Keep the connection open, but unregister on close/error.
	// The client does not send messages to the server in this design.
	defer hub.unregister(conn)
	for {
		// Read messages from the client to detect when the connection is closed.
		if _, _, err := conn.ReadMessage(); err != nil {
			break // Exit loop on error (e.g., client closed connection)
		}
	}
}

