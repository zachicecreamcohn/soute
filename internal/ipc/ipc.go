// Package ipc provides the cross-platform control channel between the CLI
// and the background daemon: POSIX signals on unix, a named pipe on Windows.
package ipc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"soute/internal/paths"
)

// Command is a control message sent to the daemon.
type Command string

const (
	CmdStop   Command = "stop"
	CmdPause  Command = "pause"
	CmdResume Command = "resume"
)

// Server receives control commands inside the daemon process.
type Server interface {
	Commands() <-chan Command
	Close() error
}

// Client sends control commands to a running daemon.
type Client interface {
	Send(Command) error
	Ping() error
}

// IsRunning reports whether a daemon is currently reachable.
func IsRunning(snapDir string) bool {
	return NewClient(snapDir).Ping() == nil
}

// PipeName derives a stable per-project control-channel name from the
// snapshots directory (used on Windows; ignored on unix).
func PipeName(snapDir string) string {
	sum := sha256.Sum256([]byte(snapDir))
	return "soute-" + hex.EncodeToString(sum[:8])
}

// WritePID records the current process id in daemon.pid.
func WritePID(snapDir string) error {
	return os.WriteFile(paths.PidPath(snapDir), []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// ReadPID returns the recorded daemon process id.
func ReadPID(snapDir string) (int, error) {
	b, err := os.ReadFile(paths.PidPath(snapDir))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file: %w", err)
	}
	return pid, nil
}

// RemovePID deletes daemon.pid.
func RemovePID(snapDir string) error {
	if err := os.Remove(paths.PidPath(snapDir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
