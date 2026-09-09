package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/manifest"
	"github.com/zachicecreamcohn/soute/internal/paths"
	"github.com/zachicecreamcohn/soute/internal/tui"
)

func newInitCmd() *cobra.Command {
	var (
		minBytes int64
		minPct   float64
		cooldown int
		maxSnaps int
	)

	cmd := &cobra.Command{
		Use:   "init <target>",
		Short: "Create a .snapshots configuration for a target file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("target path is required: `soute init <target>`")
			}
			target := paths.CleanInput(args[0])
			if target == "" {
				return errors.New("target path is required: `soute init <target>`")
			}

			prefill := config.Default()
			prefill.TargetPath = target

			// Flags select non-interactive mode; otherwise run the settings
			// wizard (it never asks for the path, which is already positional).
			interactive := !cmd.Flags().Changed("min-bytes") &&
				!cmd.Flags().Changed("min-pct") &&
				!cmd.Flags().Changed("cooldown") &&
				!cmd.Flags().Changed("max-snapshots")

			var cfg *config.Config
			if interactive {
				c, err := tui.Settings(&prefill)
				if err != nil {
					return err
				}
				cfg = c
			} else {
				if cmd.Flags().Changed("min-bytes") {
					prefill.MinDeltaBytes = minBytes
				}
				if cmd.Flags().Changed("min-pct") {
					prefill.MinDeltaPct = minPct
				}
				if cmd.Flags().Changed("cooldown") {
					prefill.CooldownSeconds = cooldown
				}
				if cmd.Flags().Changed("max-snapshots") {
					prefill.MaxSnapshots = maxSnaps
				}
				cfg = &prefill
			}
			return initProject(cfg)
		},
	}

	cmd.Flags().Int64Var(&minBytes, "min-bytes", 0, "minimum absolute size delta in bytes")
	cmd.Flags().Float64Var(&minPct, "min-pct", 0, "minimum relative size delta in percent")
	cmd.Flags().IntVar(&cooldown, "cooldown", 0, "cooldown between snapshots in seconds")
	cmd.Flags().IntVar(&maxSnaps, "max-snapshots", 0, "maximum snapshots to retain")
	return cmd
}

func initProject(cfg *config.Config) error {
	if strings.Contains(cfg.TargetPath, "\"") {
		return fmt.Errorf("invalid target path %q (remove quotes)", cfg.TargetPath)
	}
	targetAbs, err := filepath.Abs(cfg.TargetPath)
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}
	snapDir := paths.SnapDir(filepath.Dir(targetAbs))

	if _, err := os.Stat(paths.ConfigPath(snapDir)); err == nil {
		return fmt.Errorf("already initialized at %s (use `soute config edit` to change settings)", snapDir)
	}

	// Store the target relative to the snapshots directory so the config
	// stays valid if the whole project folder moves.
	if rel, err := filepath.Rel(filepath.Dir(snapDir), targetAbs); err == nil && !strings.HasPrefix(rel, "..") {
		cfg.TargetPath = filepath.ToSlash(rel)
	} else {
		cfg.TargetPath = filepath.ToSlash(targetAbs)
	}

	if err := os.MkdirAll(paths.DataDirPath(snapDir), 0o755); err != nil {
		return err
	}
	if err := config.Save(snapDir, cfg); err != nil {
		return err
	}
	if err := manifest.New().Save(snapDir); err != nil {
		return err
	}

	if _, err := os.Stat(targetAbs); err != nil {
		fmt.Printf("note: target %q does not exist yet\n", cfg.TargetPath)
	}

	fmt.Printf("initialized snapshots at %s\n  target: %s\n", snapDir, cfg.TargetPath)
	return nil
}
