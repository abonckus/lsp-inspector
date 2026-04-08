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
