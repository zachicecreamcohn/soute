package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zachicecreamcohn/soute/internal/ipc"
	"github.com/zachicecreamcohn/soute/internal/paths"
)

// requireSnapshots resolves the snapshots directory, returning a friendly
// error when the project has not been initialized.
func requireSnapshots() (string, error) {
	snapDir, err := snapshotsDir()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(paths.ConfigPath(snapDir)); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no .snapshots directory in %s (run `soute init` first)", filepath.Dir(snapDir))
		}
		return "", err
	}
	return snapDir, nil
}

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
