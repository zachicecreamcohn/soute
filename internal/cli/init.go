package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zacharycohn/soute/internal/config"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <path>",
		Short: "Initialize soute tracking for a project file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args[0])
		},
	}
}

func runInit(cmd *cobra.Command, targetArg string) error {
	paths, err := config.ResolvePaths(targetArg)
	if err != nil {
		return err
	}

	// Enforce strict single-file targeting: reject directories. A target that
	// does not yet exist is allowed (it may be created later); watch will
	// surface a clear error in that case.
	if fi, err := os.Stat(paths.TargetAbs); err == nil && fi.IsDir() {
		return fmt.Errorf("%s is a directory; soute tracks a single project file, not a directory (select the primary project file)", paths.TargetAbs)
	}

	if err := paths.Ensure(); err != nil {
		return err
	}

	cfg := config.Default(paths.TargetAbs)
	if err := cfg.Save(paths.Config); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Initialized soute tracking for %s\n", paths.TargetAbs)
	return nil
}
