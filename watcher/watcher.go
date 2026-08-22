package watcher

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config configures the recursive filesystem change watcher.
type Config struct {
	// Paths specifies the directories or files to monitor. Defaults to ["."].
	Paths []string

	// Extensions restricts change events to specific file extensions (e.g. [".go", ".html"]).
	// If empty, all file modifications trigger change events.
	Extensions []string

	// IgnoredNames specifies directory or file names to skip (e.g. [".git", "node_modules", "vendor"]).
	IgnoredNames []string

	// Debounce is the duration to coalesce rapid successive filesystem modifications.
	// Defaults to 150ms.
	Debounce time.Duration

	// PollInterval is the interval between full filesystem stat scans.
	// Defaults to 200ms.
	PollInterval time.Duration
}

// DefaultConfig returns recommended defaults for Go development watching.
func DefaultConfig(paths ...string) Config {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	return Config{
		Paths:        paths,
		Extensions:   []string{".go", ".mod", ".sum", ".json", ".yaml", ".yml", ".html"},
		IgnoredNames: []string{".git", "node_modules", "vendor", ".idea", ".vscode", "bin", "dist", ".tmp"},
		Debounce:     150 * time.Millisecond,
		PollInterval: 200 * time.Millisecond,
	}
}

// Watcher monitors directories recursively for file creations, modifications, and deletions.
type Watcher struct {
	cfg        Config
	mu         sync.Mutex
	knownFiles map[string]time.Time
	stopped    bool
}

// New constructs a new filesystem Watcher.
func New(cfg Config) *Watcher {
	if len(cfg.Paths) == 0 {
		cfg.Paths = []string{"."}
	}
	if cfg.Debounce <= 0 {
		cfg.Debounce = 150 * time.Millisecond
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 200 * time.Millisecond
	}
	return &Watcher{
		cfg:        cfg,
		knownFiles: make(map[string]time.Time),
	}
}

// Start begins monitoring the filesystem and returns a channel delivering
// debounced slices of changed filepaths.
func (w *Watcher) Start(ctx context.Context) (<-chan []string, error) {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return nil, errors.New("watcher: already stopped")
	}

	// Initialize baseline snapshot
	initialMap, err := w.scan()
	if err != nil {
		w.mu.Unlock()
		return nil, err
	}
	w.knownFiles = initialMap
	w.mu.Unlock()

	outCh := make(chan []string, 16)

	go w.pollLoop(ctx, outCh)

	return outCh, nil
}

// pollLoop performs periodic scanning and coalesces events during the debounce window.
func (w *Watcher) pollLoop(ctx context.Context, outCh chan<- []string) {
	defer close(outCh)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	var pendingChanges []string
	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case <-debounceCh:
			// Debounce window expired: emit all aggregated changes
			w.mu.Lock()
			changesToSend := pendingChanges
			pendingChanges = nil
			debounceCh = nil
			debounceTimer = nil
			w.mu.Unlock()

			if len(changesToSend) > 0 {
				select {
				case outCh <- changesToSend:
				case <-ctx.Done():
					return
				}
			}

		case <-ticker.C:
			currentMap, err := w.scan()
			if err != nil {
				continue
			}

			w.mu.Lock()
			changed := w.detectDiff(w.knownFiles, currentMap)
			w.knownFiles = currentMap

			if len(changed) > 0 {
				pendingChanges = append(pendingChanges, changed...)
				if debounceTimer == nil {
					debounceTimer = time.NewTimer(w.cfg.Debounce)
					debounceCh = debounceTimer.C
				} else {
					// Reset debounce timer on each burst event
					debounceTimer.Reset(w.cfg.Debounce)
				}
			}
			w.mu.Unlock()
		}
	}
}

// scan walks all configured paths and builds a map of filepath -> modtime.
func (w *Watcher) scan() (map[string]time.Time, error) {
	result := make(map[string]time.Time)

	for _, root := range w.cfg.Paths {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip unreadable paths without failing entire scan
			}

			baseName := d.Name()

			// Check ignore list
			for _, ignored := range w.cfg.IgnoredNames {
				if baseName == ignored || strings.EqualFold(baseName, ignored) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if d.IsDir() {
				return nil
			}

			// Check extension filter
			if len(w.cfg.Extensions) > 0 {
				ext := strings.ToLower(filepath.Ext(path))
				matched := false
				for _, allowed := range w.cfg.Extensions {
					if strings.EqualFold(ext, allowed) {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}

			info, statErr := d.Info()
			if statErr != nil {
				return nil
			}

			result[path] = info.ModTime()
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	return result, nil
}

// detectDiff returns all files created, modified, or deleted between old and new state.
func (w *Watcher) detectDiff(oldState, newState map[string]time.Time) []string {
	var diff []string

	for path, newMod := range newState {
		if oldMod, exists := oldState[path]; !exists || !oldMod.Equal(newMod) {
			diff = append(diff, path)
		}
	}

	for path := range oldState {
		if _, exists := newState[path]; !exists {
			diff = append(diff, path)
		}
	}

	return diff
}
