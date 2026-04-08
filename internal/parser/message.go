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
