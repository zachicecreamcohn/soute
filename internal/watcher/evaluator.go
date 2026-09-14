package watcher

import (
	"io/fs"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zacharycohn/soute/internal/config"
	"github.com/zacharycohn/soute/internal/hashing"
	"github.com/zacharycohn/soute/internal/store"
)

// Outcome is the result of a single evaluation pass.
type Outcome string

const (
	OutcomeCommitted      Outcome = "committed"
	OutcomeSkippedPaused  Outcome = "skipped_paused"
	OutcomeAbortedMTime   Outcome = "aborted_mtime"
	OutcomeAbortedTime    Outcome = "aborted_time"
	OutcomeAbortedHashDup Outcome = "aborted_hash_dup"
	OutcomeDropped        Outcome = "dropped"
)

// Deps holds the evaluator's external dependencies, injectable for tests.
type Deps struct {
	Now          func() time.Time
	Stat         func(path string) (fs.FileInfo, error)
	HashFile     func(path string) (hashing.Result, error)
	LoadManifest func() (store.Manifest, error)
	SaveManifest func(store.Manifest) error
	CommitBlob   func(srcPath, dataDir, hash string) (bool, error)
	Retry        func(fn func() error) error
}

// Evaluator runs the deferred-hash evaluation pipeline (spec §5.3). Hashing is
// deferred until the mtime and elapsed-time gates pass.
type Evaluator struct {
	TargetPath string
	DataDir    string
	Config     config.Config
	Deps       Deps

	mu               sync.Mutex
	lastMtime        time.Time
	lastHash         string
	lastSnapshotTime time.Time
	paused           atomic.Bool
}

// NewEvaluator builds an Evaluator wired to the real filesystem, baselined from
// the newest existing snapshot if any.
func NewEvaluator(paths config.Paths, cfg config.Config) *Evaluator {
	e := &Evaluator{
		TargetPath: paths.TargetAbs,
		DataDir:    paths.DataDir,
		Config:     cfg,
		Deps: Deps{
			Now:      time.Now,
			Stat:     os.Stat,
			HashFile: hashing.File,
			LoadManifest: func() (store.Manifest, error) {
				return store.Load(paths.Manifest)
			},
			SaveManifest: func(m store.Manifest) error {
				return store.SaveAtomic(paths.Manifest, m)
			},
			CommitBlob: func(src, dir, hash string) (bool, error) {
				return store.CommitBlob(src, dir, hash)
			},
			Retry: (Doer{}).Do,
		},
	}
	if m, err := store.Load(paths.Manifest); err == nil {
		if s, ok := store.Newest(m); ok {
			e.lastHash = s.ContentHash
			e.lastMtime = s.Mtime
			e.lastSnapshotTime = s.Timestamp
		}
	}
	return e
}

// Pause suspends evaluation, returning whether it was already paused.
func (e *Evaluator) Pause() bool { return e.paused.Swap(true) }

// IsPaused reports whether evaluation is suspended.
func (e *Evaluator) IsPaused() bool { return e.paused.Load() }

// Resume re-baselines the target (hash once, no commit) and resumes
// evaluation. It returns the observed mtime and hash so the caller can verify.
func (e *Evaluator) Resume() (string, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	fi, err := e.stat()
	if err != nil {
		e.paused.Store(false)
		return "", "", err
	}
	res, err := e.hash()
	if err != nil {
		e.paused.Store(false)
		return "", "", err
	}
	e.lastMtime = fi.ModTime()
	e.lastHash = res.Hash
	e.lastSnapshotTime = e.now()
	e.paused.Store(false)
	return fi.ModTime().UTC().Format(time.RFC3339Nano), res.Hash, nil
}

// Evaluate runs the pipeline once.
func (e *Evaluator) Evaluate() (Outcome, store.Snapshot, error) {
	if e.paused.Load() {
		return OutcomeSkippedPaused, store.Snapshot{}, nil
	}
	now := e.now()

	fi, err := e.stat()
	if err != nil {
		return OutcomeDropped, store.Snapshot{}, err
	}

	e.mu.Lock()
	if fi.ModTime().Equal(e.lastMtime) {
		e.mu.Unlock()
		return OutcomeAbortedMTime, store.Snapshot{}, nil
	}
	within := e.withinTimeWindow(now)
	e.mu.Unlock()
	if within {
		return OutcomeAbortedTime, store.Snapshot{}, nil
	}

	res, err := e.hash()
	if err != nil {
		return OutcomeDropped, store.Snapshot{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if res.Hash == e.lastHash {
		return OutcomeAbortedHashDup, store.Snapshot{}, nil
	}

	if _, err := e.Deps.CommitBlob(e.TargetPath, e.DataDir, res.Hash); err != nil {
		return OutcomeDropped, store.Snapshot{}, err
	}

	snap := store.Snapshot{
		ID:          newSnapshotID(now),
		Timestamp:   now.UTC(),
		ContentHash: res.Hash,
		SizeBytes:   res.Size,
		Mtime:       fi.ModTime(),
		Tag:         "auto",
	}

	m, err := e.Deps.LoadManifest()
	if err != nil {
		return OutcomeDropped, store.Snapshot{}, err
	}
	m.Snapshots = append(m.Snapshots, snap)
	if err := e.Deps.SaveManifest(m); err != nil {
		return OutcomeDropped, store.Snapshot{}, err
	}

	e.lastMtime = fi.ModTime()
	e.lastHash = res.Hash
	e.lastSnapshotTime = now

	// Automated GC when the manifest exceeds max_snapshots.
	if e.Config.MaxSnapshots > 0 && len(m.Snapshots) > e.Config.MaxSnapshots {
		if _, gerr := store.Prune(&m, e.Config.MaxSnapshots, e.DataDir); gerr == nil {
			_ = e.Deps.SaveManifest(m)
		}
	}

	return OutcomeCommitted, snap, nil
}

func (e *Evaluator) stat() (fs.FileInfo, error) {
	var fi fs.FileInfo
	err := e.retryFn()(func() error {
		var err error
		fi, err = e.Deps.Stat(e.TargetPath)
		return err
	})
	return fi, err
}

func (e *Evaluator) hash() (hashing.Result, error) {
	var res hashing.Result
	err := e.retryFn()(func() error {
		var err error
		res, err = e.Deps.HashFile(e.TargetPath)
		return err
	})
	return res, err
}

func (e *Evaluator) retryFn() func(func() error) error {
	if e.Deps.Retry != nil {
		return e.Deps.Retry
	}
	return (Doer{}).Do
}

func (e *Evaluator) now() time.Time {
	if e.Deps.Now != nil {
		return e.Deps.Now()
	}
	return time.Now()
}

func (e *Evaluator) withinTimeWindow(now time.Time) bool {
	if e.Config.MaxTimeSeconds == 0 {
		return false // 0 disables the elapsed-time gate
	}
	return now.Sub(e.lastSnapshotTime) < time.Duration(e.Config.MaxTimeSeconds)*time.Second
}

func newSnapshotID(t time.Time) string {
	return "snap_" + strconv.FormatInt(t.UnixNano(), 10)
}
