package watcher

import (
	"bufio"
	"log"
	"os"
	"sync"
	"time"
)

// Watcher watches a log file for appended content.
type Watcher struct {
	path   string
	offset int64
	mu     sync.Mutex
	done   chan struct{}
}

// New creates a Watcher for the given file path.
func New(path string) (*Watcher, error) {
	// Verify the file is accessible
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return &Watcher{
		path: path,
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

// Watch starts watching for file changes by polling and returns a channel
// that receives slices of new lines when the file grows.
func (w *Watcher) Watch() <-chan []string {
	ch := make(chan []string, 16)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				newLines := w.readNew()
				if len(newLines) > 0 {
					log.Printf("[watcher] poll: %d new lines", len(newLines))
					select {
					case ch <- newLines:
					case <-w.done:
						return
					}
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

// ResetOffset resets the read offset to 0 (used after clearing the log file).
func (w *Watcher) ResetOffset() {
	w.mu.Lock()
	w.offset = 0
	w.mu.Unlock()
}

// Close stops watching and cleans up.
func (w *Watcher) Close() error {
	close(w.done)
	return nil
}
