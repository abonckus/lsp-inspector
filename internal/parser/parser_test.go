package parser

import (
	"encoding/json"
	"testing"
)

func TestParseLine_RpcSend(t *testing.T) {
	line := `[DEBUG][2026-04-08 10:19:30] ...lsp/log.lua:151	"rpc.send"	{ id = 21, jsonrpc = "2.0", method = "textDocument/documentHighlight", params = { position = { character = 47, line = 58 }, textDocument = { uri = "file:///test.al" } } }`
	msg, err := ParseLine(line, 17)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.Level != "DEBUG" {
		t.Errorf("level: got %q, want %q", msg.Level, "DEBUG")
	}
	if msg.Timestamp != "2026-04-08 10:19:30" {
		t.Errorf("timestamp: got %q, want %q", msg.Timestamp, "2026-04-08 10:19:30")
	}
	if msg.Type != "rpc.send" {
		t.Errorf("type: got %q, want %q", msg.Type, "rpc.send")
	}
	if msg.Direction != DirectionSend {
		t.Errorf("direction: got %q, want %q", msg.Direction, DirectionSend)
	}
	if msg.Method != "textDocument/documentHighlight" {
		t.Errorf("method: got %q, want %q", msg.Method, "textDocument/documentHighlight")
	}
	if msg.ID == nil || *msg.ID != 21 {
		t.Errorf("id: got %v, want 21", msg.ID)
	}
	if msg.Line != 17 {
		t.Errorf("line: got %d, want 17", msg.Line)
	}
	if msg.Payload == nil {
		t.Error("payload should not be nil")
	}
}

func TestParseLine_RpcReceive(t *testing.T) {
	line := `[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.receive"	{ id = 1, jsonrpc = "2.0", result = { capabilities = { hoverProvider = true } } }`
	msg, err := ParseLine(line, 32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "rpc.receive" {
		t.Errorf("type: got %q, want %q", msg.Type, "rpc.receive")
	}
	if msg.Direction != DirectionReceive {
		t.Errorf("direction: got %q, want %q", msg.Direction, DirectionReceive)
	}
	if msg.ID == nil || *msg.ID != 1 {
		t.Errorf("id: got %v, want 1", msg.ID)
	}
	// Response has no method — method should be empty
	if msg.Method != "" {
		t.Errorf("method: got %q, want empty", msg.Method)
	}
}

func TestParseLine_Notification(t *testing.T) {
	line := `[DEBUG][2026-04-08 10:19:22] ...lsp/log.lua:151	"rpc.receive"	{ jsonrpc = "2.0", method = "al/projectsLoadedNotification", params = { projects = { "c:/test" } } }`
	msg, err := ParseLine(line, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Method != "al/projectsLoadedNotification" {
		t.Errorf("method: got %q, want %q", msg.Method, "al/projectsLoadedNotification")
	}
	if msg.ID != nil {
		t.Errorf("id: got %v, want nil (notification)", msg.ID)
	}
}

func TestParseLine_ClientRequest(t *testing.T) {
	line := `[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"LSP[al_ls]"	"client.request"	2	"textDocument/inlayHint"	{ range = { ["end"] = { character = 0, line = 203 } } }	<function 1>	23`
	msg, err := ParseLine(line, 40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "client.request" {
		t.Errorf("type: got %q, want %q", msg.Type, "client.request")
	}
	if msg.Server != "al_ls" {
		t.Errorf("server: got %q, want %q", msg.Server, "al_ls")
	}
	if msg.Method != "textDocument/inlayHint" {
		t.Errorf("method: got %q, want %q", msg.Method, "textDocument/inlayHint")
	}
}

func TestParseLine_StartMarker(t *testing.T) {
	line := `[START][2026-04-08 10:19:43] LSP logging initiated`
	msg, err := ParseLine(line, 29)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Level != "START" {
		t.Errorf("level: got %q, want %q", msg.Level, "START")
	}
	if msg.Direction != DirectionInfo {
		t.Errorf("direction: got %q, want %q", msg.Direction, DirectionInfo)
	}
	if msg.Type != "info" {
		t.Errorf("type: got %q, want %q", msg.Type, "info")
	}
}

func TestParseLine_InfoStartingRPC(t *testing.T) {
	line := `[INFO][2026-04-08 10:19:51] ...lsp/log.lua:151	"Starting RPC client"	{ cmd = { "server.exe", "--stdio" }, extra = {} }`
	msg, err := ParseLine(line, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "info" {
		t.Errorf("type: got %q, want %q", msg.Type, "info")
	}
	if msg.Direction != DirectionInfo {
		t.Errorf("direction: got %q", msg.Direction)
	}
}

func TestParseFile(t *testing.T) {
	lines := []string{
		`[START][2026-04-08 10:19:43] LSP logging initiated`,
		`[DEBUG][2026-04-08 10:19:51] ...lsp/log.lua:151	"rpc.send"	{ id = 1, jsonrpc = "2.0", method = "initialize", params = {} }`,
		`[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.receive"	{ id = 1, jsonrpc = "2.0", result = {} }`,
	}

	msgs := ParseLines(lines, 0)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Level != "START" {
		t.Errorf("msg[0] level: got %q", msgs[0].Level)
	}
	if msgs[1].Type != "rpc.send" {
		t.Errorf("msg[1] type: got %q", msgs[1].Type)
	}
	if msgs[2].Type != "rpc.receive" {
		t.Errorf("msg[2] type: got %q", msgs[2].Type)
	}
}

func TestParseLines_DeduplicatesClientRequest(t *testing.T) {
	lines := []string{
		`[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"LSP[al_ls]"	"client.request"	2	"textDocument/inlayHint"	{ range = {} }	<function 1>	23`,
		`[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.send"	{ id = 3, jsonrpc = "2.0", method = "textDocument/inlayHint", params = { range = {} } }`,
	}

	msgs := ParseLines(lines, 0)
	// client.request should be deduped, leaving only rpc.send
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after dedup, got %d", len(msgs))
	}
	if msgs[0].Type != "rpc.send" {
		t.Errorf("expected rpc.send, got %q", msgs[0].Type)
	}
	// Server name should be copied from client.request
	if msgs[0].Server != "al_ls" {
		t.Errorf("expected server al_ls, got %q", msgs[0].Server)
	}
}

// Verify payload is valid JSON
func TestParseLine_PayloadIsValidJSON(t *testing.T) {
	line := `[DEBUG][2026-04-08 10:19:30] ...lsp/log.lua:151	"rpc.send"	{ id = 21, jsonrpc = "2.0", method = "textDocument/documentHighlight", params = { position = { character = 47, line = 58 } } }`
	msg, err := ParseLine(line, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var v interface{}
	if err := json.Unmarshal(msg.Payload, &v); err != nil {
		t.Errorf("payload is not valid JSON: %v\nraw: %s", err, string(msg.Payload))
	}
}
