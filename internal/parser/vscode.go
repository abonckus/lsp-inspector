package parser

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// VS Code trace log header patterns:
//   [Trace - 10:03:59 AM] Sending request 'initialize - (0)'.
//   [Trace - 10:04:00 AM] Received response 'initialize - (0)' in 741ms.
//   [Trace - 10:04:00 AM] Sending notification 'initialized'.
//   [Trace - 10:04:11 AM] Sending response 'al/activeProjectLoaded - (50)'. Processing request took 2ms
var vscodeHeaderRe = regexp.MustCompile(
	`^\[Trace - ([^\]]+)\] (Sending|Received) (request|response|notification) '([^']+)'`,
)

// IsVSCodeFormat checks whether the lines are in VS Code trace log format
// by looking at the first non-empty line.
func IsVSCodeFormat(lines []string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return strings.HasPrefix(strings.TrimSpace(line), "[Trace")
		}
	}
	return false
}

// parseVSCodeLines parses VS Code trace log lines into Messages.
func parseVSCodeLines(lines []string, startLine int) []*Message {
	groups := groupVSCodeMessages(lines, startLine)
	var msgs []*Message
	for _, g := range groups {
		msg := parseVSCodeGroup(g)
		if msg != nil {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// lineGroup is a consecutive block of non-empty lines forming one log message.
type lineGroup struct {
	lines     []string
	startLine int // 0-based file line number of the first line
}

// groupVSCodeMessages splits lines into groups separated by blank lines.
func groupVSCodeMessages(lines []string, startLine int) []lineGroup {
	var groups []lineGroup
	var current []string
	currentStart := 0
	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(trimmed) == "" {
			if len(current) > 0 {
				groups = append(groups, lineGroup{lines: current, startLine: startLine + currentStart})
				current = nil
			}
			continue
		}
		if len(current) == 0 {
			currentStart = i
		}
		current = append(current, trimmed)
	}
	if len(current) > 0 {
		groups = append(groups, lineGroup{lines: current, startLine: startLine + currentStart})
	}
	return groups
}

// parseVSCodeGroup parses a single grouped message (header + body lines).
func parseVSCodeGroup(g lineGroup) *Message {
	if len(g.lines) == 0 {
		return nil
	}

	matches := vscodeHeaderRe.FindStringSubmatch(g.lines[0])
	if matches == nil {
		return nil
	}

	timestamp := matches[1]  // "10:03:59 AM"
	sendRecv := matches[2]   // "Sending" or "Received"
	methodRaw := matches[4]  // "initialize - (0)" or "initialized"

	msg := &Message{
		Level:     "Trace",
		Timestamp: timestamp,
		Line:      g.startLine,
	}

	// Parse method and optional ID: "method - (id)" or just "method"
	if idx := strings.LastIndex(methodRaw, " - ("); idx >= 0 {
		msg.Method = methodRaw[:idx]
		idStr := strings.TrimSuffix(methodRaw[idx+4:], ")")
		if id, err := strconv.Atoi(idStr); err == nil {
			msg.ID = &id
		}
	} else {
		msg.Method = methodRaw
	}

	// Direction: Sending → send, Received → receive
	if sendRecv == "Sending" {
		msg.Direction = DirectionSend
		msg.Type = "rpc.send"
	} else {
		msg.Direction = DirectionReceive
		msg.Type = "rpc.receive"
	}

	// Parse body (lines after header)
	if len(g.lines) > 1 {
		parseVSCodeBody(msg, g.lines[1:])
	}

	return msg
}

// parseVSCodeBody extracts the JSON payload from "Params: {…}" or "Result: {…}" body lines.
func parseVSCodeBody(msg *Message, body []string) {
	first := strings.TrimSpace(body[0])
	if first == "No result returned." || first == "No parameters provided." {
		return
	}

	// Join all body lines, strip "Params: " or "Result: " prefix
	joined := strings.Join(body, "\n")
	switch {
	case strings.HasPrefix(first, "Params: "):
		joined = strings.TrimPrefix(joined, "Params: ")
	case strings.HasPrefix(first, "Result: "):
		joined = strings.TrimPrefix(joined, "Result: ")
	}
	joined = strings.TrimSpace(joined)

	if len(joined) == 0 {
		return
	}

	var v json.RawMessage
	if json.Unmarshal([]byte(joined), &v) == nil {
		msg.Payload = v
	} else {
		msg.RawPayload = joined
	}
}
