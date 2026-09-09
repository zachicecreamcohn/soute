// Package cmd wires the cobra CLI command tree.
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/paths"
)

// version is injected at release time via
// -ldflags "-X github.com/zachicecreamcohn/soute/cmd.version=vX.Y.Z".
var version = "dev"

var dirFlag string

// NewRootCmd builds the root soute command and its subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "soute",
		Short:         "Zero-friction versioning for large binary & media files",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVarP(&dirFlag, "dir", "d", ".", "directory containing the .snapshots folder")
	root.AddCommand(
		newInitCmd(),
		newWatchCmd(),
		newStopCmd(),
		newStatusCmd(),
		newListCmd(),
		newRestoreCmd(),
		newExportCmd(),
		newPruneCmd(),
		newConfigCmd(),
	)
	return root
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}

// snapshotsDir resolves --dir to an absolute .snapshots path.
func snapshotsDir() (string, error) {
	abs, err := filepath.Abs(dirFlag)
	if err != nil {
		return "", fmt.Errorf("resolve --dir: %w", err)
	}
	return paths.SnapDir(abs), nil
}
