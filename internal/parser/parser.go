package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Regex to match the log line header: [LEVEL][timestamp] source
var logLineRe = regexp.MustCompile(`^\[(\w+)\]\[([^\]]+)\]\s+([^\t]+)`)

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
// It auto-detects the format (Neovim lsp.log vs VS Code trace log) and delegates.
func ParseLines(lines []string, startLine int) []*Message {
	if IsVSCodeFormat(lines) {
		return parseVSCodeLines(lines, startLine)
	}
	return parseNeovimLines(lines, startLine)
}

// parseNeovimLines parses Neovim lsp.log lines.
// It deduplicates client.request entries against rpc.send entries:
// client.request is used only to extract the server name, then discarded.
func parseNeovimLines(lines []string, startLine int) []*Message {
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

	// Third pass: propagate server names from requests to their responses by ID.
	// Responses (rpc.receive with id but no method) don't carry a server name in the log.
	serverByID := make(map[int]string)
	for _, m := range msgs {
		if m.Direction == DirectionSend && m.ID != nil && m.Server != "" {
			serverByID[*m.ID] = m.Server
		}
	}
	for _, m := range msgs {
		if m.Server == "" && m.ID != nil {
			if srv, ok := serverByID[*m.ID]; ok {
				m.Server = srv
			}
		}
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

// Ensure fmt is used
var _ = fmt.Sprintf
