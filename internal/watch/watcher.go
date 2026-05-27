package watch

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileIndexer interface {
	IndexFile(ctx context.Context, path string) error
}

type Watcher struct {
	indexer    FileIndexer
	fw        *fsnotify.Watcher
	mu        sync.Mutex
	timers    map[string]*time.Timer
	work      chan string
	done      chan struct{}
	stopped   bool
	watchExts map[string]bool
}

const (
	debounceDelay = 300 * time.Millisecond
	workerCount   = 4
)

func New(indexer FileIndexer, extensions []string) *Watcher {
	exts := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		exts[ext] = true
	}
	if len(exts) == 0 {
		exts[".jsonl"] = true
	}
	return &Watcher{
		indexer:    indexer,
		timers:    make(map[string]*time.Timer),
		work:      make(chan string, 64),
		done:      make(chan struct{}),
		watchExts: exts,
	}
}

func (w *Watcher) Start(ctx context.Context, roots []string) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.fw = fw

	for i := 0; i < workerCount; i++ {
		go w.worker(ctx)
	}

	for _, root := range roots {
		w.addRecursive(root)
	}

	go w.loop(ctx)
	return nil
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.mu.Unlock()

	close(w.done)
	if w.fw != nil {
		w.fw.Close()
	}
}

func (w *Watcher) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			w.handleEvent(ev)
		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(ev fsnotify.Event) {
	if ev.Op&(fsnotify.Create) != 0 {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			w.addRecursive(ev.Name)
			return
		}
	}

	if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
		return
	}

	ext := filepath.Ext(ev.Name)
	if !w.watchExts[ext] {
		return
	}

	w.debounce(ev.Name)
}

func (w *Watcher) debounce(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}

	if t, ok := w.timers[path]; ok {
		t.Stop()
	}
	w.timers[path] = time.AfterFunc(debounceDelay, func() {
		w.mu.Lock()
		delete(w.timers, path)
		w.mu.Unlock()

		select {
		case w.work <- path:
		default:
			log.Printf("watcher work queue full, dropping %s", path)
		}
	})
}

func (w *Watcher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case path, ok := <-w.work:
			if !ok {
				return
			}
			if err := w.indexer.IndexFile(ctx, path); err != nil {
				log.Printf("index %s: %v", path, err)
			}
		}
	}
}

func (w *Watcher) addRecursive(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if err := w.fw.Add(path); err != nil {
				log.Printf("watch add %s: %v", path, err)
			}
		}
		return nil
	})
}
