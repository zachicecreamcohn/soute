package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func hashN(i int) string { return fmt.Sprintf("%064x", i) }

func writeBlob(t *testing.T, dataDir, hash string) {
	t.Helper()
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, hash), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func blobExists(t *testing.T, dataDir, hash string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dataDir, hash))
	return err == nil
}

func TestPruneTrimsOldest(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	m := NewManifest()
	for i := 0; i < 5; i++ {
		h := hashN(i)
		writeBlob(t, dataDir, h)
		m.Snapshots = append(m.Snapshots, mkSnap(fmt.Sprintf("s%d", i), base.Add(time.Duration(i)*time.Minute), h))
	}

	res, err := Prune(&m, 3, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.RemovedEntries != 2 {
		t.Errorf("RemovedEntries = %d want 2", res.RemovedEntries)
	}
	if res.DeletedBlobs != 2 {
		t.Errorf("DeletedBlobs = %d want 2", res.DeletedBlobs)
	}
	if len(m.Snapshots) != 3 {
		t.Errorf("manifest len = %d want 3", len(m.Snapshots))
	}
	// Newest three (s2, s3, s4) kept; oldest two (s0, s1) blobs removed.
	for _, keep := range []int{2, 3, 4} {
		if !blobExists(t, dataDir, hashN(keep)) {
			t.Errorf("blob %d should be kept", keep)
		}
	}
	for _, del := range []int{0, 1} {
		if blobExists(t, dataDir, hashN(del)) {
			t.Errorf("blob %d should be deleted", del)
		}
	}
}

func TestPruneDedupSharedBlobRetained(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	h := hashN(1)
	writeBlob(t, dataDir, h)

	m := NewManifest()
	m.Snapshots = []Snapshot{
		mkSnap("old", base, h),
		mkSnap("new", base.Add(time.Minute), h),
	}
	res, err := Prune(&m, 1, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.RemovedEntries != 1 {
		t.Errorf("RemovedEntries = %d want 1", res.RemovedEntries)
	}
	if res.DeletedBlobs != 0 {
		t.Errorf("shared blob must not be deleted, got %d deletions", res.DeletedBlobs)
	}
	if !blobExists(t, dataDir, h) {
		t.Error("shared blob missing")
	}
}

func TestPruneOrphanSweepUnderLimit(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	h := hashN(9)
	writeBlob(t, dataDir, h) // orphan: no manifest entry references it

	m := NewManifest()
	res, err := Prune(&m, 50, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.RemovedEntries != 0 {
		t.Errorf("RemovedEntries = %d want 0", res.RemovedEntries)
	}
	if res.DeletedBlobs != 1 {
		t.Errorf("DeletedBlobs = %d want 1", res.DeletedBlobs)
	}
	if blobExists(t, dataDir, h) {
		t.Error("orphan blob not swept")
	}
}

func TestPruneBoundaryNoOp(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	m := NewManifest()
	for i := 0; i < 3; i++ {
		h := hashN(i)
		writeBlob(t, dataDir, h)
		m.Snapshots = append(m.Snapshots, mkSnap(fmt.Sprintf("s%d", i), base.Add(time.Duration(i)*time.Minute), h))
	}
	res, err := Prune(&m, 3, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.RemovedEntries != 0 || res.DeletedBlobs != 0 {
		t.Errorf("expected no-op, got %+v", res)
	}
	if len(m.Snapshots) != 3 {
		t.Errorf("manifest len = %d want 3", len(m.Snapshots))
	}
}

func TestPruneSkipsTempFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	// A leftover in-flight commit temp file (not a 64-hex name) must survive.
	writeBlob(t, dataDir, hashN(1)+".tmp")

	m := NewManifest()
	res, err := Prune(&m, 50, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.DeletedBlobs != 0 {
		t.Errorf("temp file should be skipped, got %d deletions", res.DeletedBlobs)
	}
	if !blobExists(t, dataDir, hashN(1)+".tmp") {
		t.Error("temp file was deleted")
	}
}
