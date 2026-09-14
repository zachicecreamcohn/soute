package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BlobPath returns the on-disk path of a blob in dataDir.
func BlobPath(dataDir, hash string) string {
	return filepath.Join(dataDir, hash)
}

// HasBlob reports whether the blob exists as a regular file.
func HasBlob(dataDir, hash string) bool {
	fi, err := os.Stat(BlobPath(dataDir, hash))
	return err == nil && fi.Mode().IsRegular()
}

// CommitBlob copies srcPath into the blob store at dataDir/<hash>. The copy is
// written to <hash>.tmp, fsynced, and atomically renamed to <hash>. If the blob
// already exists, it returns existed=true without copying (content addressing
// deduplicates identical saves).
func CommitBlob(srcPath, dataDir, hash string) (existed bool, err error) {
	dst := BlobPath(dataDir, hash)
	if HasBlob(dataDir, hash) {
		return true, nil
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return false, fmt.Errorf("mkdir data: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return false, fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return false, fmt.Errorf("create temp: %w", err)
	}
	cleanup := func() { _ = os.Remove(tmp) }

	if _, err := io.Copy(f, src); err != nil {
		_ = f.Close()
		cleanup()
		return false, fmt.Errorf("copy: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		cleanup()
		return false, fmt.Errorf("sync: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return false, fmt.Errorf("close: %w", err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		cleanup()
		return false, fmt.Errorf("chmod: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		cleanup()
		return false, fmt.Errorf("rename: %w", err)
	}
	return false, nil
}

// OpenBlob opens an existing blob for reading.
func OpenBlob(dataDir, hash string) (*os.File, error) {
	f, err := os.Open(BlobPath(dataDir, hash))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("blob %s not found", hash)
		}
		return nil, fmt.Errorf("open blob: %w", err)
	}
	return f, nil
}
