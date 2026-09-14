// Package cli wires soute's cobra commands to the internal packages.
package cli

import (
	"github.com/spf13/cobra"
)

// Version is the build version, injected at build time via
// -ldflags "-X github.com/zachicecreamcohn/soute/internal/cli.Version=vX.Y.Z".
var Version = "dev"

// NewRootCmd builds the soute command tree.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "soute",
		Short: "Zero-friction project file versioning",
		Long: "soute watches a single structural project file (QLab, WATCHOUT, " +
			"Ableton) and automatically creates content-addressed, deduplicated snapshots.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newInitCmd(),
		newWatchCmd(),
		newStopCmd(),
		newStatusCmd(),
		newListCmd(),
		newRestoreCmd(),
		newPruneCmd(),
		newDaemonCmd(),
	)
	return root
}

// Execute runs the root command and returns its error.
func Execute() error {
	return NewRootCmd().Execute()
}
