package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"soute/internal/config"
	"soute/internal/tui"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View or edit settings",
		Args:  cobra.NoArgs,
		RunE:  configView,
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "view",
			Short: "Print current settings",
			Args:  cobra.NoArgs,
			RunE:  configView,
		},
		&cobra.Command{
			Use:   "edit",
			Short: "Edit settings via the wizard",
			Args:  cobra.NoArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				snapDir, err := snapshotsDir()
				if err != nil {
					return err
				}
				current, err := config.Load(snapDir)
				if err != nil {
					return err
				}
				updated, err := tui.Init(current)
				if err != nil {
					return err
				}
				if err := config.Save(snapDir, updated); err != nil {
					return err
				}
				fmt.Println("settings updated")
				return nil
			},
		},
	)
	return cmd
}

func configView(_ *cobra.Command, _ []string) error {
	snapDir, err := snapshotsDir()
	if err != nil {
		return err
	}
	c, err := config.Load(snapDir)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
