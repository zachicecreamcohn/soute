package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/hashing"
	"github.com/zachicecreamcohn/soute/internal/ipc"
	"github.com/zachicecreamcohn/soute/internal/store"
	"github.com/zachicecreamcohn/soute/internal/tui"
	"github.com/zachicecreamcohn/soute/internal/watcher"
)

func newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <path> [id]",
		Short: "Restore a snapshot",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 2 {
				id = args[1]
			}
			return runRestore(cmd, args[0], id)
		},
	}
}

func runRestore(cmd *cobra.Command, pathArg, id string) error {
	paths, err := resolvePaths(pathArg)
	if err != nil {
		return err
	}
	m, err := store.Load(paths.Manifest)
	if err != nil {
		return err
	}
	if len(m.Snapshots) == 0 {
		return fmt.Errorf("no snapshots to restore")
	}

	snap, err := selectSnapshot(m, id)
	if errors.Is(err, tui.ErrCancelled) {
		return nil // user aborted the picker; not an error
	}
	if err != nil {
		return err
	}

	blob, err := store.OpenBlob(paths.DataDir, snap.ContentHash)
	if err != nil {
		return err
	}
	defer blob.Close()

	addr := ipc.EndpointFor(paths.Namespace, paths.TargetAbs)
	running, _ := ping(addr)

	if running {
		if err := sendIPC(addr, ipc.CmdPause); err != nil {
			return fmt.Errorf("pause daemon: %w", err)
		}
	}

	// --- fenced section: the CLI is the sole manifest/data writer ---

	// Pre-flight backup, committed before the swap so any failure leaves the
	// pre-restore state recoverable as a "pre-restore backup" snapshot.
	perm := os.FileMode(0o644)
	if fi, err := os.Stat(paths.TargetAbs); err == nil {
		perm = fi.Mode().Perm()
		h, herr := hashing.File(paths.TargetAbs)
		if herr != nil {
			return fmt.Errorf("backup hash: %w", herr)
		}
		if _, exists := store.HashSet(m)[h.Hash]; !exists {
			if _, err := store.CommitBlob(paths.TargetAbs, paths.DataDir, h.Hash); err != nil {
				return err
			}
			now := time.Now().UTC()
			m.Snapshots = append(m.Snapshots, store.Snapshot{
				ID:          "snap_" + strconv.FormatInt(now.UnixNano(), 10),
				Timestamp:   now,
				ContentHash: h.Hash,
				SizeBytes:   h.Size,
				Mtime:       fi.ModTime(),
				Tag:         "pre-restore backup",
			})
			if err := store.SaveAtomic(paths.Manifest, m); err != nil {
				return err
			}
		}
	}

	// Atomic swap: copy the blob into a temp file in the target's directory
	// (same filesystem → atomic rename), then rename over the target. Copying
	// (rather than renaming the blob) keeps the blob in the store.
	tmp := filepath.Join(paths.TargetDir, ".soute_"+filepath.Base(paths.TargetAbs)+".restore.tmp")
	if err := copyToFile(blob, tmp, perm); err != nil {
		return err
	}
	defer os.Remove(tmp)

	if err := (watcher.Doer{}).Do(func() error { return os.Rename(tmp, paths.TargetAbs) }); err != nil {
		return fmt.Errorf("restore swap: %w (close the application and retry)", err)
	}
	_ = os.Chtimes(paths.TargetAbs, snap.Mtime, snap.Mtime)

	if running {
		if err := sendIPC(addr, ipc.CmdResume); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: resume failed: %v\n", err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "restored %s to snapshot %s\n", paths.TargetAbs, snap.ID)
	return nil
}

// selectSnapshot resolves the target snapshot: by id when given, otherwise via
// the interactive picker (Phase 5).
func selectSnapshot(m store.Manifest, id string) (store.Snapshot, error) {
	if id != "" {
		s, ok := store.FindByID(m, id)
		if !ok {
			return store.Snapshot{}, fmt.Errorf("snapshot %q not found", id)
		}
		return s, nil
	}

	if !isTTY(os.Stdin) {
		return store.Snapshot{}, fmt.Errorf("no snapshot id provided (run interactively to pick one)")
	}

	sorted := store.SortByTimestampDesc(m.Snapshots)
	items := make([]tui.Item, len(sorted))
	for i, s := range sorted {
		items[i] = tui.Item{
			ID:          s.ID,
			Timestamp:   s.Timestamp,
			SizeBytes:   s.SizeBytes,
			Tag:         s.Tag,
			ContentHash: s.ContentHash,
		}
	}
	idx, err := tui.Pick(items)
	if err != nil {
		return store.Snapshot{}, err
	}
	return sorted[idx], nil
}

// isTTY reports whether f is a character device (an interactive terminal).
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// copyToFile copies src to dstPath, fsyncs, and applies perm.
func copyToFile(src *os.File, dstPath string, perm os.FileMode) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return err
	}
	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return err
	}
	if err := os.Chmod(dstPath, perm); err != nil {
		return err
	}
	return nil
}
