package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"soute/internal/config"
	"soute/internal/daemon"
	"soute/internal/ipc"
)

func newWatchCmd() *cobra.Command {
	var (
		daemonFlag  bool
		daemonChild bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor the target and snapshot on change",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			snapDir, err := snapshotsDir()
			if err != nil {
				return err
			}
			if _, err := config.Load(snapDir); err != nil {
				return fmt.Errorf("not initialized (run `soute init` first): %w", err)
			}

			if daemonChild {
				return runDaemon(snapDir)
			}
			if ipc.IsRunning(snapDir) {
				return errors.New("already watching this target")
			}

			if daemonFlag {
				devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				if err != nil {
					return err
				}
				defer devnull.Close()
				if err := ipc.Spawn(snapDir, devnull, devnull); err != nil {
					return fmt.Errorf("spawn daemon: %w", err)
				}
				fmt.Println("watching in the background (use `soute status` to check)")
				return nil
			}
			return runDaemon(snapDir)
		},
	}

	cmd.Flags().BoolVar(&daemonFlag, "daemon", true, "fork into the background")
	cmd.Flags().BoolVar(&daemonChild, "daemon-child", false, "internal: run as the daemon child")
	_ = cmd.Flags().MarkHidden("daemon-child")
	return cmd
}

func runDaemon(snapDir string) error {
	logger := log.New(os.Stderr, "soute: ", log.LstdFlags)
	return daemon.Run(snapDir, logger)
}
