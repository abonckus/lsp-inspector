package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	webfs "github.com/abonckus/lsp-inspector/web"
	"github.com/abonckus/lsp-inspector/internal/parser"
	"github.com/abonckus/lsp-inspector/internal/server"
	"github.com/abonckus/lsp-inspector/internal/watcher"
)

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

	// Start file watching
	lineOffset := len(lines)

	// Create server — web.FS is already rooted at "web/", no Sub needed
	srv := server.New(webfs.FS, initialMsgs, logFile, func() {
		w.ResetOffset()
		lineOffset = 0
	})
	changes := w.Watch()
	go func() {
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
		// Extract just the port number from the full address (e.g. "[::]:55755" -> ":55755")
		_, portStr, err := net.SplitHostPort(listener.Addr().String())
		if err != nil {
			log.Fatal(err)
		}
		addr = ":" + portStr
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
