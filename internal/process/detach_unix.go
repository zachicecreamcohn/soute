//go:build darwin || linux

package process

import (
	"os/exec"
	"syscall"
)

func applyDetach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
