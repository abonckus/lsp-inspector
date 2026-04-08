package parser

import (
	"encoding/json"
	"testing"
)

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
