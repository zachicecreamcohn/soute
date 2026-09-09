// Package manifest models manifest.json, the source of truth for snapshot
// history and the reconstruction timeline.
package manifest

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"time"

	"soute/internal/fsutil"
	"soute/internal/paths"
)

// Version is the manifest schema version.
const Version = "1.0"

// Snapshot tags.
const (
	TagAuto             = "auto"
	TagManual           = "manual"
	TagPreRestoreBackup = "pre_restore_backup"
	TagRestore          = "restore"
)

// Snapshot is one entry in the manifest.
type Snapshot struct {
	ID          string  `json:"id"`
	Timestamp   string  `json:"timestamp"`
	ContentHash string  `json:"content_hash"`
	SizeBytes   int64   `json:"size_bytes"`
	DeltaBytes  int64   `json:"delta_bytes"`
	DeltaPct    float64 `json:"delta_pct"`
	Tag         string  `json:"tag"`
}

// Manifest is the index database mapping ids to hashes.
type Manifest struct {
	Version   string     `json:"version"`
	Snapshots []Snapshot `json:"snapshots"`
}

// New returns an empty manifest.
func New() *Manifest {
	return &Manifest{Version: Version, Snapshots: []Snapshot{}}
}

// Load reads manifest.json under snapDir. A missing file yields an empty
// manifest so freshly initialized trees behave sensibly.
func Load(snapDir string) (*Manifest, error) {
	b, err := os.ReadFile(paths.ManifestPath(snapDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.Snapshots == nil {
		m.Snapshots = []Snapshot{}
	}
	if m.Version == "" {
		m.Version = Version
	}
	return &m, nil
}

// Save writes manifest.json under snapDir atomically.
func (m *Manifest) Save(snapDir string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return fsutil.WriteFileAtomic(paths.ManifestPath(snapDir), b, 0o644)
}

// Last returns the most recent snapshot, if any.
func (m *Manifest) Last() (Snapshot, bool) {
	if len(m.Snapshots) == 0 {
		return Snapshot{}, false
	}
	return m.Snapshots[len(m.Snapshots)-1], true
}

// Append adds a snapshot to the end of the history.
func (m *Manifest) Append(s Snapshot) {
	m.Snapshots = append(m.Snapshots, s)
}

// NextID returns a unique snapshot id based on the current unix time.
func (m *Manifest) NextID(now time.Time) string {
	base := "snap_" + strconv.FormatInt(now.Unix(), 10)
	id := base
	for i := 2; m.hasID(id); i++ {
		id = base + "_" + strconv.Itoa(i)
	}
	return id
}

// Trim keeps only the newest keep snapshots and returns the removed ones so
// their (now possibly orphaned) blobs can be garbage collected.
func (m *Manifest) Trim(keep int) []Snapshot {
	if keep <= 0 || len(m.Snapshots) <= keep {
		return nil
	}
	cut := len(m.Snapshots) - keep
	removed := m.Snapshots[:cut]
	m.Snapshots = m.Snapshots[cut:]
	return removed
}

// Hashes returns the set of content hashes still referenced by the manifest.
func (m *Manifest) Hashes() map[string]bool {
	set := make(map[string]bool, len(m.Snapshots))
	for _, s := range m.Snapshots {
		set[s.ContentHash] = true
	}
	return set
}

func (m *Manifest) hasID(id string) bool {
	for _, s := range m.Snapshots {
		if s.ID == id {
			return true
		}
	}
	return false
}
