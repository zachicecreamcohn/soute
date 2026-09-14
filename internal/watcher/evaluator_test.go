package watcher

import (
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/hashing"
	"github.com/zachicecreamcohn/soute/internal/store"
)

type fakeInfo struct{ mtime time.Time }

func (f fakeInfo) Name() string       { return "show.qlab5" }
func (f fakeInfo) Size() int64        { return 10 }
func (f fakeInfo) Mode() fs.FileMode  { return 0o644 }
func (f fakeInfo) ModTime() time.Time { return f.mtime }
func (f fakeInfo) IsDir() bool        { return false }
func (f fakeInfo) Sys() any           { return nil }

type fakeDeps struct {
	now           time.Time
	statMtime     time.Time
	statErr       error
	hash          string
	hashErr       error
	hashCalls     int
	manifest      store.Manifest
	saved         *store.Manifest
	committedBlob []string
	commitErr     error
}

func (f *fakeDeps) Now() time.Time { return f.now }
func (f *fakeDeps) Stat(string) (fs.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return fakeInfo{f.statMtime}, nil
}
func (f *fakeDeps) HashFile(string) (hashing.Result, error) {
	f.hashCalls++
	if f.hashErr != nil {
		return hashing.Result{}, f.hashErr
	}
	return hashing.Result{Hash: f.hash, Size: 10}, nil
}
func (f *fakeDeps) LoadManifest() (store.Manifest, error) { return f.manifest, nil }
func (f *fakeDeps) SaveManifest(m store.Manifest) error   { f.saved = &m; return nil }
func (f *fakeDeps) CommitBlob(_, _, hash string) (bool, error) {
	if f.commitErr != nil {
		return false, f.commitErr
	}
	f.committedBlob = append(f.committedBlob, hash)
	return false, nil
}
func (f *fakeDeps) Retry(fn func() error) error { return fn() }

func newTestEval(f *fakeDeps) *Evaluator {
	return &Evaluator{
		TargetPath: "/x/show.qlab5",
		DataDir:    "/x/.soute_show.qlab5/data",
		Config:     config.Default("/x/show.qlab5"),
		Deps: Deps{
			Now:          f.Now,
			Stat:         f.Stat,
			HashFile:     f.HashFile,
			LoadManifest: f.LoadManifest,
			SaveManifest: f.SaveManifest,
			CommitBlob:   f.CommitBlob,
			Retry:        f.Retry,
		},
	}
}

func TestEvaluateSkippedPaused(t *testing.T) {
	f := &fakeDeps{now: time.Now()}
	e := newTestEval(f)
	e.Pause()

	out, _, err := e.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if out != OutcomeSkippedPaused {
		t.Errorf("out = %q want %q", out, OutcomeSkippedPaused)
	}
	if f.hashCalls != 0 {
		t.Errorf("hash called %d times", f.hashCalls)
	}
}

func TestEvaluateAbortedMTime(t *testing.T) {
	mt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f := &fakeDeps{now: mt.Add(time.Hour), statMtime: mt}
	e := newTestEval(f)
	e.lastMtime = mt

	out, _, err := e.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if out != OutcomeAbortedMTime {
		t.Errorf("out = %q want %q", out, OutcomeAbortedMTime)
	}
	if f.hashCalls != 0 {
		t.Errorf("hash should not run on unchanged mtime")
	}
}

func TestEvaluateAbortedTime(t *testing.T) {
	now := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	f := &fakeDeps{now: now, statMtime: now} // mtime differs from zero lastMtime
	e := newTestEval(f)
	e.lastSnapshotTime = now.Add(-10 * time.Second) // within 300s window

	out, _, err := e.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if out != OutcomeAbortedTime {
		t.Errorf("out = %q want %q", out, OutcomeAbortedTime)
	}
	if f.hashCalls != 0 {
		t.Errorf("hash should not run within time window")
	}
}

func TestEvaluateAbortedHashDup(t *testing.T) {
	now := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	f := &fakeDeps{now: now, statMtime: now, hash: "same"}
	e := newTestEval(f)
	e.lastSnapshotTime = now.Add(-time.Hour) // beyond window
	e.lastHash = "same"

	out, _, err := e.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if out != OutcomeAbortedHashDup {
		t.Errorf("out = %q want %q", out, OutcomeAbortedHashDup)
	}
	if len(f.committedBlob) != 0 {
		t.Errorf("should not commit on hash dup")
	}
}

func TestEvaluateCommitted(t *testing.T) {
	now := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	f := &fakeDeps{now: now, statMtime: now, hash: "newhash"}
	e := newTestEval(f)
	e.lastHash = "oldhash"

	out, snap, err := e.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if out != OutcomeCommitted {
		t.Errorf("out = %q want %q", out, OutcomeCommitted)
	}
	if snap.ContentHash != "newhash" || snap.Tag != "auto" {
		t.Errorf("snap = %+v", snap)
	}
	if len(f.committedBlob) != 1 || f.committedBlob[0] != "newhash" {
		t.Errorf("committedBlob = %v", f.committedBlob)
	}
	if f.saved == nil || len(f.saved.Snapshots) != 1 {
		t.Fatalf("manifest not saved correctly: %+v", f.saved)
	}
	if e.lastHash != "newhash" {
		t.Errorf("lastHash = %q", e.lastHash)
	}
}

func TestEvaluateDroppedOnStatError(t *testing.T) {
	f := &fakeDeps{now: time.Now(), statErr: errors.New("locked")}
	e := newTestEval(f)

	out, _, err := e.Evaluate()
	if err == nil {
		t.Fatal("expected error")
	}
	if out != OutcomeDropped {
		t.Errorf("out = %q want %q", out, OutcomeDropped)
	}
}

func TestResumeRebaselines(t *testing.T) {
	mt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f := &fakeDeps{now: mt.Add(time.Hour), statMtime: mt, hash: "restored"}
	e := newTestEval(f)
	e.Pause()

	_, hash, err := e.Resume()
	if err != nil {
		t.Fatal(err)
	}
	if hash != "restored" {
		t.Errorf("hash = %q", hash)
	}
	if e.IsPaused() {
		t.Error("should be resumed")
	}
	if e.lastHash != "restored" || !e.lastMtime.Equal(mt) {
		t.Errorf("baseline not set: hash=%q mtime=%v", e.lastHash, e.lastMtime)
	}
}
