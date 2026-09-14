package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/watcher"
)

// newDaemonCmd is the hidden re-exec entrypoint for the detached daemon.
func newDaemonCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "__daemon <abs-path>",
		Short:  "Run the soute daemon (internal)",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon(args[0])
		},
	}
}

func runDaemon(targetArg string) error {
	paths, err := config.ResolvePaths(targetArg)
	if err != nil {
		return err
	}
	cfg, err := config.Load(paths.Config)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	return watcher.Run(context.Background(), paths, cfg)
}
