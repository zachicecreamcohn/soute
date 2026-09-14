package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zacharycohn/soute/internal/config"
)

// resolvePaths returns the derived Paths for an explicit target arg, or
// auto-detects the target from a single .soute_* directory in the working
// directory (spec §6 note).
func resolvePaths(arg string) (config.Paths, error) {
	if arg != "" {
		return config.ResolvePaths(arg)
	}
	return autoDetect(".")
}

// autoDetect locates a lone .soute_* directory under dir and reads its
// config.json to recover the authoritative target path.
func autoDetect(dir string) (config.Paths, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return config.Paths{}, fmt.Errorf("read directory: %w", err)
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), ".soute_") {
			matches = append(matches, e.Name())
		}
	}
	switch len(matches) {
	case 0:
		return config.Paths{}, errors.New("no .soute_* directory found; provide a <path> argument or run from the target's directory")
	case 1:
		cfg, err := config.Load(filepath.Join(dir, matches[0], "config.json"))
		if err != nil {
			return config.Paths{}, fmt.Errorf("read config for %s: %w", matches[0], err)
		}
		return config.ResolvePaths(cfg.TargetPath)
	default:
		return config.Paths{}, fmt.Errorf("multiple .soute_* directories found (%s); specify a <path>", strings.Join(matches, ", "))
	}
}
