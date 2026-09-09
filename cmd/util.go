package cmd

import (
	"fmt"

	"soute/internal/ipc"
)

// withDaemonPaused pauses the daemon (if running) around fn, resuming it after.
func withDaemonPaused(snapDir string, fn func() error) error {
	if !ipc.IsRunning(snapDir) {
		return fn()
	}
	client := ipc.NewClient(snapDir)
	if err := client.Send(ipc.CmdPause); err != nil {
		return fmt.Errorf("pause daemon: %w", err)
	}
	defer func() { _ = client.Send(ipc.CmdResume) }()
	return fn()
}
