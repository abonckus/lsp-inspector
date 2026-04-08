package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gorilla/websocket"

	"github.com/arbo/lsp-inspector/internal/parser"
)

func TestServer_ServesStaticFiles(t *testing.T) {
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>test</html>")},
	}
	s := New(staticFS, nil)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestServer_WebSocketSendsInitialMessages(t *testing.T) {
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	initialMsgs := []*parser.Message{
		{Level: "START", Timestamp: "2026-04-08 10:00:00", Type: "info", Direction: "info", Line: 0},
		{Level: "DEBUG", Timestamp: "2026-04-08 10:00:01", Type: "rpc.send", Direction: "send", Method: "initialize", Line: 1},
	}
	s := New(staticFS, initialMsgs)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var envelope struct {
		Type     string            `json:"type"`
		Messages []*parser.Message `json:"messages"`
	}
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Type != "initial" {
		t.Errorf("type: got %q, want %q", envelope.Type, "initial")
	}
	if len(envelope.Messages) != 2 {
		t.Fatalf("messages: got %d, want 2", len(envelope.Messages))
	}
}

func TestServer_WebSocketBroadcastsAppend(t *testing.T) {
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	s := New(staticFS, nil)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Read initial (empty)
	var initial struct {
		Type string `json:"type"`
	}
	conn.ReadJSON(&initial)

	// Broadcast new messages
	newMsgs := []*parser.Message{
		{Level: "DEBUG", Timestamp: "2026-04-08 10:00:05", Type: "rpc.send", Direction: "send", Method: "shutdown", Line: 5},
	}
	s.Broadcast(newMsgs)

	var envelope struct {
		Type     string            `json:"type"`
		Messages []*parser.Message `json:"messages"`
	}
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Type != "append" {
		t.Errorf("type: got %q, want %q", envelope.Type, "append")
	}
	if len(envelope.Messages) != 1 {
		t.Fatalf("messages: got %d, want 1", len(envelope.Messages))
	}

	// Suppress unused import
	_ = json.Marshal
	_ = fs.FS(nil)
}
