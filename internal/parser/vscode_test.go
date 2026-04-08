package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIsVSCodeFormat(t *testing.T) {
	vscode := []string{`[Trace - 10:03:59 AM] Sending request 'initialize - (0)'.`}
	if !IsVSCodeFormat(vscode) {
		t.Error("expected VS Code format to be detected")
	}

	neovim := []string{`[DEBUG][2026-04-08 10:19:51] ...lsp/log.lua:151	"rpc.send"	{}`}
	if IsVSCodeFormat(neovim) {
		t.Error("expected Neovim format not to match")
	}

	empty := []string{"", "", ""}
	if IsVSCodeFormat(empty) {
		t.Error("expected empty lines not to match")
	}
}

func TestParseVSCode_SendRequest(t *testing.T) {
	lines := []string{
		`[Trace - 10:03:59 AM] Sending request 'initialize - (0)'.`,
		`Params: {`,
		`    "processId": 36080,`,
		`    "rootUri": "file:///project"`,
		`}`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Direction != DirectionSend {
		t.Errorf("direction: got %q, want %q", m.Direction, DirectionSend)
	}
	if m.Method != "initialize" {
		t.Errorf("method: got %q, want %q", m.Method, "initialize")
	}
	if m.ID == nil || *m.ID != 0 {
		t.Errorf("id: got %v, want 0", m.ID)
	}
	if m.Timestamp != "10:03:59 AM" {
		t.Errorf("timestamp: got %q, want %q", m.Timestamp, "10:03:59 AM")
	}
	if m.Payload == nil {
		t.Fatal("payload should not be nil")
	}
	var v map[string]interface{}
	if err := json.Unmarshal(m.Payload, &v); err != nil {
		t.Errorf("payload is not valid JSON: %v", err)
	}
	if v["processId"] != float64(36080) {
		t.Errorf("payload processId: got %v", v["processId"])
	}
}

func TestParseVSCode_ReceivedResponse(t *testing.T) {
	lines := []string{
		`[Trace - 10:04:00 AM] Received response 'initialize - (0)' in 741ms.`,
		`Result: {`,
		`    "capabilities": {`,
		`        "hoverProvider": true`,
		`    }`,
		`}`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Direction != DirectionReceive {
		t.Errorf("direction: got %q, want %q", m.Direction, DirectionReceive)
	}
	if m.Method != "initialize" {
		t.Errorf("method: got %q, want %q", m.Method, "initialize")
	}
	if m.ID == nil || *m.ID != 0 {
		t.Errorf("id: got %v, want 0", m.ID)
	}
	if m.Payload == nil {
		t.Fatal("payload should not be nil")
	}
}

func TestParseVSCode_SendNotification(t *testing.T) {
	lines := []string{
		`[Trace - 10:04:00 AM] Sending notification 'initialized'.`,
		`No parameters provided.`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Direction != DirectionSend {
		t.Errorf("direction: got %q, want %q", m.Direction, DirectionSend)
	}
	if m.Method != "initialized" {
		t.Errorf("method: got %q, want %q", m.Method, "initialized")
	}
	if m.ID != nil {
		t.Errorf("id: got %v, want nil", m.ID)
	}
	if m.Payload != nil {
		t.Errorf("payload should be nil for 'No parameters provided'")
	}
}

func TestParseVSCode_ReceivedNotification(t *testing.T) {
	lines := []string{
		`[Trace - 10:04:08 AM] Received notification 'textDocument/publishDiagnostics'.`,
		`Params: {`,
		`    "uri": "file:///test.al",`,
		`    "diagnostics": []`,
		`}`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Direction != DirectionReceive {
		t.Errorf("direction: got %q, want %q", m.Direction, DirectionReceive)
	}
	if m.Method != "textDocument/publishDiagnostics" {
		t.Errorf("method: got %q, want %q", m.Method, "textDocument/publishDiagnostics")
	}
	if m.ID != nil {
		t.Errorf("id: got %v, want nil", m.ID)
	}
}

func TestParseVSCode_ReceivedRequest(t *testing.T) {
	lines := []string{
		`[Trace - 10:04:11 AM] Received request 'al/activeProjectLoaded - (50)'.`,
		`Params: {`,
		`    "activeProjectFolder": "file:///project"`,
		`}`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Direction != DirectionReceive {
		t.Errorf("direction: got %q, want %q", m.Direction, DirectionReceive)
	}
	if m.Method != "al/activeProjectLoaded" {
		t.Errorf("method: got %q, want %q", m.Method, "al/activeProjectLoaded")
	}
	if m.ID == nil || *m.ID != 50 {
		t.Errorf("id: got %v, want 50", m.ID)
	}
}

func TestParseVSCode_SendResponse(t *testing.T) {
	lines := []string{
		`[Trace - 10:04:11 AM] Sending response 'al/activeProjectLoaded - (50)'. Processing request took 2ms`,
		`No result returned.`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Direction != DirectionSend {
		t.Errorf("direction: got %q, want %q", m.Direction, DirectionSend)
	}
	if m.Method != "al/activeProjectLoaded" {
		t.Errorf("method: got %q, want %q", m.Method, "al/activeProjectLoaded")
	}
	if m.ID == nil || *m.ID != 50 {
		t.Errorf("id: got %v, want 50", m.ID)
	}
	if m.Payload != nil {
		t.Errorf("payload should be nil for 'No result returned'")
	}
}

func TestParseVSCode_NoResultResponse(t *testing.T) {
	lines := []string{
		`[Trace - 10:04:02 AM] Received response 'al/didChangeActiveDocument - (1)' in 1220ms.`,
		`No result returned.`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Payload != nil {
		t.Error("payload should be nil")
	}
}

func TestParseVSCode_MultipleMessages(t *testing.T) {
	input := strings.Join([]string{
		`[Trace - 10:03:59 AM] Sending request 'initialize - (0)'.`,
		`Params: {`,
		`    "processId": 1`,
		`}`,
		``,
		``,
		`[Trace - 10:04:00 AM] Received response 'initialize - (0)' in 741ms.`,
		`Result: {`,
		`    "capabilities": {}`,
		`}`,
		``,
		``,
		`[Trace - 10:04:00 AM] Sending notification 'initialized'.`,
		`No parameters provided.`,
		``,
		``,
		`[Trace - 10:04:00 AM] Received notification 'textDocument/publishDiagnostics'.`,
		`Params: {`,
		`    "uri": "file:///test",`,
		`    "diagnostics": []`,
		`}`,
	}, "\n")

	lines := strings.Split(input, "\n")
	msgs := ParseLines(lines, 0)
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	// Verify directions
	if msgs[0].Direction != DirectionSend {
		t.Errorf("msg[0] direction: got %q", msgs[0].Direction)
	}
	if msgs[1].Direction != DirectionReceive {
		t.Errorf("msg[1] direction: got %q", msgs[1].Direction)
	}
	if msgs[2].Direction != DirectionSend {
		t.Errorf("msg[2] direction: got %q", msgs[2].Direction)
	}
	if msgs[3].Direction != DirectionReceive {
		t.Errorf("msg[3] direction: got %q", msgs[3].Direction)
	}

	// All payloads that exist must be valid JSON
	for i, m := range msgs {
		if m.Payload != nil {
			var v interface{}
			if err := json.Unmarshal(m.Payload, &v); err != nil {
				t.Errorf("msg[%d] payload is not valid JSON: %v", i, err)
			}
		}
	}
}

func TestParseVSCode_LineNumbers(t *testing.T) {
	input := strings.Join([]string{
		`[Trace - 10:03:59 AM] Sending request 'initialize - (0)'.`,
		`Params: {`,
		`    "processId": 1`,
		`}`,
		``,
		``,
		`[Trace - 10:04:00 AM] Sending notification 'initialized'.`,
		`No parameters provided.`,
	}, "\n")

	lines := strings.Split(input, "\n")
	msgs := ParseLines(lines, 100)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Line != 100 {
		t.Errorf("msg[0] line: got %d, want 100", msgs[0].Line)
	}
	if msgs[1].Line != 106 {
		t.Errorf("msg[1] line: got %d, want 106", msgs[1].Line)
	}
}

func TestParseVSCode_NotificationWithParams(t *testing.T) {
	lines := []string{
		`[Trace - 10:04:00 AM] Sending notification 'textDocument/didOpen'.`,
		`Params: {`,
		`    "textDocument": {`,
		`        "uri": "file:///test.al",`,
		`        "languageId": "al",`,
		`        "version": 1,`,
		`        "text": "codeunit 50100 Test {}"`,
		`    }`,
		`}`,
	}
	msgs := ParseLines(lines, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.Method != "textDocument/didOpen" {
		t.Errorf("method: got %q, want %q", m.Method, "textDocument/didOpen")
	}
	if m.Payload == nil {
		t.Fatal("payload should not be nil")
	}
	var v map[string]interface{}
	if err := json.Unmarshal(m.Payload, &v); err != nil {
		t.Errorf("payload not valid JSON: %v", err)
	}
}
