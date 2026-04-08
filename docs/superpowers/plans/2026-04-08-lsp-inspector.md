# LSP Inspector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI tool that parses Neovim's lsp.log and displays LSP client/server messages in a chat-style web UI with live file watching.

**Architecture:** Single Go binary with embedded HTML/CSS/JS frontend. Go handles log parsing, file watching (fsnotify), and WebSocket streaming. Browser renders a chat timeline with filtering, collapsing, and request/response linking.

**Tech Stack:** Go 1.22+, fsnotify, gorilla/websocket, go:embed, vanilla HTML/CSS/JS

**Spec:** `docs/superpowers/specs/2026-04-08-lsp-inspector-design.md`

**Sample log for testing:** `C:\Users\arbo\AppData\Local\nvim-data\lsp.log`

---

## File Map

| File | Responsibility |
|---|---|
| `go.mod` | Module definition and dependencies |
| `internal/parser/message.go` | Message struct and constants |
| `internal/parser/lua2json.go` | Lua table syntax to JSON converter |
| `internal/parser/lua2json_test.go` | Tests for Lua-to-JSON conversion |
| `internal/parser/parser.go` | Log line parser, extracts structured Message from raw lines |
| `internal/parser/parser_test.go` | Tests for log line parsing |
| `internal/watcher/watcher.go` | File watcher with incremental reading |
| `internal/watcher/watcher_test.go` | Tests for file watcher |
| `internal/server/server.go` | HTTP server, WebSocket handler, static file embedding |
| `internal/server/server_test.go` | Tests for HTTP/WebSocket server |
| `web/index.html` | Single-page app shell |
| `web/style.css` | Dark theme styles, chat layout, bubble styling |
| `web/app.js` | Timeline rendering, WebSocket client, filtering, collapsing |
| `cmd/lsp-inspector/main.go` | CLI entry point, wires everything together |

---

### Task 1: Project Scaffolding and Message Types

**Files:**
- Create: `go.mod`
- Create: `internal/parser/message.go`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go mod init github.com/arbo/lsp-inspector
```
Expected: `go.mod` created with module path.

- [ ] **Step 2: Create message type definitions**

Create `internal/parser/message.go`:

```go
package parser

import "encoding/json"

// Direction constants
const (
	DirectionSend    = "send"
	DirectionReceive = "receive"
	DirectionInfo    = "info"
)

// Message represents a single parsed LSP log entry.
type Message struct {
	Level      string          `json:"level"`
	Timestamp  string          `json:"timestamp"`
	Source     string          `json:"source"`
	Type       string          `json:"type"`
	Direction  string          `json:"direction"`
	Server     string          `json:"server"`
	Method     string          `json:"method"`
	ID         *int            `json:"id"`
	Payload    json.RawMessage `json:"payload"`
	RawPayload string          `json:"rawPayload"`
	Line       int             `json:"line"`
}
```

- [ ] **Step 3: Verify it compiles**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go build ./internal/parser/
```
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add go.mod internal/parser/message.go
git commit -m "feat: scaffold project and define Message type"
```

---

### Task 2: Lua Table to JSON Converter

**Files:**
- Create: `internal/parser/lua2json.go`
- Create: `internal/parser/lua2json_test.go`

This is the most complex parsing component. The converter handles Lua table syntax from Neovim's lsp.log and produces valid JSON.

- [ ] **Step 1: Write failing tests for basic Lua-to-JSON conversions**

Create `internal/parser/lua2json_test.go`:

```go
package parser

import "testing"

func TestLua2JSON_SimpleTable(t *testing.T) {
	input := `{ key = "value" }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"key":"value"}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_NestedTable(t *testing.T) {
	input := `{ outer = { inner = 42 } }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"outer":{"inner":42}}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_VimNIL(t *testing.T) {
	input := `{ result = vim.NIL }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"result":null}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_VimEmptyDict(t *testing.T) {
	input := `{ params = vim.empty_dict() }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"params":{}}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_Booleans(t *testing.T) {
	input := `{ enabled = true, disabled = false }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"enabled":true,"disabled":false}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_Array(t *testing.T) {
	input := `{ "one", "two", "three" }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `["one","two","three"]`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_BracketKey(t *testing.T) {
	input := `{ ["end"] = { character = 0, line = 203 } }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"end":{"character":0,"line":203}}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_FunctionRef(t *testing.T) {
	input := `{ callback = <function 1> }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"callback":"<function 1>"}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_TableRef(t *testing.T) {
	input := `{ data = <table 2> }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"data":"<table 2>"}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_BackRefWithInline(t *testing.T) {
	input := `{ items = <2>{ name = "test" } }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"items":{"name":"test"}}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_MixedArrayAndKeys(t *testing.T) {
	input := `{ [23] = true }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"23":true}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_SingleQuoteString(t *testing.T) {
	input := `{ name = 'hello' }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name":"hello"}`
	if normalizeJSON(got) != normalizeJSON(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLua2JSON_RealLogPayload(t *testing.T) {
	input := `{ id = 21, jsonrpc = "2.0", method = "textDocument/documentHighlight", params = { position = { character = 47, line = 58 }, textDocument = { uri = "file:///C:/Users/arbo/test.al" } } }`
	got, err := Lua2JSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify it's valid JSON by unmarshalling
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("result is not valid JSON: %v\ngot: %s", err, got)
	}
	if m["id"].(float64) != 21 {
		t.Errorf("expected id=21, got %v", m["id"])
	}
	if m["method"].(string) != "textDocument/documentHighlight" {
		t.Errorf("expected method=textDocument/documentHighlight, got %v", m["method"])
	}
}

// normalizeJSON re-encodes JSON to remove whitespace differences.
func normalizeJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/parser/ -v -run TestLua2JSON
```
Expected: Compilation error — `Lua2JSON` not defined.

- [ ] **Step 3: Implement the Lua-to-JSON converter**

Create `internal/parser/lua2json.go`. This is a hand-written recursive descent parser for Lua table syntax.

```go
package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// Lua2JSON converts a Lua table string to a JSON string.
// It handles the subset of Lua syntax found in Neovim's lsp.log output.
func Lua2JSON(input string) (string, error) {
	p := &luaParser{input: input, pos: 0}
	val, err := p.parseValue()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(val)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type luaParser struct {
	input string
	pos   int
}

func (p *luaParser) parseValue() (interface{}, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	ch := p.input[p.pos]

	// Table
	if ch == '{' {
		return p.parseTable()
	}
	// String (double-quoted)
	if ch == '"' {
		return p.parseString('"')
	}
	// String (single-quoted)
	if ch == '\'' {
		return p.parseString('\'')
	}
	// Angle-bracket refs: <function N>, <table N>, <N>{ ... }
	if ch == '<' {
		return p.parseAngleBracket()
	}
	// Number or negative number
	if ch == '-' || (ch >= '0' && ch <= '9') {
		return p.parseNumber()
	}
	// Keywords: true, false, vim.NIL, vim.empty_dict()
	if p.matchKeyword("true") {
		return true, nil
	}
	if p.matchKeyword("false") {
		return false, nil
	}
	if p.matchKeyword("vim.NIL") {
		return nil, nil
	}
	if p.matchKeyword("vim.empty_dict()") {
		return map[string]interface{}{}, nil
	}

	// Identifier (bare word — treat as string, may appear as metatable key etc.)
	if isIdentStart(ch) {
		return p.parseIdentifier()
	}

	return nil, fmt.Errorf("unexpected character %q at position %d", ch, p.pos)
}

func (p *luaParser) parseTable() (interface{}, error) {
	p.pos++ // skip '{'
	p.skipWhitespace()

	if p.pos < len(p.input) && p.input[p.pos] == '}' {
		p.pos++
		return map[string]interface{}{}, nil
	}

	// Determine if this is an array or a map by peeking ahead.
	// If the first element has a key (identifier =, or ["key"] =), it's a map.
	// If the first element is a plain value, it's an array.
	isMap := p.peekIsMapEntry()

	if isMap {
		return p.parseMap()
	}
	return p.parseArray()
}

func (p *luaParser) peekIsMapEntry() bool {
	saved := p.pos
	defer func() { p.pos = saved }()

	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return false
	}

	// ["key"] = ... or [number] = ...
	if p.input[p.pos] == '[' {
		// Find the closing ] and check for =
		depth := 1
		i := p.pos + 1
		for i < len(p.input) && depth > 0 {
			if p.input[i] == '[' {
				depth++
			} else if p.input[i] == ']' {
				depth--
			}
			i++
		}
		// After ], skip whitespace, check for =
		for i < len(p.input) && (p.input[i] == ' ' || p.input[i] == '\t') {
			i++
		}
		return i < len(p.input) && p.input[i] == '='
	}

	// identifier = ...
	if isIdentStart(p.input[p.pos]) {
		i := p.pos
		for i < len(p.input) && isIdentChar(p.input[i]) {
			i++
		}
		// Skip dots for vim.NIL etc — but those are values not keys
		// A key is followed by whitespace then =
		for i < len(p.input) && (p.input[i] == ' ' || p.input[i] == '\t') {
			i++
		}
		return i < len(p.input) && p.input[i] == '='
	}

	return false
}

func (p *luaParser) parseMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unexpected end of input in table")
		}
		if p.input[p.pos] == '}' {
			p.pos++
			return m, nil
		}

		// Parse key
		var key string
		if p.input[p.pos] == '[' {
			p.pos++ // skip '['
			p.skipWhitespace()
			if p.pos < len(p.input) && (p.input[p.pos] == '"' || p.input[p.pos] == '\'') {
				// ["string key"]
				k, err := p.parseString(p.input[p.pos])
				if err != nil {
					return nil, err
				}
				key = k
			} else {
				// [number key]
				n, err := p.parseNumber()
				if err != nil {
					return nil, err
				}
				key = fmt.Sprintf("%v", n)
			}
			p.skipWhitespace()
			if p.pos >= len(p.input) || p.input[p.pos] != ']' {
				return nil, fmt.Errorf("expected ']' at position %d", p.pos)
			}
			p.pos++ // skip ']'
		} else {
			// bare identifier key
			start := p.pos
			for p.pos < len(p.input) && isIdentChar(p.input[p.pos]) {
				p.pos++
			}
			key = p.input[start:p.pos]
		}

		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != '=' {
			return nil, fmt.Errorf("expected '=' after key %q at position %d", key, p.pos)
		}
		p.pos++ // skip '='

		// Parse value
		val, err := p.parseValue()
		if err != nil {
			return nil, fmt.Errorf("parsing value for key %q: %w", key, err)
		}
		m[key] = val

		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}
}

func (p *luaParser) parseArray() ([]interface{}, error) {
	var arr []interface{}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unexpected end of input in array")
		}
		if p.input[p.pos] == '}' {
			p.pos++
			return arr, nil
		}

		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		arr = append(arr, val)

		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}
}

func (p *luaParser) parseString(quote byte) (string, error) {
	p.pos++ // skip opening quote
	var sb strings.Builder
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '\\' && p.pos+1 < len(p.input) {
			p.pos++
			esc := p.input[p.pos]
			switch esc {
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(esc)
			}
			p.pos++
			continue
		}
		if ch == quote {
			p.pos++
			return sb.String(), nil
		}
		sb.WriteByte(ch)
		p.pos++
	}
	return "", fmt.Errorf("unterminated string")
}

func (p *luaParser) parseNumber() (interface{}, error) {
	start := p.pos
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
	}
	isFloat := false
	for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9') {
		p.pos++
	}
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		isFloat = true
		p.pos++
		for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9') {
			p.pos++
		}
	}
	// Scientific notation
	if p.pos < len(p.input) && (p.input[p.pos] == 'e' || p.input[p.pos] == 'E') {
		isFloat = true
		p.pos++
		if p.pos < len(p.input) && (p.input[p.pos] == '+' || p.input[p.pos] == '-') {
			p.pos++
		}
		for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9') {
			p.pos++
		}
	}

	numStr := p.input[start:p.pos]
	if isFloat {
		var f float64
		if _, err := fmt.Sscanf(numStr, "%f", &f); err != nil {
			return nil, fmt.Errorf("invalid float %q: %w", numStr, err)
		}
		return f, nil
	}
	var i int64
	if _, err := fmt.Sscanf(numStr, "%d", &i); err != nil {
		return nil, fmt.Errorf("invalid int %q: %w", numStr, err)
	}
	return i, nil
}

func (p *luaParser) parseAngleBracket() (interface{}, error) {
	start := p.pos
	p.pos++ // skip '<'

	// Read until '>'
	for p.pos < len(p.input) && p.input[p.pos] != '>' {
		p.pos++
	}
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unterminated angle bracket at position %d", start)
	}
	p.pos++ // skip '>'
	ref := p.input[start:p.pos]

	p.skipWhitespace()

	// Check if followed by '{' — back-reference with inline table: <N>{ ... }
	if p.pos < len(p.input) && p.input[p.pos] == '{' {
		return p.parseTable()
	}

	// Otherwise it's a plain reference like <function 1> or <table 2>
	return ref, nil
}

func (p *luaParser) parseIdentifier() (string, error) {
	start := p.pos
	for p.pos < len(p.input) && (isIdentChar(p.input[p.pos]) || p.input[p.pos] == '.') {
		p.pos++
	}
	word := p.input[start:p.pos]

	// Check for vim.NIL, vim.empty_dict() which might not have been caught
	if word == "vim" {
		if p.pos < len(p.input) && p.input[p.pos] == '.' {
			p.pos++
			rest := p.pos
			for p.pos < len(p.input) && isIdentChar(p.input[p.pos]) {
				p.pos++
			}
			full := "vim." + p.input[rest:p.pos]
			if full == "vim.NIL" {
				return "", nil // will be nil
			}
			if strings.HasPrefix(full, "vim.empty_dict") {
				// skip ()
				if p.pos < len(p.input) && p.input[p.pos] == '(' {
					p.pos++
					if p.pos < len(p.input) && p.input[p.pos] == ')' {
						p.pos++
					}
				}
				return "", nil // handled as empty dict
			}
			return full, nil
		}
	}
	return word, nil
}

func (p *luaParser) matchKeyword(kw string) bool {
	if p.pos+len(kw) > len(p.input) {
		return false
	}
	if p.input[p.pos:p.pos+len(kw)] != kw {
		return false
	}
	// Make sure the keyword isn't a prefix of a longer identifier
	end := p.pos + len(kw)
	if end < len(p.input) && isIdentChar(p.input[end]) {
		return false
	}
	// Special case: vim.empty_dict() — keyword includes ()
	p.pos += len(kw)
	return true
}

func (p *luaParser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '\n' || p.input[p.pos] == '\r') {
		p.pos++
	}
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentChar(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

// isWhitespace checks if the given rune is whitespace (unused but kept for clarity).
var _ = unicode.IsSpace
```

- [ ] **Step 4: Run tests**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/parser/ -v -run TestLua2JSON
```
Expected: All tests PASS. If any fail, fix and re-run.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/lua2json.go internal/parser/lua2json_test.go
git commit -m "feat: add Lua table to JSON converter"
```

---

### Task 3: Log Line Parser

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

The parser takes raw log lines and produces structured `Message` values. It uses the Lua2JSON converter from Task 2.

- [ ] **Step 1: Write failing tests for log line parsing**

Create `internal/parser/parser_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/parser/ -v -run "TestParseLine|TestParseFile"
```
Expected: Compilation error — `ParseLine` and `ParseLines` not defined.

- [ ] **Step 3: Implement the log line parser**

Create `internal/parser/parser.go`:

```go
package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Regex to match the log line header: [LEVEL][timestamp] source
var logLineRe = regexp.MustCompile(`^\[(\w+)\]\[([^\]]+)\]\s+(\S+)`)

// ParseLine parses a single lsp.log line into a Message.
// lineNum is the 0-based line number in the file.
// Returns nil (no error) for lines that don't match the expected format.
func ParseLine(line string, lineNum int) (*Message, error) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return nil, nil
	}

	matches := logLineRe.FindStringSubmatch(line)
	if matches == nil {
		return nil, nil
	}

	level := matches[1]
	timestamp := matches[2]
	source := matches[3]

	msg := &Message{
		Level:     level,
		Timestamp: timestamp,
		Source:    source,
		Line:      lineNum,
	}

	// For [START] lines, there's no tab-separated fields
	if level == "START" {
		msg.Type = "info"
		msg.Direction = DirectionInfo
		// Extract the text after the source
		idx := strings.Index(line, "] ")
		if idx >= 0 {
			msg.RawPayload = strings.TrimSpace(line[idx+2:])
		}
		return msg, nil
	}

	// Split the remainder by tabs to get fields after the source path
	headerEnd := strings.Index(line, matches[0]) + len(matches[0])
	remainder := line[headerEnd:]

	// Fields are tab-separated
	fields := strings.Split(remainder, "\t")
	// Remove empty first field (line starts with tab after source)
	var cleanFields []string
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			cleanFields = append(cleanFields, f)
		}
	}

	if len(cleanFields) == 0 {
		msg.Type = "info"
		msg.Direction = DirectionInfo
		return msg, nil
	}

	return parseFields(msg, cleanFields)
}

func parseFields(msg *Message, fields []string) (*Message, error) {
	first := unquote(fields[0])

	switch {
	case first == "rpc.send":
		msg.Type = "rpc.send"
		msg.Direction = DirectionSend
		if len(fields) > 1 {
			parseRpcPayload(msg, fields[1])
		}

	case first == "rpc.receive":
		msg.Type = "rpc.receive"
		msg.Direction = DirectionReceive
		if len(fields) > 1 {
			parseRpcPayload(msg, fields[1])
		}

	case strings.HasPrefix(first, "LSP["):
		// Extract server name from LSP[name]
		serverEnd := strings.Index(first, "]")
		if serverEnd > 4 {
			msg.Server = first[4:serverEnd]
		}
		if len(fields) > 1 {
			second := unquote(fields[1])
			if second == "client.request" {
				msg.Type = "client.request"
				msg.Direction = DirectionSend
				// Fields: "LSP[name]" "client.request" clientId "method" {payload} <function> bufnr
				if len(fields) > 3 {
					msg.Method = unquote(fields[3])
				}
				if len(fields) > 4 {
					parseLuaPayload(msg, fields[4])
				}
			} else if second == "server_capabilities" {
				msg.Type = "info"
				msg.Direction = DirectionInfo
				if len(fields) > 2 {
					parseLuaPayload(msg, fields[2])
				}
			} else {
				msg.Type = "info"
				msg.Direction = DirectionInfo
				msg.RawPayload = strings.Join(fields[1:], "\t")
			}
		}

	case first == "Starting RPC client":
		msg.Type = "info"
		msg.Direction = DirectionInfo
		if len(fields) > 1 {
			parseLuaPayload(msg, fields[1])
		}

	case first == "exit_handler":
		msg.Type = "info"
		msg.Direction = DirectionInfo
		if len(fields) > 1 {
			msg.RawPayload = fields[1]
		}

	default:
		msg.Type = "info"
		msg.Direction = DirectionInfo
		msg.RawPayload = strings.Join(fields, "\t")
	}

	return msg, nil
}

// parseRpcPayload extracts method, id, and payload from an rpc.send/rpc.receive Lua table.
func parseRpcPayload(msg *Message, raw string) {
	msg.RawPayload = raw

	jsonStr, err := Lua2JSON(raw)
	if err != nil {
		return
	}
	msg.Payload = json.RawMessage(jsonStr)

	// Extract method and id from the JSON
	var rpcMsg struct {
		Method string `json:"method"`
		ID     *int   `json:"id"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &rpcMsg); err == nil {
		msg.Method = rpcMsg.Method
		msg.ID = rpcMsg.ID
	}
}

// parseLuaPayload converts a Lua table to JSON and stores it.
func parseLuaPayload(msg *Message, raw string) {
	msg.RawPayload = raw
	jsonStr, err := Lua2JSON(raw)
	if err != nil {
		return
	}
	msg.Payload = json.RawMessage(jsonStr)
}

// ParseLines parses multiple log lines starting from a given line offset.
// It deduplicates client.request entries against rpc.send entries:
// client.request is used only to extract the server name, then discarded.
func ParseLines(lines []string, startLine int) []*Message {
	var msgs []*Message
	// First pass: collect all messages
	var all []*Message
	for i, line := range lines {
		msg, err := ParseLine(line, startLine+i)
		if err != nil || msg == nil {
			continue
		}
		all = append(all, msg)
	}

	// Second pass: for each client.request, find the matching rpc.send
	// (same timestamp + method) and copy the server name over. Then drop
	// the client.request entry.
	serverByKey := make(map[string]string) // "timestamp|method" -> server
	for _, m := range all {
		if m.Type == "client.request" && m.Server != "" {
			key := m.Timestamp + "|" + m.Method
			serverByKey[key] = m.Server
		}
	}
	for _, m := range all {
		if m.Type == "client.request" {
			continue // skip — deduped
		}
		if m.Server == "" && m.Method != "" {
			key := m.Timestamp + "|" + m.Method
			if srv, ok := serverByKey[key]; ok {
				m.Server = srv
			}
		}
		msgs = append(msgs, m)
	}
	return msgs
}

// unquote removes surrounding double quotes if present.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		unq, err := strconv.Unquote(s)
		if err != nil {
			return s[1 : len(s)-1]
		}
		return unq
	}
	return s
}

// Ensure strconv and fmt are used
var _ = fmt.Sprintf
var _ = strconv.Itoa
```

- [ ] **Step 4: Run tests**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/parser/ -v
```
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: add log line parser with server/method/id extraction"
```

---

### Task 4: File Watcher

**Files:**
- Create: `internal/watcher/watcher.go`
- Create: `internal/watcher/watcher_test.go`

- [ ] **Step 1: Add fsnotify dependency**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go get github.com/fsnotify/fsnotify
```

- [ ] **Step 2: Write failing test for file watcher**

Create `internal/watcher/watcher_test.go`:

```go
package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcher_InitialRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	content := "[START][2026-04-08 10:00:00] LSP logging initiated\n" +
		`[DEBUG][2026-04-08 10:00:01] ...lsp/log.lua:151	"rpc.send"	{ id = 1, jsonrpc = "2.0", method = "initialize", params = {} }` + "\n"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	lines := w.ReadAll()
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestWatcher_IncrementalRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	content := "[START][2026-04-08 10:00:00] LSP logging initiated\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	lines := w.ReadAll()
	if len(lines) != 1 {
		t.Fatalf("initial: expected 1 line, got %d", len(lines))
	}

	// Append new content
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	newLine := `[DEBUG][2026-04-08 10:00:02] ...lsp/log.lua:151	"rpc.send"	{ id = 2, jsonrpc = "2.0", method = "shutdown" }` + "\n"
	f.WriteString(newLine)
	f.Close()

	// Start watching and wait for the event
	ch := w.Watch()
	select {
	case newLines := <-ch:
		if len(newLines) != 1 {
			t.Fatalf("incremental: expected 1 new line, got %d", len(newLines))
		}
		if newLines[0] != newLine[:len(newLine)-1] { // trim newline
			t.Errorf("got %q", newLines[0])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for file change")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/watcher/ -v
```
Expected: Compilation error — `New`, `ReadAll`, `Watch`, `Close` not defined.

- [ ] **Step 4: Implement the file watcher**

Create `internal/watcher/watcher.go`:

```go
package watcher

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a log file for appended content.
type Watcher struct {
	path    string
	offset  int64
	mu      sync.Mutex
	fsw     *fsnotify.Watcher
	done    chan struct{}
}

// New creates a Watcher for the given file path.
func New(path string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the directory containing the file (more reliable for file writes on some OS)
	dir := path
	if idx := strings.LastIndexAny(path, "/\\"); idx >= 0 {
		dir = path[:idx]
	}
	if err := fsw.Add(dir); err != nil {
		fsw.Close()
		return nil, err
	}

	return &Watcher{
		path: path,
		fsw:  fsw,
		done: make(chan struct{}),
	}, nil
}

// ReadAll reads all current lines from the file and advances the offset.
func (w *Watcher) ReadAll() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.Open(w.path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Record file size as offset
	info, err := f.Stat()
	if err == nil {
		w.offset = info.Size()
	}

	return lines
}

// Watch starts watching for file changes and returns a channel that receives
// slices of new lines when the file grows.
func (w *Watcher) Watch() <-chan []string {
	ch := make(chan []string, 16)

	go func() {
		defer close(ch)
		for {
			select {
			case event, ok := <-w.fsw.Events:
				if !ok {
					return
				}
				// Normalize paths for comparison
				if !w.isSameFile(event.Name) {
					continue
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					newLines := w.readNew()
					if len(newLines) > 0 {
						select {
						case ch <- newLines:
						case <-w.done:
							return
						}
					}
				}
			case _, ok := <-w.fsw.Errors:
				if !ok {
					return
				}
			case <-w.done:
				return
			}
		}
	}()

	return ch
}

// readNew reads bytes from the last known offset to the current end of file.
func (w *Watcher) readNew() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.Open(w.path)
	if err != nil {
		return nil
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil
	}

	// File was truncated (e.g., log rotation)
	if info.Size() < w.offset {
		w.offset = 0
	}

	if info.Size() <= w.offset {
		return nil
	}

	if _, err := f.Seek(w.offset, 0); err != nil {
		return nil
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	w.offset = info.Size()
	return lines
}

func (w *Watcher) isSameFile(eventPath string) bool {
	// Normalize separators for Windows
	a := strings.ReplaceAll(w.path, "\\", "/")
	b := strings.ReplaceAll(eventPath, "\\", "/")
	return strings.EqualFold(a, b)
}

// Close stops watching and cleans up.
func (w *Watcher) Close() error {
	close(w.done)
	return w.fsw.Close()
}
```

- [ ] **Step 5: Run tests**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/watcher/ -v
```
Expected: All tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/watcher/ go.mod go.sum
git commit -m "feat: add file watcher with incremental reading"
```

---

### Task 5: HTTP + WebSocket Server

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`

- [ ] **Step 1: Add gorilla/websocket dependency**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go get github.com/gorilla/websocket
```

- [ ] **Step 2: Write failing test for WebSocket server**

Create `internal/server/server_test.go`:

```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/server/ -v
```
Expected: Compilation error — `New`, `Handler`, `Broadcast` not defined.

- [ ] **Step 4: Implement the server**

Create `internal/server/server.go`:

```go
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
```

- [ ] **Step 5: Run tests**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./internal/server/ -v
```
Expected: All tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/server/ go.mod go.sum
git commit -m "feat: add HTTP + WebSocket server with broadcast"
```

---

### Task 6: Frontend — HTML Shell and CSS

**Files:**
- Create: `web/index.html`
- Create: `web/style.css`

- [ ] **Step 1: Create the HTML shell**

Create `web/index.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>LSP Inspector</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <div id="toolbar">
    <span class="brand">LSP Inspector</span>
    <span class="separator">|</span>
    <div id="server-filters"></div>
    <span class="separator">|</span>
    <input type="text" id="method-filter" placeholder="Filter by method..." />
    <span class="spacer"></span>
    <span id="status" class="status connected">● watching</span>
    <span id="msg-count" class="msg-count">0 messages</span>
  </div>

  <div id="timeline"></div>

  <div id="new-msg-indicator" class="hidden" onclick="scrollToBottom()">
    ↓ <span id="new-msg-count">0</span> new messages
  </div>

  <script src="app.js"></script>
</body>
</html>
```

- [ ] **Step 2: Create the CSS**

Create `web/style.css`:

```css
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

:root {
  --bg: #1a1a2e;
  --bg-toolbar: #16213e;
  --border: #0f3460;
  --text: #e0e0e0;
  --text-muted: #888;
  --text-dim: #555;
  --send-bg: #1a3a5c;
  --send-border: #2563eb;
  --send-text: #60a5fa;
  --recv-bg: #2d1a3a;
  --recv-border: #9333ea;
  --recv-text: #c084fc;
  --accent: #e94560;
  --green: #4ade80;
  --red: #ef4444;
  --pill-bg: #333;
  --pill-text: #aaa;
  --bubble-max-width: 700px;
}

html, body {
  height: 100%;
  overflow: hidden;
  background: var(--bg);
  color: var(--text);
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 13px;
}

/* Toolbar */
#toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: var(--bg-toolbar);
  border-bottom: 1px solid var(--border);
  height: 42px;
  flex-shrink: 0;
}

.brand {
  color: var(--accent);
  font-weight: bold;
  font-size: 15px;
}

.separator {
  color: var(--text-dim);
  font-size: 12px;
}

#server-filters {
  display: flex;
  gap: 6px;
}

.server-pill {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  cursor: pointer;
  background: var(--pill-bg);
  color: var(--pill-text);
  border: none;
  transition: background 0.15s, color 0.15s;
}

.server-pill.active {
  background: var(--accent);
  color: white;
}

#method-filter {
  background: var(--border);
  border: 1px solid var(--pill-bg);
  color: var(--text);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-family: inherit;
  width: 180px;
}

#method-filter::placeholder {
  color: var(--text-dim);
}

.spacer {
  flex: 1;
}

.status {
  font-size: 11px;
}

.status.connected {
  color: var(--green);
}

.status.disconnected {
  color: var(--red);
}

.msg-count {
  color: var(--text-dim);
  font-size: 11px;
}

/* Timeline */
#timeline {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  height: calc(100vh - 42px);
}

/* System messages (centered pills) */
.msg-system {
  text-align: center;
  padding: 4px 0;
}

.msg-system .pill {
  background: var(--border);
  color: var(--text-muted);
  padding: 2px 12px;
  border-radius: 10px;
  font-size: 11px;
  display: inline-block;
}

.msg-system .pill.start-rpc {
  background: #1a3a1a;
  color: var(--green);
}

/* Message rows */
.msg-row {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  margin-bottom: 6px;
}

.msg-row.send {
  justify-content: flex-start;
}

.msg-row.recv {
  justify-content: flex-end;
}

.msg-timestamp {
  min-width: 56px;
  color: var(--text-dim);
  font-size: 11px;
  padding-top: 6px;
  flex-shrink: 0;
}

.msg-row.send .msg-timestamp {
  text-align: right;
}

.msg-row.recv .msg-timestamp {
  text-align: left;
}

/* Bubbles */
.bubble {
  display: inline-block;
  max-width: var(--bubble-max-width);
  border-radius: 8px;
  padding: 8px 12px;
}

.bubble.send {
  background: var(--send-bg);
  border: 1px solid var(--send-border);
}

.bubble.recv {
  background: var(--recv-bg);
  border: 1px solid var(--recv-border);
}

.bubble.collapsed {
  background: var(--bg);
  border: 1px dashed var(--text-dim);
}

.bubble-header {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.badge {
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: bold;
  color: white;
}

.badge.send {
  background: var(--send-border);
}

.badge.recv {
  background: var(--recv-border);
}

.method-name {
  font-size: 12px;
  font-weight: 500;
}

.method-name.send {
  color: var(--send-text);
}

.method-name.recv {
  color: var(--recv-text);
}

.msg-id {
  color: var(--text-dim);
  font-size: 10px;
}

.msg-server {
  color: #444;
  font-size: 10px;
  font-style: italic;
}

.msg-category {
  color: var(--text-dim);
  font-size: 10px;
}

.size-badge {
  background: var(--pill-bg);
  color: var(--text-muted);
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 9px;
}

.count-badge {
  background: var(--accent);
  color: white;
  padding: 1px 6px;
  border-radius: 8px;
  font-size: 10px;
  font-weight: bold;
}

/* Payload */
.payload-toggle {
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
  margin-top: 4px;
  user-select: none;
}

.payload-toggle:hover {
  color: var(--text);
}

.payload-preview {
  color: var(--pill-text);
}

.payload-expanded {
  display: none;
  margin-top: 6px;
  padding: 8px;
  background: rgba(0, 0, 0, 0.3);
  border-radius: 4px;
  overflow-x: auto;
  font-size: 12px;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
}

.payload-expanded.open {
  display: block;
}

/* JSON syntax highlighting */
.json-key { color: #60a5fa; }
.json-string { color: #4ade80; }
.json-number { color: #f59e0b; }
.json-bool { color: #e94560; }
.json-null { color: #888; }

/* Response link */
.response-link {
  color: var(--text-dim);
  font-size: 10px;
  margin-top: 4px;
}

/* New messages indicator */
#new-msg-indicator {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--accent);
  color: white;
  padding: 6px 20px;
  border-radius: 16px;
  font-size: 12px;
  cursor: pointer;
  box-shadow: 0 2px 12px rgba(233, 69, 96, 0.4);
  z-index: 100;
  transition: opacity 0.2s;
}

#new-msg-indicator.hidden {
  opacity: 0;
  pointer-events: none;
}

#new-msg-indicator:hover {
  box-shadow: 0 2px 16px rgba(233, 69, 96, 0.6);
}
```

- [ ] **Step 3: Verify files exist**

Run:
```bash
ls -la C:/Users/arbo/Documents/source/repos/lsp-inspector/web/
```
Expected: `index.html` and `style.css` listed.

- [ ] **Step 4: Commit**

```bash
git add web/index.html web/style.css
git commit -m "feat: add frontend HTML shell and dark theme CSS"
```

---

### Task 7: Frontend — JavaScript Application

**Files:**
- Create: `web/app.js`

This is the main frontend logic: WebSocket client, message rendering, filtering, collapsing, and auto-scroll behavior.

- [ ] **Step 1: Create app.js**

Create `web/app.js`:

```javascript
(function () {
  "use strict";

  // --- State ---
  let allMessages = [];
  let serverFilter = "all";
  let methodFilter = "";
  let newMsgCount = 0;
  let isAtBottom = true;

  // --- DOM refs ---
  const timeline = document.getElementById("timeline");
  const serverFiltersEl = document.getElementById("server-filters");
  const methodFilterEl = document.getElementById("method-filter");
  const statusEl = document.getElementById("status");
  const msgCountEl = document.getElementById("msg-count");
  const newMsgIndicator = document.getElementById("new-msg-indicator");
  const newMsgCountEl = document.getElementById("new-msg-count");

  // --- WebSocket ---
  let ws;
  let reconnectTimer;

  function connect() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      statusEl.textContent = "● watching";
      statusEl.className = "status connected";
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
    };

    ws.onmessage = function (evt) {
      const data = JSON.parse(evt.data);
      if (data.type === "initial") {
        allMessages = data.messages || [];
        renderAll();
      } else if (data.type === "append") {
        const newMsgs = data.messages || [];
        allMessages = allMessages.concat(newMsgs);
        appendRendered(newMsgs);
      }
      updateMsgCount();
    };

    ws.onclose = function () {
      statusEl.textContent = "● disconnected";
      statusEl.className = "status disconnected";
      reconnectTimer = setTimeout(connect, 2000);
    };

    ws.onerror = function () {
      ws.close();
    };
  }

  // --- Rendering ---

  function renderAll() {
    timeline.innerHTML = "";
    newMsgCount = 0;
    hideNewMsgIndicator();
    const groups = collapseConsecutive(filterMessages(allMessages));
    groups.forEach(function (group) {
      timeline.appendChild(renderGroup(group));
    });
    scrollToBottom();
  }

  function appendRendered(msgs) {
    const visible = filterMessages(msgs);
    if (visible.length === 0) return;

    // Try to merge with existing collapsed group at the end
    const groups = collapseConsecutive(visible);
    groups.forEach(function (group) {
      timeline.appendChild(renderGroup(group));
    });

    if (isAtBottom) {
      scrollToBottom();
    } else {
      newMsgCount += visible.length;
      showNewMsgIndicator();
    }
  }

  function filterMessages(msgs) {
    return msgs.filter(function (m) {
      if (serverFilter !== "all" && m.server !== serverFilter && m.direction !== "info") {
        return false;
      }
      if (methodFilter && m.method && m.method.toLowerCase().indexOf(methodFilter.toLowerCase()) === -1) {
        return false;
      }
      return true;
    });
  }

  // Collapse consecutive messages with same method + direction
  function collapseConsecutive(msgs) {
    var groups = [];
    var i = 0;
    while (i < msgs.length) {
      var current = msgs[i];
      // Only collapse rpc messages with a method
      if (current.method && (current.direction === "send" || current.direction === "receive")) {
        var run = [current];
        var j = i + 1;
        while (j < msgs.length && msgs[j].method === current.method && msgs[j].direction === current.direction) {
          run.push(msgs[j]);
          j++;
        }
        if (run.length > 1) {
          groups.push({ type: "collapsed", messages: run, method: current.method, direction: current.direction });
          i = j;
          continue;
        }
      }
      groups.push({ type: "single", message: current });
      i++;
    }
    return groups;
  }

  function renderGroup(group) {
    if (group.type === "collapsed") {
      return renderCollapsed(group);
    }
    return renderMessage(group.message);
  }

  function renderMessage(msg) {
    if (msg.direction === "info") {
      return renderSystemMessage(msg);
    }

    var row = document.createElement("div");
    row.className = "msg-row " + msg.direction;

    var ts = document.createElement("div");
    ts.className = "msg-timestamp";
    ts.textContent = formatTime(msg.timestamp);

    var bubble = document.createElement("div");
    bubble.className = "bubble " + msg.direction;

    // Header
    var header = document.createElement("div");
    header.className = "bubble-header";

    var badge = document.createElement("span");
    badge.className = "badge " + msg.direction;
    badge.textContent = msg.direction === "send" ? "SEND" : "RECV";
    header.appendChild(badge);

    if (msg.method) {
      var methodEl = document.createElement("span");
      methodEl.className = "method-name " + msg.direction;
      methodEl.textContent = msg.method;
      header.appendChild(methodEl);
    }

    if (msg.id !== null && msg.id !== undefined) {
      var idEl = document.createElement("span");
      idEl.className = "msg-id";
      idEl.textContent = "id:" + msg.id;
      header.appendChild(idEl);
    } else if (msg.method && !msg.method.startsWith("$")) {
      // No id = notification (unless it's a response)
      var catEl = document.createElement("span");
      catEl.className = "msg-category";
      catEl.textContent = "(notification)";
      header.appendChild(catEl);
    }

    if (msg.server) {
      var serverEl = document.createElement("span");
      serverEl.className = "msg-server";
      serverEl.textContent = msg.server;
      header.appendChild(serverEl);
    }

    // Size badge for large payloads
    var payloadStr = msg.payload ? JSON.stringify(msg.payload) : msg.rawPayload || "";
    if (payloadStr.length > 500) {
      var sizeEl = document.createElement("span");
      sizeEl.className = "size-badge";
      sizeEl.textContent = formatSize(payloadStr.length);
      header.appendChild(sizeEl);
    }

    bubble.appendChild(header);

    // Payload
    if (payloadStr && payloadStr !== "{}" && payloadStr !== "null") {
      var payloadToggle = document.createElement("div");
      payloadToggle.className = "payload-toggle";

      var preview = makePreview(msg.payload || msg.rawPayload);
      payloadToggle.innerHTML = "▶ <span class='payload-preview'>" + escapeHtml(preview) + "</span>";

      var expanded = document.createElement("div");
      expanded.className = "payload-expanded";
      if (msg.payload) {
        expanded.innerHTML = syntaxHighlight(msg.payload);
      } else {
        expanded.textContent = msg.rawPayload;
      }

      payloadToggle.onclick = function () {
        var isOpen = expanded.classList.toggle("open");
        payloadToggle.innerHTML = (isOpen ? "▼" : "▶") + " <span class='payload-preview'>" + escapeHtml(preview) + "</span>";
      };

      bubble.appendChild(payloadToggle);
      bubble.appendChild(expanded);
    }

    // Response link
    if (msg.direction === "receive" && msg.id !== null && msg.id !== undefined && !msg.method) {
      var linkEl = document.createElement("div");
      linkEl.className = "response-link";
      var reqMsg = findRequest(msg.id);
      var elapsed = "";
      if (reqMsg) {
        elapsed = " · " + calcElapsed(reqMsg.timestamp, msg.timestamp);
      }
      linkEl.textContent = "↩ responds to id:" + msg.id + elapsed;
      bubble.appendChild(linkEl);
    }

    if (msg.direction === "send") {
      row.appendChild(ts);
      row.appendChild(bubble);
    } else {
      row.appendChild(bubble);
      row.appendChild(ts);
    }

    return row;
  }

  function renderSystemMessage(msg) {
    var div = document.createElement("div");
    div.className = "msg-system";
    var pill = document.createElement("span");
    pill.className = "pill";

    var text = formatTime(msg.timestamp);
    if (msg.level === "START") {
      text += " — LSP logging initiated";
    } else if (msg.rawPayload && msg.rawPayload.indexOf && msg.rawPayload.indexOf("cmd") >= 0) {
      pill.className = "pill start-rpc";
      // Try to extract server name from payload
      var serverName = msg.server || "unknown";
      text += " — Starting RPC client: " + serverName;
    } else if (msg.type === "info" && msg.server) {
      text += " — " + msg.server + " info";
    } else {
      text += " — " + (msg.type || "info");
    }

    pill.textContent = text;
    div.appendChild(pill);
    return div;
  }

  function renderCollapsed(group) {
    var row = document.createElement("div");
    row.className = "msg-row " + group.direction;

    var ts = document.createElement("div");
    ts.className = "msg-timestamp";
    ts.textContent = formatTime(group.messages[0].timestamp);

    var bubble = document.createElement("div");
    bubble.className = "bubble collapsed";

    var header = document.createElement("div");
    header.className = "bubble-header";

    var badge = document.createElement("span");
    badge.className = "badge " + group.direction;
    badge.textContent = group.direction === "send" ? "SEND" : "RECV";
    header.appendChild(badge);

    var methodEl = document.createElement("span");
    methodEl.className = "method-name " + group.direction;
    methodEl.textContent = group.method;
    header.appendChild(methodEl);

    var countEl = document.createElement("span");
    countEl.className = "count-badge";
    countEl.textContent = "×" + group.messages.length;
    header.appendChild(countEl);

    if (group.messages[0].server) {
      var serverEl = document.createElement("span");
      serverEl.className = "msg-server";
      serverEl.textContent = group.messages[0].server;
      header.appendChild(serverEl);
    }

    bubble.appendChild(header);

    var expandHint = document.createElement("div");
    expandHint.className = "payload-toggle";
    expandHint.textContent = "▶ click to expand all";

    var expandedContainer = document.createElement("div");
    expandedContainer.className = "payload-expanded";
    expandedContainer.style.display = "none";

    expandHint.onclick = function () {
      var isOpen = expandedContainer.style.display !== "none";
      if (isOpen) {
        expandedContainer.style.display = "none";
        expandHint.textContent = "▶ click to expand all";
      } else {
        expandedContainer.innerHTML = "";
        group.messages.forEach(function (m) {
          expandedContainer.appendChild(renderMessage(m));
        });
        expandedContainer.style.display = "block";
        expandHint.textContent = "▼ collapse";
      }
    };

    bubble.appendChild(expandHint);
    bubble.appendChild(expandedContainer);

    if (group.direction === "send") {
      row.appendChild(ts);
      row.appendChild(bubble);
    } else {
      row.appendChild(bubble);
      row.appendChild(ts);
    }

    return row;
  }

  // --- Helpers ---

  function formatTime(ts) {
    if (!ts) return "";
    // "2026-04-08 10:19:30" -> "10:19:30"
    var parts = ts.split(" ");
    return parts.length > 1 ? parts[1] : ts;
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + " B";
    return (bytes / 1024).toFixed(1) + " KB";
  }

  function makePreview(payload) {
    var str;
    if (typeof payload === "string") {
      str = payload;
    } else {
      str = JSON.stringify(payload);
    }
    if (str.length > 120) {
      return str.substring(0, 120) + "...";
    }
    return str;
  }

  function escapeHtml(s) {
    var div = document.createElement("div");
    div.textContent = s;
    return div.innerHTML;
  }

  function syntaxHighlight(payload) {
    var json;
    if (typeof payload === "string") {
      json = payload;
    } else {
      json = JSON.stringify(payload, null, 2);
    }
    return json.replace(
      /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
      function (match) {
        var cls = "json-number";
        if (/^"/.test(match)) {
          if (/:$/.test(match)) {
            cls = "json-key";
            // Remove the colon for styling, add it back
            return '<span class="' + cls + '">' + escapeHtml(match.slice(0, -1)) + "</span>:";
          } else {
            cls = "json-string";
          }
        } else if (/true|false/.test(match)) {
          cls = "json-bool";
        } else if (/null/.test(match)) {
          cls = "json-null";
        }
        return '<span class="' + cls + '">' + escapeHtml(match) + "</span>";
      }
    );
  }

  function findRequest(id) {
    for (var i = allMessages.length - 1; i >= 0; i--) {
      var m = allMessages[i];
      if (m.id === id && m.direction === "send") {
        return m;
      }
    }
    return null;
  }

  function calcElapsed(ts1, ts2) {
    var d1 = new Date("2000-01-01 " + ts1.split(" ").pop());
    var d2 = new Date("2000-01-01 " + ts2.split(" ").pop());
    var ms = d2 - d1;
    if (isNaN(ms) || ms < 0) return "";
    if (ms < 1) return "<1ms";
    if (ms < 1000) return ms + "ms";
    return (ms / 1000).toFixed(1) + "s";
  }

  // --- Scroll tracking ---

  timeline.addEventListener("scroll", function () {
    var threshold = 50;
    isAtBottom = timeline.scrollHeight - timeline.scrollTop - timeline.clientHeight < threshold;
    if (isAtBottom) {
      newMsgCount = 0;
      hideNewMsgIndicator();
    }
  });

  // Exported for onclick
  window.scrollToBottom = function () {
    timeline.scrollTop = timeline.scrollHeight;
    newMsgCount = 0;
    hideNewMsgIndicator();
  };

  function showNewMsgIndicator() {
    newMsgCountEl.textContent = newMsgCount;
    newMsgIndicator.classList.remove("hidden");
  }

  function hideNewMsgIndicator() {
    newMsgIndicator.classList.add("hidden");
  }

  // --- Server filter pills ---

  function updateServerFilters() {
    var servers = new Set();
    allMessages.forEach(function (m) {
      if (m.server) servers.add(m.server);
    });

    serverFiltersEl.innerHTML = "";

    var allPill = document.createElement("button");
    allPill.className = "server-pill" + (serverFilter === "all" ? " active" : "");
    allPill.textContent = "All";
    allPill.onclick = function () {
      serverFilter = "all";
      renderAll();
      updateServerFilters();
    };
    serverFiltersEl.appendChild(allPill);

    servers.forEach(function (name) {
      var pill = document.createElement("button");
      pill.className = "server-pill" + (serverFilter === name ? " active" : "");
      pill.textContent = name;
      pill.onclick = function () {
        serverFilter = name;
        renderAll();
        updateServerFilters();
      };
      serverFiltersEl.appendChild(pill);
    });
  }

  function updateMsgCount() {
    msgCountEl.textContent = allMessages.length + " messages";
    updateServerFilters();
  }

  // --- Method filter ---

  var filterTimeout;
  methodFilterEl.addEventListener("input", function () {
    clearTimeout(filterTimeout);
    filterTimeout = setTimeout(function () {
      methodFilter = methodFilterEl.value;
      renderAll();
    }, 200);
  });

  function scrollToBottom() {
    timeline.scrollTop = timeline.scrollHeight;
  }

  // --- Init ---
  connect();
})();
```

- [ ] **Step 2: Verify the file was created**

Run:
```bash
ls -la C:/Users/arbo/Documents/source/repos/lsp-inspector/web/app.js
```
Expected: File exists.

- [ ] **Step 3: Commit**

```bash
git add web/app.js
git commit -m "feat: add frontend JS with timeline rendering, filtering, and WebSocket"
```

---

### Task 8: CLI Entry Point — Wire Everything Together

**Files:**
- Create: `cmd/lsp-inspector/main.go`

- [ ] **Step 1: Create main.go**

Create `cmd/lsp-inspector/main.go`:

```go
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/arbo/lsp-inspector/internal/parser"
	"github.com/arbo/lsp-inspector/internal/server"
	"github.com/arbo/lsp-inspector/internal/watcher"
)

//go:embed ../../web/*
var webFS embed.FS

func main() {
	port := flag.Int("port", 0, "HTTP port (default: random available port)")
	noOpen := flag.Bool("no-open", false, "Don't auto-open browser")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lsp-inspector [flags] <logfile>\n\nArguments:\n  logfile    Path to Neovim lsp.log file\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	logFile := flag.Arg(0)

	// Verify file exists
	if _, err := os.Stat(logFile); err != nil {
		log.Fatalf("Cannot access log file: %v", err)
	}

	// Set up file watcher
	w, err := watcher.New(logFile)
	if err != nil {
		log.Fatalf("Failed to watch file: %v", err)
	}
	defer w.Close()

	// Read and parse initial content
	lines := w.ReadAll()
	initialMsgs := parser.ParseLines(lines, 0)
	log.Printf("Parsed %d messages from %s", len(initialMsgs), logFile)

	// Create server with sub-filesystem rooted at "web"
	webSubFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to create sub-filesystem: %v", err)
	}
	srv := server.New(webSubFS, initialMsgs)

	// Start file watching
	changes := w.Watch()
	go func() {
		lineOffset := len(lines)
		for newLines := range changes {
			msgs := parser.ParseLines(newLines, lineOffset)
			lineOffset += len(newLines)
			if len(msgs) > 0 {
				srv.Broadcast(msgs)
			}
		}
	}()

	// Find available port
	addr := fmt.Sprintf(":%d", *port)
	if *port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal(err)
		}
		addr = listener.Addr().String()
		listener.Close()
	}

	url := fmt.Sprintf("http://localhost%s", addr)
	log.Printf("Serving on %s", url)

	// Open browser
	if !*noOpen {
		openBrowser(url)
	}

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}
```

- [ ] **Step 2: Fix the import — add `io/fs` import**

The `fs.Sub` call requires `io/fs`. Update the imports in `main.go`:

```go
import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/arbo/lsp-inspector/internal/parser"
	"github.com/arbo/lsp-inspector/internal/server"
	"github.com/arbo/lsp-inspector/internal/watcher"
)
```

- [ ] **Step 3: Verify it compiles**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go build ./cmd/lsp-inspector/
```
Expected: No errors. Binary `lsp-inspector.exe` created.

- [ ] **Step 4: Test with the real log file**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go run ./cmd/lsp-inspector/ --no-open "C:/Users/arbo/AppData/Local/nvim-data/lsp.log"
```
Expected: Prints "Parsed N messages" and "Serving on http://localhost:XXXXX". Open that URL manually and verify the timeline renders.

- [ ] **Step 5: Commit**

```bash
git add cmd/lsp-inspector/main.go
git commit -m "feat: add CLI entry point wiring parser, watcher, and server"
```

---

### Task 9: Integration Test — End to End

**Files:**
- Create: `cmd/lsp-inspector/main_test.go`

- [ ] **Step 1: Write an integration test**

Create `cmd/lsp-inspector/main_test.go`:

```go
package main

import (
	"encoding/json"
	"io/fs"
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

	_ "embed"
)

func TestEndToEnd(t *testing.T) {
	// Create a temp log file with realistic content
	dir := t.TempDir()
	logPath := filepath.Join(dir, "lsp.log")

	content := `[START][2026-04-08 10:19:43] LSP logging initiated
[DEBUG][2026-04-08 10:19:51] ...lsp/log.lua:151	"rpc.send"	{ id = 1, jsonrpc = "2.0", method = "initialize", params = { capabilities = {} } }
[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.receive"	{ id = 1, jsonrpc = "2.0", result = { capabilities = { hoverProvider = true } } }
[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.send"	{ jsonrpc = "2.0", method = "initialized", params = vim.empty_dict() }
[DEBUG][2026-04-08 10:19:52] ...lsp/log.lua:151	"rpc.receive"	{ jsonrpc = "2.0", method = "al/projectsLoadedNotification", params = { projects = { "c:/test" } } }
[DEBUG][2026-04-08 10:19:53] ...lsp/log.lua:151	"rpc.receive"	{ jsonrpc = "2.0", method = "al/projectsLoadedNotification", params = { projects = { "c:/test" } } }
[DEBUG][2026-04-08 10:19:54] ...lsp/log.lua:151	"rpc.receive"	{ jsonrpc = "2.0", method = "al/projectsLoadedNotification", params = { projects = { "c:/test" } } }
`
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse
	w, err := watcher.New(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	lines := w.ReadAll()
	msgs := parser.ParseLines(lines, 0)

	if len(msgs) != 7 {
		t.Fatalf("expected 7 messages, got %d", len(msgs))
	}

	// Verify specific messages
	if msgs[0].Level != "START" {
		t.Errorf("msg[0]: expected START, got %s", msgs[0].Level)
	}
	if msgs[1].Method != "initialize" {
		t.Errorf("msg[1]: expected initialize, got %s", msgs[1].Method)
	}
	if msgs[1].ID == nil || *msgs[1].ID != 1 {
		t.Errorf("msg[1]: expected id=1, got %v", msgs[1].ID)
	}
	if msgs[2].Direction != "receive" {
		t.Errorf("msg[2]: expected receive, got %s", msgs[2].Direction)
	}
	// Notification has no id
	if msgs[3].Method != "initialized" {
		t.Errorf("msg[3]: expected initialized, got %s", msgs[3].Method)
	}
	if msgs[3].ID != nil {
		t.Errorf("msg[3]: expected nil id for notification, got %v", msgs[3].ID)
	}

	// Verify payload is valid JSON
	for i, m := range msgs {
		if m.Payload != nil {
			var v interface{}
			if err := json.Unmarshal(m.Payload, &v); err != nil {
				t.Errorf("msg[%d] payload is not valid JSON: %v", i, err)
			}
		}
	}

	// Test WebSocket serving
	staticFS := webFS
	subFS, _ := fs.Sub(staticFS, "web")
	srv := server.New(subFS, msgs)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Connect via WebSocket
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
		t.Errorf("expected initial, got %s", envelope.Type)
	}
	if len(envelope.Messages) != 7 {
		t.Errorf("expected 7 messages via WS, got %d", len(envelope.Messages))
	}

	// Verify HTTP serves the page
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("HTTP status: got %d, want 200", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run the integration test**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./cmd/lsp-inspector/ -v -run TestEndToEnd
```
Expected: PASS.

- [ ] **Step 3: Run all tests**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go test ./... -v
```
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/lsp-inspector/main_test.go
git commit -m "test: add end-to-end integration test"
```

---

### Task 10: Manual Smoke Test and Polish

**Files:**
- Modify: any files needing fixes found during manual testing

- [ ] **Step 1: Build the binary**

Run:
```bash
cd C:/Users/arbo/Documents/source/repos/lsp-inspector
go build -o lsp-inspector.exe ./cmd/lsp-inspector/
```

- [ ] **Step 2: Run against the real log file**

Run:
```bash
./lsp-inspector.exe "C:/Users/arbo/AppData/Local/nvim-data/lsp.log"
```
Expected: Browser opens. Verify:
- Timeline shows messages with SEND (left, blue) and RECV (right, purple)
- System messages are centered
- Duplicate `al/projectsLoadedNotification` entries are collapsed with count badge
- Payloads are collapsed, clickable to expand
- JSON is syntax-highlighted when expanded
- Server filter pills appear in toolbar (al_ls, GitHub Copilot)
- Method filter works
- Timestamps display correctly

- [ ] **Step 3: Test live watching**

In another terminal, trigger some LSP activity in Neovim (open a file, trigger hover, etc.) and verify new messages appear in the browser. If already at bottom, it should auto-scroll. If scrolled up, the floating "N new messages" pill should appear.

- [ ] **Step 4: Fix any issues found, commit**

```bash
git add -A
git commit -m "fix: polish from manual smoke testing"
```

- [ ] **Step 5: Add .gitignore**

Create `.gitignore`:
```
lsp-inspector.exe
lsp-inspector
.superpowers/
```

```bash
git add .gitignore
git commit -m "chore: add .gitignore"
```
