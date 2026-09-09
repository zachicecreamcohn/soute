// Package store implements the content-addressed blob store under
// .snapshots/data, backed by SHA-256 object names.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zachicecreamcohn/soute/internal/paths"
)

// HashFile returns the lowercase hex SHA-256 of the file at path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Exists reports whether a blob is already stored.
func Exists(snapDir, hash string) (bool, error) {
	_, err := os.Stat(paths.BlobPath(snapDir, hash))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Put streams src into the store under its content hash. If the blob already
// exists, Put is a no-op (identical saves never duplicate storage). The copy
// is written to <hash>.tmp and atomically renamed to <hash> so a crash can
// never leave a partial object behind.
func Put(snapDir, hash, src string) error {
	exists, err := Exists(snapDir, hash)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	dst := paths.BlobPath(snapDir, hash)
	if err := os.MkdirAll(paths.DataDirPath(snapDir), 0o755); err != nil {
		return err
	}

	tmp := dst + ".tmp"
	if err := copyFile(tmp, src); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename blob: %w", err)
	}
	return nil
}

// RestoreTo copies a stored blob to dst atomically (dst.tmp then rename), so
// the destination file is never left half-written.
func RestoreTo(snapDir, hash, dst string) error {
	src := paths.BlobPath(snapDir, hash)
	if err := ensureBlob(src, hash); err != nil {
		return err
	}

	tmp := dst + ".tmp"
	if err := copyFile(tmp, src); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ExportTo copies a stored blob directly to dst, leaving the working file and
// daemon untouched.
func ExportTo(snapDir, hash, dst string) error {
	src := paths.BlobPath(snapDir, hash)
	if err := ensureBlob(src, hash); err != nil {
		return err
	}
	return copyFile(dst, src)
}

// Delete removes a stored blob, treating a missing blob as a no-op.
func Delete(snapDir, hash string) error {
	err := os.Remove(paths.BlobPath(snapDir, hash))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func ensureBlob(src, hash string) error {
	if _, err := os.Stat(src); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("blob %s is missing from the store", hash)
		}
		return err
	}
	return nil
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}
