// Package store implements soute's content-addressed snapshot store: the
// manifest index (id → hash) and the blob store, plus reference-counted GC.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/zacharycohn/soute/internal/atomicwrite"
)

// ManifestVersion is the on-disk schema version (spec §3.2).
const ManifestVersion = "1.1"

// Snapshot is a single record in the manifest.
type Snapshot struct {
	ID          string    `json:"id"`           // "snap_<unix-nanos>"
	Timestamp   time.Time `json:"timestamp"`    // snapshot creation time (UTC)
	ContentHash string    `json:"content_hash"` // lowercase 64-hex SHA-256
	SizeBytes   int64     `json:"size_bytes"`
	Mtime       time.Time `json:"mtime"` // source file mtime at snapshot
	Tag         string    `json:"tag"`   // "auto" | "backup"
}

// Manifest is the index mapping snapshot IDs to content hashes.
type Manifest struct {
	Version   string     `json:"version"`
	Snapshots []Snapshot `json:"snapshots"`
}

// NewManifest returns an empty manifest at the current schema version.
func NewManifest() Manifest {
	return Manifest{Version: ManifestVersion, Snapshots: []Snapshot{}}
}

// Load reads and parses a manifest from path. A missing file yields an empty
// manifest (nil snapshots are normalized to an empty slice).
func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewManifest(), nil
		}
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Snapshots == nil {
		m.Snapshots = []Snapshot{}
	}
	return m, nil
}

// SaveAtomic writes the manifest atomically to path.
func SaveAtomic(path string, m Manifest) error {
	if m.Snapshots == nil {
		m.Snapshots = []Snapshot{}
	}
	return atomicwrite.JSON(path, m, 0o600)
}

// Newest returns the snapshot with the latest timestamp.
func Newest(m Manifest) (Snapshot, bool) {
	if len(m.Snapshots) == 0 {
		return Snapshot{}, false
	}
	newest := m.Snapshots[0]
	for _, s := range m.Snapshots[1:] {
		if s.Timestamp.After(newest.Timestamp) {
			newest = s
		}
	}
	return newest, true
}

// FindByID returns the snapshot with the given id.
func FindByID(m Manifest, id string) (Snapshot, bool) {
	for _, s := range m.Snapshots {
		if s.ID == id {
			return s, true
		}
	}
	return Snapshot{}, false
}

// HashSet returns the set of content hashes referenced by the manifest.
func HashSet(m Manifest) map[string]struct{} {
	set := make(map[string]struct{}, len(m.Snapshots))
	for _, s := range m.Snapshots {
		set[s.ContentHash] = struct{}{}
	}
	return set
}

// SortByTimestampDesc returns a copy of snapshots sorted newest-first. The
// input is not modified. Ordering by Timestamp (never by ID) keeps pruning
// correct even if IDs were generated out of order.
func SortByTimestampDesc(snaps []Snapshot) []Snapshot {
	out := make([]Snapshot, len(snaps))
	copy(out, snaps)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	return out
}
