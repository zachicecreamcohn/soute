package watcher

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zacharycohn/soute/internal/config"
	"github.com/zacharycohn/soute/internal/ipc"
	"github.com/zacharycohn/soute/internal/store"
)

// Handler serves IPC commands for a running daemon.
type Handler struct {
	eval     *Evaluator
	cfg      config.Config
	paths    config.Paths
	pid      int
	started  time.Time
	shutdown func() // invoked by "stop" to close the listener
}

// NewHandler builds a Handler.
func NewHandler(eval *Evaluator, cfg config.Config, paths config.Paths, pid int, started time.Time, shutdown func()) *Handler {
	return &Handler{eval: eval, cfg: cfg, paths: paths, pid: pid, started: started, shutdown: shutdown}
}

// Handle implements ipc.Handler.
func (h *Handler) Handle(cmd string, _ json.RawMessage) (any, error) {
	switch cmd {
	case ipc.CmdPing:
		return ipc.PingResponse{
			UptimeSeconds: time.Since(h.started).Seconds(),
			Paused:        h.eval.IsPaused(),
			PID:           h.pid,
		}, nil
	case ipc.CmdStatus:
		return h.status(), nil
	case ipc.CmdPause:
		return ipc.PauseResponse{WasPaused: h.eval.Pause()}, nil
	case ipc.CmdResume:
		mtime, hash, err := h.eval.Resume()
		if err != nil {
			return nil, err
		}
		return ipc.ResumeResponse{BaselineMtime: mtime, BaselineHash: hash}, nil
	case ipc.CmdPrune:
		return h.prune(), nil
	case ipc.CmdStop:
		go h.shutdown()
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown command %q", cmd)
	}
}

func (h *Handler) status() ipc.StatusResponse {
	r := ipc.StatusResponse{
		Status:        stateRunning,
		Paused:        h.eval.IsPaused(),
		UptimeSeconds: time.Since(h.started).Seconds(),
		PID:           h.pid,
		TargetPath:    h.cfg.TargetPath,
	}
	if m, err := store.Load(h.paths.Manifest); err == nil {
		if s, ok := store.Newest(m); ok {
			r.LastSnapshotID = s.ID
			r.LastSnapshotTime = s.Timestamp.UTC().Format(time.RFC3339)
		}
	}
	return r
}

func (h *Handler) prune() ipc.PruneResponse {
	m, err := store.Load(h.paths.Manifest)
	if err != nil {
		return ipc.PruneResponse{}
	}
	res, err := store.Prune(&m, h.cfg.MaxSnapshots, h.paths.DataDir)
	if err != nil {
		return ipc.PruneResponse{}
	}
	if res.RemovedEntries > 0 {
		_ = store.SaveAtomic(h.paths.Manifest, m)
	}
	return ipc.PruneResponse{Removed: res.RemovedEntries, DeletedBlobs: res.DeletedBlobs}
}

const stateRunning = "running"
