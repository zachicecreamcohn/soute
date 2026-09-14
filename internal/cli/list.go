package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/zacharycohn/soute/internal/store"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [path]",
		Short: "List the snapshot history",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			return runList(cmd, arg)
		},
	}
}

func runList(cmd *cobra.Command, arg string) error {
	paths, err := resolvePaths(arg)
	if err != nil {
		return err
	}
	m, err := store.Load(paths.Manifest)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTIMESTAMP\tSIZE")
	for _, s := range m.Snapshots {
		fmt.Fprintf(w, "%s\t%s\t%d\n", s.ID, s.Timestamp.UTC().Format(time.RFC3339), s.SizeBytes)
	}
	return w.Flush()
}
