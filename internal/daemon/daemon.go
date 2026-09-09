// Package daemon runs the background file watcher: fsnotify events are
// coalesced through a 300ms debounce window before the evaluation engine runs.
package daemon

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/engine"
	"github.com/zachicecreamcohn/soute/internal/ipc"
)

const debounceWindow = 300 * time.Millisecond

// Run starts the watcher loop in the current process and blocks until it
// receives a stop command.
func Run(snapDir string, logger *log.Logger) error {
	if err := ipc.WritePID(snapDir); err != nil {
		return err
	}
	defer ipc.RemovePID(snapDir)

	server, err := ipc.NewServer(snapDir)
	if err != nil {
		return err
	}
	defer server.Close()

	cfg, err := config.Load(snapDir)
	if err != nil {
		return err
	}
	target := cfg.TargetAbs(snapDir)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	// Watch the target's directory: atomic saves swap files in and out, so the
	// directory (not the file) is the stable watch point.
	if err := watcher.Add(filepath.Dir(target)); err != nil {
		return err
	}

	d := &daemon{
		snapDir: snapDir,
		target:  target,
		logger:  logger,
		eng:     engine.New(snapDir),
		watcher: watcher,
	}
	d.timer = time.NewTimer(time.Hour)
	d.timer.Stop()

	return d.loop(server)
}

type daemon struct {
	snapDir string
	target  string
	logger  *log.Logger
	eng     *engine.Engine
	watcher *fsnotify.Watcher
	timer   *time.Timer
	paused  bool
}

func (d *daemon) loop(server ipc.Server) error {
	for {
		select {
		case cmd := <-server.Commands():
			switch cmd {
			case ipc.CmdStop:
				return nil
			case ipc.CmdPause:
				d.paused = true
			case ipc.CmdResume:
				d.paused = false
				d.evaluate()
			}

		case ev, ok := <-d.watcher.Events:
			if !ok {
				return nil
			}
			if !d.paused && relevant(ev.Name, d.target, d.snapDir) {
				d.arm(debounceWindow)
			}

		case err, ok := <-d.watcher.Errors:
			if ok {
				d.logger.Printf("watch error: %v", err)
			}

		case <-d.timer.C:
			d.evaluate()
		}
	}
}

func (d *daemon) arm(after time.Duration) {
	if !d.timer.Stop() {
		select {
		case <-d.timer.C:
		default:
		}
	}
	d.timer.Reset(after)
}

func (d *daemon) evaluate() {
	cfg, err := config.Load(d.snapDir)
	if err != nil {
		d.logger.Printf("evaluate: %v", err)
		return
	}
	res, snap, err := d.eng.Evaluate(time.Now())
	switch {
	case err != nil:
		d.logger.Printf("evaluate: %v", err)
	case res == engine.EvalCaptured:
		d.logger.Printf("snapshot %s (%s, delta %+d bytes)", snap.ID, snap.Tag, snap.DeltaBytes)
	case res == engine.EvalCooldown:
		d.arm(time.Duration(cfg.CooldownSeconds) * time.Second)
	}
}

// relevant reports whether an fsnotify event concerns the target file while
// ignoring the daemon's own writes into .snapshots.
func relevant(name, target, snapDir string) bool {
	return !under(name, snapDir) && filepath.Base(name) == filepath.Base(target)
}

func under(name, dir string) bool {
	rel, err := filepath.Rel(dir, name)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
