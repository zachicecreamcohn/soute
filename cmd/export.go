package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/engine"
	"github.com/zachicecreamcohn/soute/internal/manifest"
	"github.com/zachicecreamcohn/soute/internal/tui"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [snapshot_id] [destination]",
		Short: "Export a snapshot to a destination path",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			snapDir, err := requireSnapshots()
			if err != nil {
				return err
			}
			eng := engine.New(snapDir)

			var snap *manifest.Snapshot
			if len(args) >= 1 {
				snap, err = eng.Find(args[0])
				if err != nil {
					return err
				}
			} else {
				m, err := manifest.Load(snapDir)
				if err != nil {
					return err
				}
				snap, err = tui.SelectSnapshot(m.Snapshots)
				if err != nil {
					return err
				}
			}

			dst := ""
			if len(args) >= 2 {
				dst = args[1]
			} else {
				dst, err = tui.InputPath("Export destination")
				if err != nil {
					return err
				}
			}

			if err := eng.Export(snap.ID, dst); err != nil {
				return err
			}
			fmt.Printf("exported %s to %s\n", snap.ID, dst)
			return nil
		},
	}
}
