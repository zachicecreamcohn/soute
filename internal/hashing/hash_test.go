package hashing

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestFile(t *testing.T) {
	content := []byte("hello soute\nwith some content\x00\x01binary")
	path := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := File(path)
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	sum := sha256.Sum256(content)
	if want := hex.EncodeToString(sum[:]); r.Hash != want {
		t.Errorf("Hash = %q want %q", r.Hash, want)
	}
	if r.Size != int64(len(content)) {
		t.Errorf("Size = %d want %d", r.Size, len(content))
	}
}

func TestFileEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := File(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(nil)
	if want := hex.EncodeToString(sum[:]); r.Hash != want {
		t.Errorf("Hash = %q want %q", r.Hash, want)
	}
	if r.Size != 0 {
		t.Errorf("Size = %d want 0", r.Size)
	}
}

func TestFileMissing(t *testing.T) {
	if _, err := File(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
