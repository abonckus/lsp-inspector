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
	path   string
	offset int64
	mu     sync.Mutex
	fsw    *fsnotify.Watcher
	done   chan struct{}
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
