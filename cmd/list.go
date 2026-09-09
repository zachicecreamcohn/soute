package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/manifest"
	"github.com/zachicecreamcohn/soute/internal/render"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List full snapshot history",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			snapDir, err := snapshotsDir()
			if err != nil {
				return err
			}
			m, err := manifest.Load(snapDir)
			if err != nil {
				return err
			}
			if len(m.Snapshots) == 0 {
				fmt.Println("no snapshots yet")
				return nil
			}
			fmt.Print(render.Snapshots(m.Snapshots))
			return nil
		},
	}
}
