package kami

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// OrphanGuard tracks child processes spawned by Kami (e.g. MCP stdio
// servers). When the parent process receives SIGTERM or SIGINT, or when
// Cleanup is called, all tracked children are sent SIGTERM.
type OrphanGuard struct {
	mu       sync.Mutex
	children []*os.Process
	done     chan struct{}
	logger   *slog.Logger
}

// NewOrphanGuard creates and starts an orphan guard. It installs signal
// handlers for SIGTERM and SIGINT.
func NewOrphanGuard(logger *slog.Logger) *OrphanGuard {
	if logger == nil {
		logger = slog.Default()
	}
	og := &OrphanGuard{
		done:   make(chan struct{}),
		logger: logger,
	}
	go og.watch()
	return og
}

// Track registers a child process for cleanup.
func (og *OrphanGuard) Track(proc *os.Process) {
	og.mu.Lock()
	defer og.mu.Unlock()
	og.children = append(og.children, proc)
}

// Cleanup sends SIGTERM to all tracked children. Safe to call multiple times.
func (og *OrphanGuard) Cleanup() {
	og.mu.Lock()
	children := make([]*os.Process, len(og.children))
	copy(children, og.children)
	og.children = nil
	og.mu.Unlock()

	for _, proc := range children {
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			og.logger.Debug("orphan guard: failed to signal child",
				"pid", proc.Pid, "error", err)
		} else {
			og.logger.Info("orphan guard: sent SIGTERM to child", "pid", proc.Pid)
		}
	}
}

func (og *OrphanGuard) watch() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-ch:
		og.Cleanup()
	case <-og.done:
	}
}

// Close stops the signal watcher.
func (og *OrphanGuard) Close() {
	select {
	case og.done <- struct{}{}:
	default:
	}
}
