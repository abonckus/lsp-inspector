# LSP Inspector - Design Spec

A terminal-launched web app that parses Neovim's `lsp.log` file and displays LSP client/server communication in a chat-style timeline with live file watching.

## Usage

```bash
lsp-inspector /path/to/lsp.log
# Opens browser to http://localhost:<port>
# Watches file for changes, streams new entries via WebSocket
```

## Architecture

Single Go binary with embedded static assets (`go:embed`). No external dependencies.

```
lsp.log ──(fsnotify)──> Go server ──(WebSocket)──> Browser UI
                            |
                       HTTP serves embedded HTML/CSS/JS
```

### Components

**Go backend (`cmd/lsp-inspector/main.go`):**
- CLI entry point: accepts log file path, optional `--port` flag
- Parses entire log file on startup, streams new entries on file change
- Serves embedded static assets over HTTP
- WebSocket endpoint pushes parsed messages to the browser
- Opens default browser on startup

**Log parser (`internal/parser/`):**
- Parses Neovim lsp.log format: `[LEVEL][timestamp] source\t"message_type"\t{lua_table_payload}`
- Converts Lua table syntax to JSON where possible (key = value -> "key": value, vim.NIL -> null, vim.empty_dict() -> {})
- Handles non-convertible Lua values gracefully (`<function N>` -> string `"<function N>"`, `<table N>` -> string `"<table N>"`)
- Extracts structured fields: level, timestamp, source, message type (rpc.send, rpc.receive, client.request, etc.), method name, id, server name, payload

**File watcher (`internal/watcher/`):**
- Uses fsnotify to watch the log file for writes
- On change, reads only new bytes from last known offset
- Parses new lines and pushes them through a channel to the WebSocket handler

**HTTP + WebSocket server (`internal/server/`):**
- Serves embedded static assets from `web/`
- WebSocket endpoint at `/ws`
- On connect: sends all existing parsed messages as initial batch
- On file change: pushes new messages incrementally
- JSON protocol: `{"type": "initial", "messages": [...]}` and `{"type": "append", "messages": [...]}`

**Frontend (`web/`):**
- Single-page vanilla HTML/CSS/JS, no framework
- Embedded into the Go binary via `go:embed`

## Log Format

Each log line follows this pattern:

```
[LEVEL][YYYY-MM-DD HH:MM:SS] ...truncated/source/path.lua:NNN\t"message_type"\t{lua_table_payload}
```

### Message types identified

| Log field | Direction | Has ID | Description |
|---|---|---|---|
| `"rpc.send"` | Client -> Server | Sometimes | Raw JSON-RPC message sent |
| `"rpc.receive"` | Server -> Client | Sometimes | Raw JSON-RPC message received |
| `"LSP[name]" "client.request"` | Client -> Server | Yes | Higher-level request log (duplicate of rpc.send) |
| `"LSP[name]" "server_capabilities"` | Info | No | Capabilities summary after initialize |
| `"Starting RPC client"` | Info | No | Server startup |
| `"exit_handler"` | Info | No | Server shutdown (huge payload, contains full client state) |
| `[START]` | Info | No | Log session start marker |

### LSP message categories (from JSON-RPC content)

- **Request** (has `id` + `method`): expects a response. E.g., `initialize`, `textDocument/documentHighlight`
- **Response** (has `id` + `result` or `error`, no `method`): answers a request
- **Notification** (has `method`, no `id`): fire-and-forget. E.g., `initialized`, `textDocument/didOpen`, `al/projectsLoadedNotification`

## Lua Table to JSON Conversion

The parser converts Lua table syntax to JSON on a best-effort basis:

| Lua | JSON |
|---|---|
| `{ key = value }` | `{ "key": value }` |
| `{ "string" }` (array) | `["string"]` |
| `vim.NIL` | `null` |
| `vim.empty_dict()` | `{}` |
| `true` / `false` | `true` / `false` |
| `"string"` / `'string'` | `"string"` |
| `123` / `1.5` | `123` / `1.5` |
| `["key with spaces"]` = value | `"key with spaces": value` |
| `<function N>` | `"<function N>"` (string) |
| `<table N>` | `"<table N>"` (string) |
| `<N>{ ... }` (back-reference with inline) | Parse the inline table |

## Frontend Design

### Layout

Full-height app layout, no scrolling on the page itself:

```
+------------------------------------------------------------------+
| LSP Inspector  | [All] [al_ls] [Copilot] | Filter... | * watching |
+------------------------------------------------------------------+
|                                                                  |
|  10:19:43 — LSP logging initiated                                |
|                                                                  |
|  10:19:51  [SEND] initialize  id:1  al_ls                       |
|            > { capabilities, clientInfo, ... }                   |
|                                                                  |
|            [RECV] initialize response  id:1  al_ls    10:19:52   |
|            > { capabilities: { ... } }                           |
|            responds to id:1 - 1.0s                               |
|                                                                  |
|            [RECV] al/projectsLoadedNotification x15   10:19:22   |
|            > click to expand                                     |
|                                                                  |
|  (scrollable timeline, full viewport height)                     |
|                                                                  |
|                    [ 3 new messages ]  <-- floating, fixed bottom |
+------------------------------------------------------------------+
```

### Visual Design

- **Dark theme** matching terminal aesthetics (background `#1a1a2e`)
- **Monospace font** (Cascadia Code / Fira Code / system monospace)
- **SEND messages**: left-aligned, blue border/badge (`#2563eb`)
- **RECV messages**: right-aligned, purple border/badge (`#9333ea`)
- **System messages** (START, Starting RPC, exit_handler): centered pills, muted colors
- **Timestamps**: outside the bubble, muted gray

### Message Bubbles

- Content-sized width with `max-width: 700px` cap
- Bubble contains: direction badge, method name, id (if present), server name
- Payloads collapsed by default, click `>` to expand inline
- Large payloads show size badge (e.g., `4.2 KB`)
- Expanded payloads rendered as formatted JSON with syntax highlighting

### Request/Response Linking

- Responses display `responds to id:N - Xs` below the payload
- Matching done by `id` field between rpc.send and rpc.receive
- Messages stay in chronological order (not grouped)

### Duplicate Collapsing

- Consecutive messages with the same method name AND same direction are collapsed into one entry
- Shows count badge (e.g., `x15`) with dashed border
- Click to expand all individual messages
- Toggle in toolbar to disable collapsing

### Server Filtering

- Toolbar shows pill per detected server name + "All" pill
- "All" is active by default, shows interleaved timeline
- Clicking a server pill filters to only that server's messages
- Server names extracted from `"LSP[name]"` entries and from the initialize request/response pairs

### Method Filtering

- Text input in toolbar for filtering by method name
- Filters as you type, matches substring against method name
- Clear button to reset

### Live Watch

- File watcher detects appended content, pushes via WebSocket
- Auto-scroll when user is at bottom of timeline
- When scrolled up: floating "N new messages" pill at bottom of viewport, click to jump down
- Toolbar shows green "watching" indicator when connected, red "disconnected" when WebSocket drops
- Auto-reconnect on WebSocket disconnect

### Deduplication of rpc.send / client.request

The log often contains both a `"LSP[name]" "client.request"` line and a `"rpc.send"` line for the same request. These are the same message logged twice. The parser deduplicates by matching: same timestamp + same method + same id. Only the `rpc.send` / `rpc.receive` entries are shown (they contain the full JSON-RPC payload). The `client.request` entries are used only to extract the server name when it's not otherwise available.

## Project Structure

```
lsp-inspector/
  cmd/lsp-inspector/
    main.go              # CLI entry, flag parsing, server startup, browser open
  internal/
    parser/
      parser.go          # Log line parsing, Lua-to-JSON conversion
      parser_test.go
      message.go         # Message type definitions
    watcher/
      watcher.go         # File watching, incremental reading
      watcher_test.go
    server/
      server.go          # HTTP server, WebSocket handler, static file serving
      server_test.go
  web/
    index.html           # Single-page app
    style.css            # Dark theme styles
    app.js               # Timeline rendering, WebSocket client, filtering, collapsing
  go.mod
  go.sum
```

## Go Dependencies

- `github.com/fsnotify/fsnotify` - file watching
- `github.com/gorilla/websocket` - WebSocket support (or `nhooyr.io/websocket` for a more modern API)
- Standard library for everything else (HTTP server, JSON, embed, flag, exec)

## Message Data Model

```go
type Message struct {
    Level      string          `json:"level"`      // DEBUG, INFO, WARN, ERROR, START
    Timestamp  string          `json:"timestamp"`  // "2026-04-08 10:19:30"
    Source     string          `json:"source"`     // truncated source path
    Type       string          `json:"type"`       // "rpc.send", "rpc.receive", "client.request", "info"
    Direction  string          `json:"direction"`  // "send", "receive", "info"
    Server     string          `json:"server"`     // "al_ls", "GitHub Copilot", ""
    Method     string          `json:"method"`     // "textDocument/documentHighlight", ""
    ID         *int            `json:"id"`         // JSON-RPC id, nil for notifications
    Payload    json.RawMessage `json:"payload"`    // JSON-converted payload
    RawPayload string          `json:"rawPayload"` // Original Lua string (fallback)
    Line       int             `json:"line"`       // Line number in log file
}
```

## WebSocket Protocol

```jsonc
// Server -> Client: initial load
{ "type": "initial", "messages": [Message, ...] }

// Server -> Client: new messages appended
{ "type": "append", "messages": [Message, ...] }
```

## CLI Interface

```
Usage: lsp-inspector [flags] <logfile>

Arguments:
  logfile    Path to Neovim lsp.log file

Flags:
  --port     HTTP port (default: random available port)
  --no-open  Don't auto-open browser
```
