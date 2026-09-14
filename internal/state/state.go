// Package state persists the daemon's daemon.json state so CLI commands can
// route IPC requests to the correct local endpoint.
package state

import (
	"encoding/json"
	"os"
	"time"

	"github.com/zacharycohn/soute/internal/atomicwrite"
)

// Status values.
const (
	StatusRunning  = "running"
	StatusStopping = "stopping"
)

// DaemonState is written to daemon.json at daemon startup.
type DaemonState struct {
	PID       int       `json:"pid"`
	IPCPath   string    `json:"ipc_path"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// Save writes the state atomically.
func (s DaemonState) Save(path string) error {
	return atomicwrite.JSON(path, s, 0o600)
}

// Load reads and parses the state.
func Load(path string) (DaemonState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DaemonState{}, err
	}
	var s DaemonState
	if err := json.Unmarshal(data, &s); err != nil {
		return DaemonState{}, err
	}
	return s, nil
}

// Remove deletes the state file, ignoring absence.
func Remove(path string) {
	_ = os.Remove(path)
}
