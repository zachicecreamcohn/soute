// Package paths centralizes the on-disk layout of a .snapshots tree.
package paths

import "path/filepath"

const (
	// SnapDirName is the hidden directory created adjacent to the target.
	SnapDirName = ".snapshots"
	// ConfigFile holds user settings.
	ConfigFile = "config.json"
	// ManifestFile is the index of all snapshots.
	ManifestFile = "manifest.json"
	// PidFile locks the daemon to a single instance.
	PidFile = "daemon.pid"
	// DataDir is the content-addressed blob store.
	DataDir = "data"
)

// SnapDir returns the .snapshots directory adjacent to base.
func SnapDir(base string) string {
	return filepath.Join(base, SnapDirName)
}

// ConfigPath returns the path to config.json under a snapshots directory.
func ConfigPath(snapDir string) string {
	return filepath.Join(snapDir, ConfigFile)
}

// ManifestPath returns the path to manifest.json under a snapshots directory.
func ManifestPath(snapDir string) string {
	return filepath.Join(snapDir, ManifestFile)
}

// PidPath returns the path to daemon.pid under a snapshots directory.
func PidPath(snapDir string) string {
	return filepath.Join(snapDir, PidFile)
}

// DataDirPath returns the blob store directory under a snapshots directory.
func DataDirPath(snapDir string) string {
	return filepath.Join(snapDir, DataDir)
}

// BlobPath returns the content-addressed object path for a hash.
func BlobPath(snapDir, hash string) string {
	return filepath.Join(snapDir, DataDir, hash)
}
