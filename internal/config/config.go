// Package config defines soute's per-target configuration and derives every
// filesystem location from the absolute target path.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zachicecreamcohn/soute/internal/atomicwrite"
)

// Safe defaults applied by `soute init` (spec §6).
const (
	DefaultMaxTimeSeconds = 300
	DefaultDebounceMS     = 300
	DefaultMaxSnapshots   = 50
)

// Config holds user settings for a single watched target file.
type Config struct {
	// TargetPath is the ABSOLUTE path to the monitored project file. It is
	// stored absolute (not relative, as the spec's example suggests) because a
	// detached daemon has no meaningful working directory.
	TargetPath     string `json:"target_path"`
	MaxTimeSeconds int    `json:"max_time_seconds"` // 0 = hash on any mtime change
	DebounceMS     int    `json:"debounce_ms"`
	MaxSnapshots   int    `json:"max_snapshots"`
}

// Default returns a Config for targetAbsPath with safe defaults.
func Default(targetAbsPath string) Config {
	return Config{
		TargetPath:     targetAbsPath,
		MaxTimeSeconds: DefaultMaxTimeSeconds,
		DebounceMS:     DefaultDebounceMS,
		MaxSnapshots:   DefaultMaxSnapshots,
	}
}

// Validate returns an error if the config is unusable.
func (c Config) Validate() error {
	switch {
	case c.TargetPath == "":
		return errors.New("target_path is empty")
	case c.DebounceMS <= 0:
		return errors.New("debounce_ms must be > 0")
	case c.MaxSnapshots <= 0:
		return errors.New("max_snapshots must be > 0")
	case c.MaxTimeSeconds < 0:
		return errors.New("max_time_seconds must be >= 0")
	}
	return nil
}

// Save writes the config atomically to path.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	return atomicwrite.JSON(path, c, 0o600)
}

// Load reads and parses a config from path.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return c, nil
}

// Paths holds every filesystem location derived from the target file.
type Paths struct {
	TargetAbs string
	TargetDir string
	Namespace string
	Config    string
	Manifest  string
	Daemon    string
	DataDir   string
	LogFile   string
}

// ResolvePaths derives all paths from targetPath, made absolute and cleaned.
func ResolvePaths(targetPath string) (Paths, error) {
	abs, err := filepath.Abs(targetPath)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve target path: %w", err)
	}
	abs = filepath.Clean(abs)
	dir := filepath.Dir(abs)
	base := sanitizeName(filepath.Base(abs))
	ns := filepath.Join(dir, ".soute_"+base)
	return Paths{
		TargetAbs: abs,
		TargetDir: dir,
		Namespace: ns,
		Config:    filepath.Join(ns, "config.json"),
		Manifest:  filepath.Join(ns, "manifest.json"),
		Daemon:    filepath.Join(ns, "daemon.json"),
		DataDir:   filepath.Join(ns, "data"),
		LogFile:   filepath.Join(ns, "daemon.log"),
	}, nil
}

// Ensure creates the namespace directory (including the data/ blob dir) with
// owner-only permissions.
func (p Paths) Ensure() error {
	if err := os.MkdirAll(p.DataDir, 0o700); err != nil {
		return fmt.Errorf("create namespace: %w", err)
	}
	return nil
}

// sanitizeName replaces characters unsafe in a directory name with '_'.
func sanitizeName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, s)
}
