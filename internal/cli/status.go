package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/zacharycohn/soute/internal/ipc"
	"github.com/zacharycohn/soute/internal/store"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [path]",
		Short: "Show daemon health and recent snapshots",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			return runStatus(cmd, arg)
		},
	}
}

func runStatus(cmd *cobra.Command, arg string) error {
	paths, err := resolvePaths(arg)
	if err != nil {
		return err
	}
	addr := ipc.EndpointFor(paths.Namespace, paths.TargetAbs)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := ipc.DialClient(ctx, addr)
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}
	defer c.Close()

	raw, err := c.Call(ctx, ipc.CmdStatus, nil)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	var st ipc.StatusResponse
	if err := json.Unmarshal(raw, &st); err != nil {
		return err
	}

	status := st.Status
	if st.Paused {
		status += " (paused)"
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "DAEMON\tSTATUS\tUPTIME\tTARGET")
	fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", st.PID, status, formatUptime(st.UptimeSeconds), st.TargetPath)
	if err := w.Flush(); err != nil {
		return err
	}

	// The 3 most recent snapshots come straight from the on-disk manifest.
	m, err := store.Load(paths.Manifest)
	if err != nil {
		return err
	}
	recent := store.SortByTimestampDesc(m.Snapshots)
	if len(recent) > 3 {
		recent = recent[:3]
	}

	fmt.Fprintln(cmd.OutOrStdout(), "\nRECENT SNAPSHOTS")
	sw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(sw, "ID\tTIMESTAMP\tSIZE")
	for _, s := range recent {
		fmt.Fprintf(sw, "%s\t%s\t%d\n", s.ID, s.Timestamp.UTC().Format(time.RFC3339), s.SizeBytes)
	}
	return sw.Flush()
}

func formatUptime(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second)).Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
