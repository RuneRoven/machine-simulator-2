package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/event"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Hub manages WebSocket connections and broadcasts events.
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
	bus     *event.Bus
}

type wsClient struct {
	conn   *websocket.Conn
	send   chan []byte
	closed atomic.Bool
}

// NewHub creates a new WebSocket hub and subscribes to events.
func NewHub(bus *event.Bus) *Hub {
	h := &Hub{
		clients: make(map[*wsClient]struct{}),
		bus:     bus,
	}

	// Subscribe to all relevant events
	eventTypes := []string{
		event.MachineStateChange,
		event.MachineCycleComplete,
		event.LinePartMoved,
		event.OrderCreated,
		event.OrderReleased,
		event.OrderProgress,
		event.OrderClosed,
		event.BufferFull,
		event.BufferEmpty,
		event.LineDeadlockRecovered,
	}

	for _, et := range eventTypes {
		et := et
		_ = et
		bus.Subscribe(et, func(e event.Event) {
			h.broadcast(e)
		})
	}

	return h
}

// HandleWebSocket upgrades an HTTP connection to WebSocket.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}

	client := &wsClient{
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	go h.writePump(client)
	go h.readPump(client)
}

func (h *Hub) broadcast(e event.Event) {
	msg := wsMessage{
		Type:      string(e.Type),
		MachineID: e.MachineID,
		LineID:    e.LineID,
		Data:      e.Data,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.closed.Load() {
			continue
		}
		select {
		case client.send <- data:
		default:
			// Client buffer full, skip
		}
	}
}

func (h *Hub) removeClient(client *wsClient) {
	if !client.closed.CompareAndSwap(false, true) {
		return // already closed
	}
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
	close(client.send)
	client.conn.Close()
}

func (h *Hub) writePump(client *wsClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer h.removeClient(client)

	for {
		select {
		case msg, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) readPump(client *wsClient) {
	defer h.removeClient(client)
	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// ClientCount returns the number of connected WebSocket clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

type wsMessage struct {
	Type      string      `json:"type"`
	MachineID string      `json:"machine_id,omitempty"`
	LineID    string      `json:"line_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}
