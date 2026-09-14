package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/zacharycohn/soute/internal/ipc"
	"github.com/zacharycohn/soute/internal/state"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [path]",
		Short: "Stop the soute daemon",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			return runStop(cmd, arg)
		},
	}
}

func runStop(cmd *cobra.Command, arg string) error {
	paths, err := resolvePaths(arg)
	if err != nil {
		return err
	}
	addr := ipc.EndpointFor(paths.Namespace, paths.TargetAbs)

	if _, err := state.Load(paths.Daemon); err != nil {
		ipc.Cleanup(addr)
		fmt.Fprintln(cmd.OutOrStdout(), "not running")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := ipc.DialClient(ctx, addr)
	if err != nil {
		// Daemon is down; clean up stale state.
		state.Remove(paths.Daemon)
		ipc.Cleanup(addr)
		fmt.Fprintln(cmd.OutOrStdout(), "not running")
		return nil
	}
	defer c.Close()

	if _, err := c.Call(ctx, ipc.CmdStop, nil); err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "stopped")
	return nil
}
