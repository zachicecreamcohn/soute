//go:build !windows

package ipc

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Spawn re-executes the current binary as a detached daemon child.
func Spawn(snapDir string, stdout, stderr *os.File) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "watch", "--daemon-child", "--dir", filepath.Dir(snapDir))
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
