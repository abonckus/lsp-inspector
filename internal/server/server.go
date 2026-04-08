package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/arbo/lsp-inspector/internal/parser"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server handles HTTP and WebSocket connections.
type Server struct {
	staticFS    fs.FS
	initialMsgs []*parser.Message
	mu          sync.RWMutex
	clients     map[*websocket.Conn]struct{}
}

// New creates a Server with the given static filesystem and initial messages.
func New(staticFS fs.FS, initialMsgs []*parser.Message) *Server {
	if initialMsgs == nil {
		initialMsgs = []*parser.Message{}
	}
	return &Server{
		staticFS:    staticFS,
		initialMsgs: initialMsgs,
		clients:     make(map[*websocket.Conn]struct{}),
	}
}

// Handler returns an http.Handler that serves static files and WebSocket.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.Handle("/", http.FileServer(http.FS(s.staticFS)))
	return mux
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.clients[conn] = struct{}{}
	msgs := make([]*parser.Message, len(s.initialMsgs))
	copy(msgs, s.initialMsgs)
	s.mu.Unlock()

	// Send initial messages
	envelope := map[string]interface{}{
		"type":     "initial",
		"messages": msgs,
	}
	if err := conn.WriteJSON(envelope); err != nil {
		s.removeClient(conn)
		return
	}

	// Keep connection alive, read (and discard) client messages
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.removeClient(conn)
			return
		}
	}
}

// Broadcast sends new messages to all connected WebSocket clients
// and adds them to the initial message list for future connections.
func (s *Server) Broadcast(msgs []*parser.Message) {
	s.mu.Lock()
	s.initialMsgs = append(s.initialMsgs, msgs...)
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.Unlock()

	envelope := map[string]interface{}{
		"type":     "append",
		"messages": msgs,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return
	}

	for _, c := range clients {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			s.removeClient(c)
		}
	}
}

func (s *Server) removeClient(conn *websocket.Conn) {
	s.mu.Lock()
	delete(s.clients, conn)
	s.mu.Unlock()
	conn.Close()
}
