package store

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestCommitBlob(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	src := filepath.Join(dir, "src.bin")
	content := []byte("blob content")
	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256Hex(content)

	existed, err := CommitBlob(src, dataDir, hash)
	if err != nil {
		t.Fatalf("CommitBlob: %v", err)
	}
	if existed {
		t.Error("expected existed=false on first commit")
	}

	got, err := os.ReadFile(BlobPath(dataDir, hash))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("blob content = %q want %q", got, content)
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 blob (no .tmp residue), got %d", len(entries))
	}
}

func TestCommitBlobDedup(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	src := filepath.Join(dir, "src.bin")
	content := []byte("same content")
	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256Hex(content)

	if _, err := CommitBlob(src, dataDir, hash); err != nil {
		t.Fatal(err)
	}
	// Rewrite source to prove a second commit does not touch the blob.
	if err := os.WriteFile(src, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	existed, err := CommitBlob(src, dataDir, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !existed {
		t.Error("expected existed=true on dedup commit")
	}
	got, _ := os.ReadFile(BlobPath(dataDir, hash))
	if string(got) != string(content) {
		t.Errorf("blob was rewritten: %q", got)
	}
}

func TestOpenBlob(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	src := filepath.Join(dir, "src.bin")
	content := []byte("payload")
	os.WriteFile(src, content, 0o600)
	hash := sha256Hex(content)
	if _, err := CommitBlob(src, dataDir, hash); err != nil {
		t.Fatal(err)
	}

	f, err := OpenBlob(dataDir, hash)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if string(got) != string(content) {
		t.Errorf("content = %q", got)
	}

	if _, err := OpenBlob(dataDir, sha256Hex([]byte("missing"))); err == nil {
		t.Error("expected error for missing blob")
	}
}
