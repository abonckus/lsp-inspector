package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/abonckus/lsp-inspector/internal/parser"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// Server handles HTTP and WebSocket connections.
type Server struct {
	staticFS    fs.FS
	logPath     string
	initialMsgs []*parser.Message
	mu          sync.RWMutex
	clients     map[*client]struct{}
	onClear     func() // called after log file is truncated
}

// New creates a Server with the given static filesystem, initial messages,
// the path to the log file, and a callback invoked after the log is cleared.
func New(staticFS fs.FS, initialMsgs []*parser.Message, logPath string, onClear func()) *Server {
	if initialMsgs == nil {
		initialMsgs = []*parser.Message{}
	}
	return &Server{
		staticFS:    staticFS,
		logPath:     logPath,
		initialMsgs: initialMsgs,
		clients:     make(map[*client]struct{}),
		onClear:     onClear,
	}
}

// Handler returns an http.Handler that serves static files and WebSocket.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/clear", s.handleClear)
	mux.Handle("/", http.FileServer(http.FS(s.staticFS)))
	return mux
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &client{conn: conn}

	s.mu.Lock()
	s.clients[c] = struct{}{}
	msgs := make([]*parser.Message, len(s.initialMsgs))
	copy(msgs, s.initialMsgs)
	s.mu.Unlock()

	// Send initial messages
	envelope := map[string]interface{}{
		"type":     "initial",
		"messages": msgs,
	}
	c.mu.Lock()
	err = conn.WriteJSON(envelope)
	c.mu.Unlock()
	if err != nil {
		s.removeClient(c)
		return
	}

	// Keep connection alive, read (and discard) client messages
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.removeClient(c)
			return
		}
	}
}

// Broadcast sends new messages to all connected WebSocket clients
// and adds them to the initial message list for future connections.
func (s *Server) Broadcast(msgs []*parser.Message) {
	s.mu.Lock()
	s.initialMsgs = append(s.initialMsgs, msgs...)
	clients := make([]*client, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.Unlock()

	data, err := json.Marshal(map[string]interface{}{
		"type":     "append",
		"messages": msgs,
	})
	if err != nil {
		return
	}

	for _, c := range clients {
		c.mu.Lock()
		err := c.conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
		if err != nil {
			s.removeClient(c)
		}
	}
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Truncate the log file
	if err := os.Truncate(s.logPath, 0); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reset server state
	s.mu.Lock()
	s.initialMsgs = []*parser.Message{}
	clients := make([]*client, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.Unlock()

	// Notify watcher to reset its offset
	if s.onClear != nil {
		s.onClear()
	}

	// Tell all clients to clear
	data, _ := json.Marshal(map[string]interface{}{
		"type":     "initial",
		"messages": []*parser.Message{},
	})
	for _, c := range clients {
		c.mu.Lock()
		c.conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) removeClient(c *client) {
	s.mu.Lock()
	delete(s.clients, c)
	s.mu.Unlock()
	c.conn.Close()
}
