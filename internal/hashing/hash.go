// Package hashing streams a file through SHA-256, deferring the full read so
// large files never sit in memory.
package hashing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// Result holds the outcome of hashing a file's contents.
type Result struct {
	Hash string // lowercase 64-hex SHA-256 digest
	Size int64  // number of bytes streamed
}

// File streams path through SHA-256 and returns the digest and size.
func File(path string) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return Result{}, fmt.Errorf("hash: %w", err)
	}
	return Result{Hash: hex.EncodeToString(h.Sum(nil)), Size: n}, nil
}
