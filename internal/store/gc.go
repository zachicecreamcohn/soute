package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// PruneResult reports the outcome of a garbage-collection pass.
type PruneResult struct {
	RemovedEntries int
	DeletedBlobs   int
}

// Prune trims the manifest to the newest maxSnapshots entries and deletes any
// blob in dataDir not referenced by a surviving entry. Reference counting is
// derived by scanning the manifest, so a blob shared by two snapshots survives
// while any referencing entry does, and is deleted only when its last
// referencing entry leaves. Orphaned blobs (on disk with no manifest entry)
// are swept as well. The manifest is mutated in place; the caller persists it
// via SaveAtomic.
func Prune(m *Manifest, maxSnapshots int, dataDir string) (PruneResult, error) {
	var res PruneResult

	keepHashes := HashSet(*m)
	if maxSnapshots > 0 && len(m.Snapshots) > maxSnapshots {
		sorted := SortByTimestampDesc(m.Snapshots)
		keep := sorted[:maxSnapshots]
		res.RemovedEntries = len(sorted) - len(keep)
		m.Snapshots = keep
		keepHashes = make(map[string]struct{}, len(keep))
		for _, s := range keep {
			keepHashes[s.ContentHash] = struct{}{}
		}
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return res, fmt.Errorf("read data dir: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !isHexHash(name) {
			continue // skip in-flight temp files and unrelated entries
		}
		if _, ok := keepHashes[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dataDir, name)); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return res, fmt.Errorf("remove orphan blob %s: %w", name, err)
		}
		res.DeletedBlobs++
	}
	return res, nil
}

// isHexHash reports whether name is a 64-char lowercase hex digest (a blob).
func isHexHash(name string) bool {
	if len(name) != 64 {
		return false
	}
	for _, c := range name {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
