package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSManager tracks active WebSocket connections and provides send/receive
// operations across them.
type WSManager struct {
	connections map[string]*wsConn
	mu          sync.RWMutex
}

type wsConn struct {
	ID      string
	URL     string
	conn    *websocket.Conn
	messages []WSMessage
	mu       sync.Mutex
	closed   bool
	closeCh  chan struct{}
}

const maxWSMessages = 100

// NewWSManager creates a new WebSocket connection manager.
func NewWSManager() *WSManager {
	return &WSManager{
		connections: make(map[string]*wsConn),
	}
}

// Connect dials a WebSocket endpoint and starts a background read loop.
// Returns the connection ID on success.
func (m *WSManager) Connect(wsURL string, headers map[string]string) (string, error) {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, h)
	if err != nil {
		return "", fmt.Errorf("websocket dial: %w", err)
	}

	id := generateWSID()
	wc := &wsConn{
		ID:      id,
		URL:     wsURL,
		conn:    conn,
		closeCh: make(chan struct{}),
	}

	m.mu.Lock()
	m.connections[id] = wc
	m.mu.Unlock()

	// Background read goroutine
	go func() {
		defer func() {
			wc.mu.Lock()
			wc.closed = true
			wc.mu.Unlock()
		}()
		for {
			select {
			case <-wc.closeCh:
				return
			default:
			}
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			wc.mu.Lock()
			wc.messages = append(wc.messages, WSMessage{
				Direction: "received",
				Data:      string(msg),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			if len(wc.messages) > maxWSMessages {
				wc.messages = wc.messages[len(wc.messages)-maxWSMessages:]
			}
			wc.mu.Unlock()
		}
	}()

	return id, nil
}

// Send writes a text message to the specified connection.
func (m *WSManager) Send(connID string, message string) error {
	m.mu.RLock()
	wc, ok := m.connections[connID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("connection %s not found", connID)
	}
	wc.mu.Lock()
	defer wc.mu.Unlock()
	if wc.closed {
		return fmt.Errorf("connection %s is closed", connID)
	}
	if err := wc.conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	wc.messages = append(wc.messages, WSMessage{
		Direction: "sent",
		Data:      message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if len(wc.messages) > maxWSMessages {
		wc.messages = wc.messages[len(wc.messages)-maxWSMessages:]
	}
	return nil
}

// Close terminates the connection and removes it from the manager.
// Returns the number of messages exchanged during the session.
func (m *WSManager) Close(connID string) (int, error) {
	m.mu.Lock()
	wc, ok := m.connections[connID]
	if !ok {
		m.mu.Unlock()
		return 0, fmt.Errorf("connection %s not found", connID)
	}
	delete(m.connections, connID)
	m.mu.Unlock()

	wc.mu.Lock()
	count := len(wc.messages)
	wc.mu.Unlock()

	close(wc.closeCh)
	wc.conn.Close()

	return count, nil
}

// Get returns a snapshot of the connection state and message history.
func (m *WSManager) Get(connID string) (*WSConnection, error) {
	m.mu.RLock()
	wc, ok := m.connections[connID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("connection %s not found", connID)
	}
	wc.mu.Lock()
	defer wc.mu.Unlock()
	msgs := make([]WSMessage, len(wc.messages))
	copy(msgs, wc.messages)
	return &WSConnection{
		ID:        wc.ID,
		URL:       wc.URL,
		Connected: !wc.closed,
		Messages:  msgs,
	}, nil
}

func generateWSID() string {
	b := make([]byte, 3)
	rand.Read(b)
	return "ws-" + hex.EncodeToString(b)
}
