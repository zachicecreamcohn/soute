package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"soute/internal/ipc"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running daemon",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			snapDir, err := snapshotsDir()
			if err != nil {
				return err
			}
			if !ipc.IsRunning(snapDir) {
				return errors.New("not running")
			}
			if err := ipc.NewClient(snapDir).Send(ipc.CmdStop); err != nil {
				return fmt.Errorf("send stop: %w", err)
			}

			deadline := time.Now().Add(5 * time.Second)
			for ipc.IsRunning(snapDir) {
				if time.Now().After(deadline) {
					return errors.New("daemon did not stop in time")
				}
				time.Sleep(100 * time.Millisecond)
			}
			_ = ipc.RemovePID(snapDir)
			fmt.Println("stopped")
			return nil
		},
	}
}
