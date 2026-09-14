// Package atomicwrite writes files atomically via a temp file in the same
// directory followed by fsync and os.Rename. A crash leaves either the old or
// the new complete file, never a torn one.
package atomicwrite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSON marshals v with indentation and writes it atomically to path with the
// given file mode.
func JSON(path string, v any, perm os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')
	return Bytes(path, data, perm)
}

// Bytes writes data atomically to path with the given file mode.
func Bytes(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".soute-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
