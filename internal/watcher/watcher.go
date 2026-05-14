// Package watcher provides file-system watching for timerd.
//
// When config files inside the timerd config directory change, the registered
// Handler is invoked with the path that triggered the event. Changes are
// debounced so that rapid sequential writes (e.g. editor atomic saves) produce
// only a single callback.
package watcher

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Handler is called when a relevant file change is detected.
// path is the absolute path of the file that changed.
type Handler func(path string)

// Watcher monitors a directory for relevant file changes.
type Watcher struct {
	dir     string
	handler Handler
	fw      *fsnotify.Watcher
}

// New creates a new Watcher that will call handler whenever a YAML or template
// file inside dir is created, written, removed, or renamed.
// The caller must call Stop when done to release resources.
func New(dir string, handler Handler) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fw.Add(dir); err != nil {
		_ = fw.Close()
		return nil, err
	}
	return &Watcher{dir: dir, handler: handler, fw: fw}, nil
}

// Run starts the event loop. It blocks until Stop is called or the underlying
// fsnotify watcher channel closes. Changes are debounced for 300 ms.
func (w *Watcher) Run() {
	const debounce = 300 * time.Millisecond

	// timer is used for debouncing; start it stopped.
	timer := time.NewTimer(debounce)
	timer.Stop()

	var lastPath string

	for {
		select {
		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}

			// Only react to YAML / template files.
			ext := filepath.Ext(event.Name)
			if ext != ".yml" && ext != ".yaml" && ext != ".tmpl" {
				continue
			}

			// Ignore chmod-only events.
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			lastPath = event.Name
			// Reset debounce timer.
			timer.Reset(debounce)

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", "err", err)

		case <-timer.C:
			if lastPath != "" && w.handler != nil {
				w.handler(lastPath)
			}
			lastPath = ""
		}
	}
}

// Stop closes the underlying fsnotify watcher, which causes Run to return.
func (w *Watcher) Stop() error {
	return w.fw.Close()
}
