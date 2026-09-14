package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zacharycohn/soute/internal/config"
	"github.com/zacharycohn/soute/internal/store"
)

func newPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune [path]",
		Short: "Trigger garbage collection",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			return runPrune(cmd, arg)
		},
	}
}

func runPrune(cmd *cobra.Command, arg string) error {
	paths, err := resolvePaths(arg)
	if err != nil {
		return err
	}
	cfg, err := config.Load(paths.Config)
	if err != nil {
		return err
	}
	m, err := store.Load(paths.Manifest)
	if err != nil {
		return err
	}

	res, err := store.Prune(&m, cfg.MaxSnapshots, paths.DataDir)
	if err != nil {
		return err
	}
	if res.RemovedEntries > 0 {
		if err := store.SaveAtomic(paths.Manifest, m); err != nil {
			return err
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Pruned %d snapshot(s), deleted %d blob(s)\n", res.RemovedEntries, res.DeletedBlobs)
	return nil
}
