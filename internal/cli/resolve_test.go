package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zachicecreamcohn/soute/internal/config"
)

func TestResolvePathsExplicit(t *testing.T) {
	target := filepath.Join(t.TempDir(), "x.als")
	p, err := resolvePaths(target)
	if err != nil {
		t.Fatalf("resolvePaths: %v", err)
	}
	if p.TargetAbs != target {
		t.Errorf("TargetAbs = %q want %q", p.TargetAbs, target)
	}
}

func TestAutoDetectSingle(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "MainShow.qlab5")
	ns := filepath.Join(dir, ".soute_MainShow.qlab5")
	if err := os.MkdirAll(ns, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Default(target).Save(filepath.Join(ns, "config.json")); err != nil {
		t.Fatal(err)
	}

	p, err := autoDetect(dir)
	if err != nil {
		t.Fatalf("autoDetect: %v", err)
	}
	if p.TargetAbs != target {
		t.Errorf("TargetAbs = %q want %q", p.TargetAbs, target)
	}
}

func TestAutoDetectNone(t *testing.T) {
	if _, err := autoDetect(t.TempDir()); err == nil {
		t.Fatal("expected error with no .soute_* directory")
	}
}

func TestAutoDetectMultiple(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".soute_a", ".soute_b"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := autoDetect(dir); err == nil {
		t.Fatal("expected error with multiple .soute_* directories")
	}
}
