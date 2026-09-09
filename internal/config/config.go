// Package config loads and persists the user's .snapshots settings.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"soute/internal/fsutil"
	"soute/internal/paths"
)

// Defaults used when the user accepts the recommended settings.
const (
	DefaultMinDeltaBytes = 1 << 20 // 1 MiB
	DefaultMinDeltaPct   = 2.5
	DefaultCooldownSecs  = 5
	DefaultMaxSnapshots  = 50
)

// Config is the config.json schema.
type Config struct {
	TargetPath      string  `json:"target_path"`
	MinDeltaBytes   int64   `json:"min_delta_bytes"`
	MinDeltaPct     float64 `json:"min_delta_pct"`
	CooldownSeconds int     `json:"cooldown_seconds"`
	MaxSnapshots    int     `json:"max_snapshots"`
}

// Default returns a Config with recommended values and no target.
func Default() Config {
	return Config{
		MinDeltaBytes:   DefaultMinDeltaBytes,
		MinDeltaPct:     DefaultMinDeltaPct,
		CooldownSeconds: DefaultCooldownSecs,
		MaxSnapshots:    DefaultMaxSnapshots,
	}
}

// Load reads and parses config.json under snapDir.
func Load(snapDir string) (*Config, error) {
	b, err := os.ReadFile(paths.ConfigPath(snapDir))
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Save writes config.json under snapDir atomically.
func Save(snapDir string, c *Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return fsutil.WriteFileAtomic(paths.ConfigPath(snapDir), b, 0o644)
}

// TargetAbs resolves the configured target path to an absolute path. Relative
// paths are interpreted against the directory that contains the snapshots
// directory, so the config remains valid regardless of the current directory.
func (c *Config) TargetAbs(snapDir string) string {
	p := c.TargetPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(filepath.Dir(snapDir), p)
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}
