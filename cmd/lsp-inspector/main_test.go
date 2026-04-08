package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/arbo/lsp-inspector/internal/parser"
	"github.com/arbo/lsp-inspector/internal/server"
	"github.com/arbo/lsp-inspector/internal/watcher"
	webfs "github.com/arbo/lsp-inspector/web"
)

// lspLogLines contains a realistic lsp.log with 7 parseable lines.
var lspLogLines = strings.Join([]string{
	// 0: START marker
	`[START][2026-04-08 10:19:43] LSP logging initiated`,
	// 1: rpc.send initialize (id=1)
	`[DEBUG][2026-04-08 10:19:51] ...lsp/log.lua:151	"rpc.send"	{ id = 1, jsonrpc = "2.0", method = "initialize", params = { rootUri = "file:///project" } }`,
	// 2: rpc.receive initialize response (id=1)
	`[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.receive"	{ id = 1, jsonrpc = "2.0", result = { capabilities = { hoverProvider = true } } }`,
	// 3: rpc.send initialized notification (no id, vim.empty_dict() params)
	`[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.send"	{ jsonrpc = "2.0", method = "initialized", params = vim.empty_dict() }`,
	// 4: rpc.receive al/projectsLoadedNotification #1
	`[DEBUG][2026-04-08 10:19:53] ...lsp/log.lua:151	"rpc.receive"	{ jsonrpc = "2.0", method = "al/projectsLoadedNotification", params = { projects = { "c:/project1" } } }`,
	// 5: rpc.receive al/projectsLoadedNotification #2
	`[DEBUG][2026-04-08 10:19:54] ...lsp/log.lua:151	"rpc.receive"	{ jsonrpc = "2.0", method = "al/projectsLoadedNotification", params = { projects = { "c:/project2" } } }`,
	// 6: rpc.receive al/projectsLoadedNotification #3
	`[DEBUG][2026-04-08 10:19:55] ...lsp/log.lua:151	"rpc.receive"	{ jsonrpc = "2.0", method = "al/projectsLoadedNotification", params = { projects = { "c:/project3" } } }`,
}, "\n")

func TestEndToEnd(t *testing.T) {
	// --- Step 1: Create temp dir with lsp.log ---
	dir := t.TempDir()
	logPath := filepath.Join(dir, "lsp.log")
	if err := os.WriteFile(logPath, []byte(lspLogLines), 0644); err != nil {
		t.Fatalf("write lsp.log: %v", err)
	}

	// --- Step 2: Create watcher, read all lines, parse ---
	w, err := watcher.New(logPath)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}
	defer w.Close()

	lines := w.ReadAll()
	msgs := parser.ParseLines(lines, 0)

	// --- Step 3: Verify parsed messages ---
	if len(msgs) != 7 {
		t.Fatalf("expected 7 messages, got %d", len(msgs))
	}

	// msg[0]: START
	if msgs[0].Level != "START" {
		t.Errorf("msg[0] level: got %q, want %q", msgs[0].Level, "START")
	}

	// msg[1]: rpc.send initialize with id=1
	if msgs[1].Method != "initialize" {
		t.Errorf("msg[1] method: got %q, want %q", msgs[1].Method, "initialize")
	}
	if msgs[1].ID == nil || *msgs[1].ID != 1 {
		t.Errorf("msg[1] id: got %v, want 1", msgs[1].ID)
	}

	// msg[2]: direction is "receive"
	if msgs[2].Direction != parser.DirectionReceive {
		t.Errorf("msg[2] direction: got %q, want %q", msgs[2].Direction, parser.DirectionReceive)
	}

	// msg[3]: initialized notification — method "initialized", nil id
	if msgs[3].Method != "initialized" {
		t.Errorf("msg[3] method: got %q, want %q", msgs[3].Method, "initialized")
	}
	if msgs[3].ID != nil {
		t.Errorf("msg[3] id: got %v, want nil (notification)", msgs[3].ID)
	}

	// All payloads that exist must be valid JSON
	for i, m := range msgs {
		if m.Payload != nil {
			var v interface{}
			if err := json.Unmarshal(m.Payload, &v); err != nil {
				t.Errorf("msg[%d] payload is not valid JSON: %v — raw: %s", i, err, string(m.Payload))
			}
		}
	}

	// --- Step 4: Create server with web.FS and parsed messages ---
	srv := server.New(webfs.FS, msgs)

	// --- Step 5: Start httptest server, connect WebSocket, verify ---
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer conn.Close()

	// Read initial envelope
	var envelope struct {
		Type     string            `json:"type"`
		Messages []*parser.Message `json:"messages"`
	}
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatalf("read websocket envelope: %v", err)
	}
	if envelope.Type != "initial" {
		t.Errorf("envelope type: got %q, want %q", envelope.Type, "initial")
	}
	if len(envelope.Messages) != 7 {
		t.Errorf("envelope messages count: got %d, want 7", len(envelope.Messages))
	}

	// HTTP GET / returns 200
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("http GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status: got %d, want 200", resp.StatusCode)
	}
}
