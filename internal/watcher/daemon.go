package watcher

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/zacharycohn/soute/internal/config"
	"github.com/zacharycohn/soute/internal/ipc"
	"github.com/zacharycohn/soute/internal/state"
)

// Run starts the daemon: fsnotify watcher, debounce/evaluation loop, and IPC
// listener. It blocks until stop is requested (via IPC or signal), then cleans
// up and returns nil. Any startup error is returned to the caller.
func Run(ctx context.Context, paths config.Paths, cfg config.Config) error {
	// Canonicalize the target for watching so symlink aliases (e.g. /var vs
	// /private/var on macOS) don't cause event-filter mismatches. The namespace
	// and endpoint stay on the original logical paths for CLI consistency.
	watchPaths := paths
	if real, err := filepath.EvalSymlinks(paths.TargetAbs); err == nil {
		watchPaths.TargetAbs = real
		watchPaths.TargetDir = filepath.Dir(real)
	}
	if _, err := os.Stat(watchPaths.TargetAbs); err != nil {
		return fmt.Errorf("target not found: %w", err)
	}

	eval := NewEvaluator(watchPaths, cfg)

	// Set up fsnotify before announcing readiness so watch failures surface
	// during startup rather than after the daemon reports "started".
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("new watcher: %w", err)
	}
	defer w.Close()
	if err := w.Add(watchPaths.TargetDir); err != nil {
		if err2 := w.Add(watchPaths.TargetAbs); err2 != nil {
			return fmt.Errorf("watch target: %w (dir: %v)", err2, err)
		}
	}

	addr := ipc.EndpointFor(paths.Namespace, paths.TargetAbs)
	ipc.Cleanup(addr)
	ln, err := ipc.Listen(addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer ipc.Cleanup(addr)

	started := time.Now().UTC()
	st := state.DaemonState{
		PID:       os.Getpid(),
		IPCPath:   addr,
		Status:    state.StatusRunning,
		StartedAt: started,
	}
	if err := st.Save(paths.Daemon); err != nil {
		_ = ln.Close()
		return fmt.Errorf("write daemon state: %w", err)
	}
	defer state.Remove(paths.Daemon)

	daemonCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	shutdown := func() {
		cancel()
		_ = ln.Close()
	}
	handler := NewHandler(eval, cfg, paths, st.PID, started, shutdown)

	serveErr := make(chan error, 1)
	go func() { serveErr <- ipc.Serve(ln, handler) }()

	deb := NewDebounce(time.Duration(cfg.DebounceMS)*time.Millisecond, nil)
	defer deb.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-daemonCtx.Done():
			return nil
		case err := <-serveErr:
			select {
			case <-daemonCtx.Done():
				return nil
			default:
				if err != nil {
					return fmt.Errorf("ipc serve: %w", err)
				}
				return nil
			}
		case <-sigCh:
			return nil
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if filepath.Clean(ev.Name) != watchPaths.TargetAbs {
				continue // strict single-file targeting: ignore sibling/media events
			}
			deb.Trigger()
		case <-w.Errors:
			// Watcher errors are typically transient; continue.
		case <-deb.C():
			// Evaluation failure is non-fatal; drop and continue.
			_, _, _ = eval.Evaluate()
		}
	}
}
