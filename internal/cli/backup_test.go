package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/store"
)

// setupBackupTarget creates an initialized target file for backup tests.
func setupBackupTarget(t *testing.T, content string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "show.qlab5")
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := config.ResolvePaths(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := config.Default(paths.TargetAbs).Save(paths.Config); err != nil {
		t.Fatal(err)
	}
	return target
}

func discardCmd() *cobra.Command {
	c := &cobra.Command{}
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	return c
}

func loadManifest(t *testing.T, target string) store.Manifest {
	t.Helper()
	paths, err := config.ResolvePaths(target)
	if err != nil {
		t.Fatal(err)
	}
	m, err := store.Load(paths.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestBackupWithTag(t *testing.T) {
	target := setupBackupTarget(t, "hello world")
	if err := runBackup(discardCmd(), target, "before the show"); err != nil {
		t.Fatalf("runBackup: %v", err)
	}

	m := loadManifest(t, target)
	if len(m.Snapshots) != 1 {
		t.Fatalf("len = %d want 1", len(m.Snapshots))
	}
	if got := m.Snapshots[0].Tag; got != "before the show" {
		t.Errorf("Tag = %q want %q", got, "before the show")
	}
	paths, _ := config.ResolvePaths(target)
	if !store.HasBlob(paths.DataDir, m.Snapshots[0].ContentHash) {
		t.Error("blob missing from store")
	}
}

func TestBackupDefaultTag(t *testing.T) {
	target := setupBackupTarget(t, "hello world")
	if err := runBackup(discardCmd(), target, ""); err != nil {
		t.Fatalf("runBackup: %v", err)
	}
	m := loadManifest(t, target)
	if got := m.Snapshots[0].Tag; got != defaultBackupTag {
		t.Errorf("Tag = %q want %q", got, defaultBackupTag)
	}
}

func TestBackupWhitespaceTagDefaults(t *testing.T) {
	target := setupBackupTarget(t, "hello world")
	if err := runBackup(discardCmd(), target, "   "); err != nil {
		t.Fatalf("runBackup: %v", err)
	}
	m := loadManifest(t, target)
	if got := m.Snapshots[0].Tag; got != defaultBackupTag {
		t.Errorf("Tag = %q want %q", got, defaultBackupTag)
	}
}

func TestBackupAlwaysAppendsAndDedups(t *testing.T) {
	target := setupBackupTarget(t, "hello world")
	if err := runBackup(discardCmd(), target, "first"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := runBackup(discardCmd(), target, "second"); err != nil {
		t.Fatalf("second: %v", err)
	}

	m := loadManifest(t, target)
	if len(m.Snapshots) != 2 {
		t.Fatalf("len = %d want 2", len(m.Snapshots))
	}
	if m.Snapshots[0].ContentHash != m.Snapshots[1].ContentHash {
		t.Errorf("expected shared hash, got %q vs %q", m.Snapshots[0].ContentHash, m.Snapshots[1].ContentHash)
	}
	if m.Snapshots[0].Tag == m.Snapshots[1].Tag {
		t.Errorf("expected distinct tags, both %q", m.Snapshots[0].Tag)
	}
}
