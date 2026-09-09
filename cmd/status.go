package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/ipc"
	"github.com/zachicecreamcohn/soute/internal/manifest"
	"github.com/zachicecreamcohn/soute/internal/render"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon health and recent snapshots",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			snapDir, err := requireSnapshots()
			if err != nil {
				return err
			}
			cfg, err := config.Load(snapDir)
			if err != nil {
				return err
			}
			m, err := manifest.Load(snapDir)
			if err != nil {
				return err
			}

			state := "Stopped"
			if ipc.IsRunning(snapDir) {
				state = "Running"
			}
			fmt.Printf("state:     %s\n", state)
			if pid, err := ipc.ReadPID(snapDir); err == nil && state == "Running" {
				fmt.Printf("pid:       %s\n", strconv.Itoa(pid))
			}
			fmt.Printf("target:    %s\n", cfg.TargetPath)
			fmt.Printf("snapshots: %d\n", len(m.Snapshots))

			last := m.Snapshots
			if len(last) > 3 {
				last = last[len(last)-3:]
			}
			if len(last) > 0 {
				fmt.Println()
				fmt.Print(render.Snapshots(last))
			}
			return nil
		},
	}
}
