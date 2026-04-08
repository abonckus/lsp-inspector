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
