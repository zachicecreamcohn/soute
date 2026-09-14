// Package process provides OS-specific detachment for the soute daemon.
package process

import (
	"os"
	"os/exec"
)

// Detach applies the platform-specific attributes that detach cmd from the
// parent's session/console.
func Detach(cmd *exec.Cmd) {
	applyDetach(cmd)
}

// RedirectToLog sends cmd's stdout and stderr to the log file at path (created
// if needed) and disconnects stdin (which Go maps to the null device). The
// returned file is the parent's copy and should be closed by the caller after
// Start.
func RedirectToLog(cmd *exec.Cmd, path string) (*os.File, error) {
	logf, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.Stdin = nil
	return logf, nil
}
