package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"soute/internal/config"
	"soute/internal/manifest"
	"soute/internal/paths"
	"soute/internal/tui"
)

func newInitCmd() *cobra.Command {
	var (
		target   string
		minBytes int64
		minPct   float64
		cooldown int
		maxSnaps int
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a .snapshots configuration for a target file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			interactive := !cmd.Flags().Changed("target") &&
				!cmd.Flags().Changed("min-bytes") &&
				!cmd.Flags().Changed("min-pct") &&
				!cmd.Flags().Changed("cooldown") &&
				!cmd.Flags().Changed("max-snapshots")

			var cfg *config.Config
			if interactive {
				c, err := tui.Init(nil)
				if err != nil {
					return err
				}
				cfg = c
			} else {
				c := config.Default()
				c.TargetPath = target
				if cmd.Flags().Changed("min-bytes") {
					c.MinDeltaBytes = minBytes
				}
				if cmd.Flags().Changed("min-pct") {
					c.MinDeltaPct = minPct
				}
				if cmd.Flags().Changed("cooldown") {
					c.CooldownSeconds = cooldown
				}
				if cmd.Flags().Changed("max-snapshots") {
					c.MaxSnapshots = maxSnaps
				}
				cfg = &c
			}

			if strings.TrimSpace(cfg.TargetPath) == "" {
				return errors.New("target path is required (use --target or run interactively)")
			}
			return initProject(cfg)
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "target file to watch (non-interactive)")
	cmd.Flags().Int64Var(&minBytes, "min-bytes", 0, "minimum absolute size delta in bytes")
	cmd.Flags().Float64Var(&minPct, "min-pct", 0, "minimum relative size delta in percent")
	cmd.Flags().IntVar(&cooldown, "cooldown", 0, "cooldown between snapshots in seconds")
	cmd.Flags().IntVar(&maxSnaps, "max-snapshots", 0, "maximum snapshots to retain")
	return cmd
}

func initProject(cfg *config.Config) error {
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
