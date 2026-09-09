package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"soute/internal/config"
	"soute/internal/manifest"
)

// setup creates a snapshots dir plus a target file, returning the engine and
// the target path.
func setup(t *testing.T, cfg config.Config, content []byte) (*Engine, string) {
	t.Helper()
	dir := t.TempDir()
	snap := filepath.Join(dir, ".snapshots")
	if err := os.MkdirAll(filepath.Join(snap, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg.TargetPath = "target.bin"
	if err := config.Save(snap, &cfg); err != nil {
		t.Fatal(err)
	}
	if err := manifest.New().Save(snap); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "target.bin")
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return New(snap), target
}

func writeTarget(t *testing.T, target string, content []byte) {
	t.Helper()
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func loadManifest(t *testing.T, e *Engine) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Load(e.SnapDir)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestEvaluateFirstSnapshot(t *testing.T) {
	e, _ := setup(t, config.Default(), []byte("baseline"))
	res, snap, err := e.Evaluate(time.Now())
	if err != nil || res != EvalCaptured || snap == nil {
		t.Fatalf("Evaluate = %v, %v, %v; want captured", res, snap, err)
	}

	m := loadManifest(t, e)
	if len(m.Snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(m.Snapshots))
	}
	last := m.Snapshots[0]
	if last.Tag != manifest.TagAuto {
		t.Fatalf("tag = %q, want %q", last.Tag, manifest.TagAuto)
	}
	if last.SizeBytes != int64(len("baseline")) {
		t.Fatalf("size = %d, want %d", last.SizeBytes, len("baseline"))
	}
	if last.DeltaBytes != 0 || last.DeltaPct != 0 {
		t.Fatalf("first snapshot deltas = %d, %f; want 0, 0", last.DeltaBytes, last.DeltaPct)
	}
}

func TestEvaluateBelowThreshold(t *testing.T) {
	cfg := config.Default()
	cfg.MinDeltaBytes = 1 << 20
	cfg.MinDeltaPct = 100
	e, target := setup(t, cfg, []byte("1234567890"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	writeTarget(t, target, []byte("1234567890X")) // +1 byte, well under 1 MiB/100%

	res, snap, err := e.Evaluate(time.Now().Add(10 * time.Second))
	if err != nil || res != EvalSkipped || snap != nil {
		t.Fatalf("Evaluate = %v, %v, %v; want skipped", res, snap, err)
	}
}

func TestEvaluatePercentThreshold(t *testing.T) {
	cfg := config.Default()
	cfg.MinDeltaBytes = 1 << 30 // force the pct path
	cfg.MinDeltaPct = 10
	e, target := setup(t, cfg, []byte("1234567890"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	writeTarget(t, target, []byte("12345678901234")) // +40%

	res, snap, err := e.Evaluate(time.Now().Add(10 * time.Second))
	if err != nil || res != EvalCaptured || snap == nil {
		t.Fatalf("Evaluate = %v, %v, %v; want captured", res, snap, err)
	}
	if snap.DeltaPct < 10 {
		t.Fatalf("delta_pct = %f, want >= 10", snap.DeltaPct)
	}
}

func TestEvaluateHashDedup(t *testing.T) {
	cfg := config.Default()
	cfg.MinDeltaBytes = 0
	cfg.MinDeltaPct = 0
	e, target := setup(t, cfg, []byte("same content"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	// Identical save must not create a second snapshot.
	writeTarget(t, target, []byte("same content"))
	res, snap, err := e.Evaluate(time.Now().Add(10 * time.Second))
	if err != nil || res != EvalSkipped || snap != nil {
		t.Fatalf("identical save = %v, %v, %v; want skipped", res, snap, err)
	}
	if m := loadManifest(t, e); len(m.Snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(m.Snapshots))
	}
}

func TestEvaluateCooldown(t *testing.T) {
	cfg := config.Default()
	cfg.CooldownSeconds = 100
	e, _ := setup(t, cfg, []byte("first"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	res, _, err := e.Evaluate(time.Now().Add(5 * time.Second))
	if err != nil || res != EvalCooldown {
		t.Fatalf("within cooldown = %v, %v; want EvalCooldown", res, err)
	}
	// After the window lapses, a real change captures normally.
	res, _, err = e.Evaluate(time.Now().Add(200 * time.Second))
	if err != nil || res != EvalSkipped {
		t.Fatalf("after cooldown (no change) = %v, %v; want skipped", res, err)
	}
}

func TestGarbageCollection(t *testing.T) {
	cfg := config.Default()
	cfg.MaxSnapshots = 2
	cfg.CooldownSeconds = 0
	cfg.MinDeltaBytes = 0
	cfg.MinDeltaPct = 0
	e, target := setup(t, cfg, []byte("v1"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	v1Hash := loadManifest(t, e).Snapshots[0].ContentHash

	for i, c := range [][]byte{[]byte("v2"), []byte("v3")} {
		writeTarget(t, target, c)
		if _, _, err := e.Evaluate(time.Now().Add(time.Duration(i+2) * time.Second)); err != nil {
			t.Fatal(err)
		}
	}

	m := loadManifest(t, e)
	if len(m.Snapshots) != 2 {
		t.Fatalf("snapshot count = %d, want 2", len(m.Snapshots))
	}
	if _, err := os.Stat(filepath.Join(e.SnapDir, "data", v1Hash)); !os.IsNotExist(err) {
		t.Fatalf("oldest blob should be collected, stat err = %v", err)
	}
	for _, s := range m.Snapshots {
		if _, err := os.Stat(filepath.Join(e.SnapDir, "data", s.ContentHash)); err != nil {
			t.Fatalf("retained blob %s missing: %v", s.ContentHash, err)
		}
	}
}

func TestPrune(t *testing.T) {
	cfg := config.Default()
	cfg.MaxSnapshots = 10 // no auto-GC during setup
	cfg.CooldownSeconds = 0
	cfg.MinDeltaBytes = 0
	cfg.MinDeltaPct = 0
	e, target := setup(t, cfg, []byte("a"))

	for i, c := range [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")} {
		writeTarget(t, target, c)
		if _, _, err := e.Evaluate(time.Now().Add(time.Duration(i+1) * time.Second)); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := e.Prune(2)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if m := loadManifest(t, e); len(m.Snapshots) != 2 {
		t.Fatalf("snapshot count = %d, want 2", len(m.Snapshots))
	}
}

func TestPruneAll(t *testing.T) {
	cfg := config.Default()
	cfg.MaxSnapshots = 10
	cfg.CooldownSeconds = 0
	cfg.MinDeltaBytes = 0
	cfg.MinDeltaPct = 0
	e, target := setup(t, cfg, []byte("a"))

	for i, c := range [][]byte{[]byte("a"), []byte("b"), []byte("c")} {
		writeTarget(t, target, c)
		if _, _, err := e.Evaluate(time.Now().Add(time.Duration(i+1) * time.Second)); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := e.PruneAll()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	m := loadManifest(t, e)
	if len(m.Snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(m.Snapshots))
	}
}

func TestPreRestoreBackup(t *testing.T) {
	cfg := config.Default()
	cfg.CooldownSeconds = 0
	cfg.MinDeltaBytes = 0
	cfg.MinDeltaPct = 0
	e, target := setup(t, cfg, []byte("saved"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	// Unsaved change since the last snapshot.
	writeTarget(t, target, []byte("unsaved work"))

	backedUp, err := e.PreRestoreBackup()
	if err != nil || !backedUp {
		t.Fatalf("PreRestoreBackup = %v, %v; want true, nil", backedUp, err)
	}
	m := loadManifest(t, e)
	if len(m.Snapshots) != 2 {
		t.Fatalf("snapshot count = %d, want 2", len(m.Snapshots))
	}
	if m.Snapshots[1].Tag != manifest.TagPreRestoreBackup {
		t.Fatalf("tag = %q, want %q", m.Snapshots[1].Tag, manifest.TagPreRestoreBackup)
	}
	// No further change, so a second call must be a no-op.
	if backedUp, err = e.PreRestoreBackup(); err != nil || backedUp {
		t.Fatalf("second PreRestoreBackup = %v, %v; want false, nil", backedUp, err)
	}
}

func TestRestore(t *testing.T) {
	cfg := config.Default()
	cfg.CooldownSeconds = 0
	cfg.MinDeltaBytes = 0
	cfg.MinDeltaPct = 0
	e, target := setup(t, cfg, []byte("v1"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	writeTarget(t, target, []byte("v2-longer"))
	if _, _, err := e.Evaluate(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	v1 := loadManifest(t, e).Snapshots[0]

	// Simulate unsaved edits, then restore v1.
	writeTarget(t, target, []byte("unsaved-v3-edits"))
	if _, err := e.PreRestoreBackup(); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Restore(v1.ID); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v1" {
		t.Fatalf("target = %q, want %q", got, "v1")
	}

	m := loadManifest(t, e)
	if len(m.Snapshots) != 4 {
		t.Fatalf("snapshot count = %d, want 4", len(m.Snapshots))
	}
	if m.Snapshots[2].Tag != manifest.TagPreRestoreBackup {
		t.Fatalf("entry 2 tag = %q, want pre_restore_backup", m.Snapshots[2].Tag)
	}
	if m.Snapshots[3].Tag != manifest.TagRestore {
		t.Fatalf("entry 3 tag = %q, want restore", m.Snapshots[3].Tag)
	}
	if m.Snapshots[3].ContentHash != v1.ContentHash {
		t.Fatal("restore entry hash does not match the restored snapshot")
	}
}

func TestExport(t *testing.T) {
	cfg := config.Default()
	e, target := setup(t, cfg, []byte("export content"))

	if _, _, err := e.Evaluate(time.Now()); err != nil {
		t.Fatal(err)
	}
	id := loadManifest(t, e).Snapshots[0].ID

	dst := filepath.Join(t.TempDir(), "out.bin")
	if err := e.Export(id, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "export content" {
		t.Fatalf("export = %q, want %q", got, "export content")
	}
	if gotTarget, _ := os.ReadFile(target); string(gotTarget) != "export content" {
		t.Fatal("export mutated the working file")
	}
}
