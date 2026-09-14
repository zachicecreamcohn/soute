package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/zacharycohn/soute/internal/config"
	"github.com/zacharycohn/soute/internal/ipc"
	"github.com/zacharycohn/soute/internal/process"
	"github.com/zacharycohn/soute/internal/state"
)

func newWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch [path]",
		Short: "Start the soute daemon in the background",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			return runWatch(cmd, arg)
		},
	}
}

func runWatch(cmd *cobra.Command, arg string) error {
	paths, err := resolvePaths(arg)
	if err != nil {
		return err
	}
	cfg, err := config.Load(paths.Config)
	if err != nil {
		return fmt.Errorf("load config (run `soute init` first): %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	addr := ipc.EndpointFor(paths.Namespace, paths.TargetAbs)

	// Already-running check (by ping, not pid — immune to pid reuse).
	if running, _ := ping(addr); running {
		fmt.Fprintf(cmd.OutOrStdout(), "already watching %s\n", paths.TargetAbs)
		return nil
	}
	// Stale state from a crashed predecessor.
	state.Remove(paths.Daemon)
	ipc.Cleanup(addr)

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	child := exec.Command(exe, "__daemon", paths.TargetAbs)
	logf, err := process.RedirectToLog(child, paths.LogFile)
	if err != nil {
		return err
	}
	defer logf.Close()
	process.Detach(child)

	if err := child.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	if err := waitForDaemon(addr, 5*time.Second); err != nil {
		return fmt.Errorf("daemon failed to start: %w (see %s)", err, paths.LogFile)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "started watching %s (pid %d)\n", paths.TargetAbs, child.Process.Pid)
	return nil
}

func waitForDaemon(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok, _ := ping(addr); ok {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for daemon")
}
