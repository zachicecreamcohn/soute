package store

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func digest(t *testing.T, content []byte) string {
	t.Helper()
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.bin")
	content := []byte("hello world")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := HashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := digest(t, content); got != want {
		t.Fatalf("HashFile = %q, want %q", got, want)
	}
}

func TestPutAndExists(t *testing.T) {
	dir := t.TempDir()
	snap := filepath.Join(dir, ".snapshots")
	src := filepath.Join(dir, "src.bin")
	content := []byte("content-addressed")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	hash := digest(t, content)

	if ok, err := Exists(snap, hash); err != nil || ok {
		t.Fatalf("Exists before Put = %v, %v; want false, nil", ok, err)
	}
	if err := Put(snap, hash, src); err != nil {
		t.Fatal(err)
	}
	if ok, err := Exists(snap, hash); err != nil || !ok {
		t.Fatalf("Exists after Put = %v, %v; want true, nil", ok, err)
	}

	blob := filepath.Join(snap, "data", hash)
	got, err := os.ReadFile(blob)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("blob = %q, want %q", got, content)
	}
}

func TestPutDedupDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	snap := filepath.Join(dir, ".snapshots")
	src := filepath.Join(dir, "src.bin")
	original := []byte("original bytes")
	if err := os.WriteFile(src, original, 0o644); err != nil {
		t.Fatal(err)
	}
	hash := digest(t, original)

	if err := Put(snap, hash, src); err != nil {
		t.Fatal(err)
	}
	// Change src and Put again; the stored blob must remain the original.
	if err := os.WriteFile(src, []byte("something else entirely"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Put(snap, hash, src); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(snap, "data", hash))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("blob mutated by dedup Put: %q", got)
	}
}

func TestRestoreToIsAtomic(t *testing.T) {
	dir := t.TempDir()
	snap := filepath.Join(dir, ".snapshots")
	src := filepath.Join(dir, "src.bin")
	content := []byte("restore target")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	hash := digest(t, content)
	if err := Put(snap, hash, src); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "working.bin")
	if err := os.WriteFile(dst, []byte("old state"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RestoreTo(snap, hash, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("dst = %q, want %q", got, content)
	}
	if _, err := os.Stat(dst + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file left behind: %v", err)
	}
}

func TestExportTo(t *testing.T) {
	dir := t.TempDir()
	snap := filepath.Join(dir, ".snapshots")
	src := filepath.Join(dir, "src.bin")
	content := []byte("export me")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	hash := digest(t, content)
	if err := Put(snap, hash, src); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "nested", "export.bin")
	if err := ExportTo(snap, hash, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("dst = %q, want %q", got, content)
	}
}

func TestDeleteMissingIsNoop(t *testing.T) {
	if err := Delete(t.TempDir(), "deadbeef"); err != nil {
		t.Fatalf("Delete of missing blob: %v", err)
	}
}
