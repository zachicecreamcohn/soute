package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/engine"
	"github.com/zachicecreamcohn/soute/internal/manifest"
	"github.com/zachicecreamcohn/soute/internal/tui"
)

func newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [snapshot_id]",
		Short: "Restore the target to a previous snapshot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			snapDir, err := requireSnapshots()
			if err != nil {
				return err
			}
			eng := engine.New(snapDir)

			var chosen *manifest.Snapshot
			if len(args) == 1 {
				chosen, err = eng.Find(args[0])
				if err != nil {
					return err
				}
			} else {
				m, err := manifest.Load(snapDir)
				if err != nil {
					return err
				}
				chosen, err = tui.SelectSnapshot(m.Snapshots)
				if err != nil {
					return err
				}
			}

			return withDaemonPaused(snapDir, func() error {
				if _, err := eng.PreRestoreBackup(); err != nil {
					return err
				}
				entry, err := eng.Restore(chosen.ID)
				if err != nil {
					return err
				}
				fmt.Printf("restored snapshot %s\n", entry.ID)
				return nil
			})
		},
	}
}
