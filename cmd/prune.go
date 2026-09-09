package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/engine"
)

func newPruneCmd() *cobra.Command {
	var (
		keep int
		all  bool
	)

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Garbage-collect snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapDir, err := requireSnapshots()
			if err != nil {
				return err
			}
			eng := engine.New(snapDir)

			return withDaemonPaused(snapDir, func() error {
				if all {
					removed, err := eng.PruneAll()
					if err != nil {
						return err
					}
					fmt.Printf("pruned %d snapshots (kept most recent)\n", removed)
					return nil
				}
				if cmd.Flags().Changed("keep") {
					removed, err := eng.Prune(keep)
					if err != nil {
						return err
					}
					fmt.Printf("pruned %d snapshots\n", removed)
					return nil
				}
				cfg, err := config.Load(snapDir)
				if err != nil {
					return err
				}
				removed, err := eng.Prune(cfg.MaxSnapshots)
				if err != nil {
					return err
				}
				fmt.Printf("pruned %d snapshots\n", removed)
				return nil
			})
		},
	}

	cmd.Flags().IntVar(&keep, "keep", 0, "keep the newest N snapshots")
	cmd.Flags().BoolVar(&all, "all", false, "remove everything but the most recent snapshot")
	return cmd
}
