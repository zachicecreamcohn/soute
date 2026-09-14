package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mkSnap(id string, ts time.Time, hash string) Snapshot {
	return Snapshot{ID: id, Timestamp: ts, ContentHash: hash, SizeBytes: 100, Mtime: ts, Tag: "auto"}
}

func TestNewManifestVersion(t *testing.T) {
	if got := NewManifest().Version; got != ManifestVersion {
		t.Errorf("Version = %q want %q", got, ManifestVersion)
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	m, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Version != ManifestVersion {
		t.Errorf("Version = %q", m.Version)
	}
	if len(m.Snapshots) != 0 {
		t.Errorf("expected empty snapshots, got %d", len(m.Snapshots))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	base := time.Date(2026, 9, 8, 15, 52, 0, 0, time.UTC)

	m := NewManifest()
	m.Snapshots = []Snapshot{
		mkSnap("snap_1", base, "a1b2"),
		mkSnap("snap_2", base.Add(time.Minute), "c3d4"),
	}
	if err := SaveAtomic(path, m); err != nil {
		t.Fatalf("SaveAtomic: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Snapshots) != 2 {
		t.Fatalf("len = %d want 2", len(got.Snapshots))
	}
	if got.Snapshots[0].ID != "snap_1" {
		t.Errorf("order not preserved: first = %q", got.Snapshots[0].ID)
	}
	if got.Snapshots[1].ContentHash != "c3d4" {
		t.Errorf("second hash = %q", got.Snapshots[1].ContentHash)
	}

	// Atomic write leaves no .tmp residue.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected only manifest.json, found %d entries", len(entries))
	}
}

func TestNewest(t *testing.T) {
	if _, ok := Newest(NewManifest()); ok {
		t.Fatal("Newest on empty manifest should return ok=false")
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := NewManifest()
	m.Snapshots = []Snapshot{
		mkSnap("a", base, "h1"),
		mkSnap("b", base.Add(time.Hour), "h2"),
		mkSnap("c", base.Add(30*time.Minute), "h3"),
	}
	got, ok := Newest(m)
	if !ok || got.ID != "b" {
		t.Errorf("Newest = %q want b", got.ID)
	}
}

func TestFindByID(t *testing.T) {
	m := NewManifest()
	m.Snapshots = []Snapshot{mkSnap("snap_x", time.Now(), "h")}
	if _, ok := FindByID(m, "snap_x"); !ok {
		t.Fatal("expected to find snap_x")
	}
	if _, ok := FindByID(m, "missing"); ok {
		t.Fatal("did not expect to find missing")
	}
}

func TestHashSet(t *testing.T) {
	m := NewManifest()
	m.Snapshots = []Snapshot{
		mkSnap("a", time.Now(), "h1"),
		mkSnap("b", time.Now(), "h2"),
		mkSnap("c", time.Now(), "h1"), // duplicate hash
	}
	set := HashSet(m)
	if len(set) != 2 {
		t.Errorf("HashSet len = %d want 2", len(set))
	}
}

func TestSortByTimestampDesc(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	in := []Snapshot{
		mkSnap("old", base, "h1"),
		mkSnap("new", base.Add(time.Hour), "h2"),
		mkSnap("mid", base.Add(time.Minute), "h3"),
	}
	out := SortByTimestampDesc(in)
	if out[0].ID != "new" || out[1].ID != "mid" || out[2].ID != "old" {
		t.Errorf("wrong order: %v", []string{out[0].ID, out[1].ID, out[2].ID})
	}
	// Original slice untouched.
	if in[0].ID != "old" {
		t.Errorf("input modified")
	}
}
