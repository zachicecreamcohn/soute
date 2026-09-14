package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/hashing"
	"github.com/zachicecreamcohn/soute/internal/ipc"
	"github.com/zachicecreamcohn/soute/internal/store"
)

// defaultBackupTag is the tag applied to a manual backup when -t is omitted.
const defaultBackupTag = "Manual Backup"

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup [path]",
		Short: "Create a manual snapshot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			tag, err := cmd.Flags().GetString("tag")
			if err != nil {
				return err
			}
			return runBackup(cmd, arg, tag)
		},
	}
	cmd.Flags().StringP("tag", "t", "", "label for the backup (shown in the restore picker)")
	return cmd
}

// runBackup force-commits a snapshot of the target now, bypassing the daemon's
// debounce / minimum-interval / dedup gates. The blob is still deduplicated on
// disk; only the manifest entry is always appended, so a point in time can be
// pinned and tagged.
func runBackup(cmd *cobra.Command, arg, tag string) error {
	paths, err := resolvePaths(arg)
	if err != nil {
		return err
	}
	if _, err := config.Load(paths.Config); err != nil {
		return fmt.Errorf("load config (run `soute init` first): %w", err)
	}

	fi, err := os.Stat(paths.TargetAbs)
	if err != nil {
		return fmt.Errorf("stat target: %w", err)
	}
	h, err := hashing.File(paths.TargetAbs)
	if err != nil {
		return fmt.Errorf("hash target: %w", err)
	}

	tag = strings.TrimSpace(tag)
	if tag == "" {
		tag = defaultBackupTag
	}

	// Fence the manifest read-modify-write against a running daemon.
	addr := ipc.EndpointFor(paths.Namespace, paths.TargetAbs)
	running, _ := ping(addr)
	if running {
		if err := sendIPC(addr, ipc.CmdPause); err != nil {
			return fmt.Errorf("pause daemon: %w", err)
		}
	}

	if _, err := store.CommitBlob(paths.TargetAbs, paths.DataDir, h.Hash); err != nil {
		return err
	}

	m, err := store.Load(paths.Manifest)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	snap := store.Snapshot{
		ID:          "snap_" + strconv.FormatInt(now.UnixNano(), 10),
		Timestamp:   now,
		ContentHash: h.Hash,
		SizeBytes:   h.Size,
		Mtime:       fi.ModTime(),
		Tag:         tag,
	}
	m.Snapshots = append(m.Snapshots, snap)
	if err := store.SaveAtomic(paths.Manifest, m); err != nil {
		return err
	}

	if running {
		if err := sendIPC(addr, ipc.CmdResume); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: resume failed: %v\n", err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "backed up %s as %s (tag: %s)\n", paths.TargetAbs, snap.ID, tag)
	return nil
}
