// Package engine implements the snapshot evaluation pipeline: threshold and
// delta math, hash deduplication, object storage, and garbage collection.
package engine

import (
	"math"
	"os"
	"time"

	"soute/internal/config"
	"soute/internal/manifest"
	"soute/internal/store"
)

// EvalResult reports the outcome of an evaluation pass.
type EvalResult int

const (
	// EvalSkipped means no snapshot was warranted (below threshold, unchanged
	// hash, or the file could not be read).
	EvalSkipped EvalResult = iota
	// EvalCaptured means a new snapshot was written.
	EvalCaptured
	// EvalCooldown means a snapshot was suppressed by the cooldown window;
	// the caller should re-evaluate once it lapses.
	EvalCooldown
)

// retryDelays is the exponential backoff for transient lock/stat failures.
var retryDelays = []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

// Engine snapshots the configured target into the .snapshots store.
type Engine struct {
	SnapDir string
}

// New returns an Engine operating on the given snapshots directory.
func New(snapDir string) *Engine {
	return &Engine{SnapDir: snapDir}
}

// Evaluate decides whether the target changed enough to snapshot and, if so,
// performs the capture.
func (e *Engine) Evaluate(now time.Time) (EvalResult, *manifest.Snapshot, error) {
	cfg, err := config.Load(e.SnapDir)
	if err != nil {
		return EvalSkipped, nil, err
	}
	m, err := manifest.Load(e.SnapDir)
	if err != nil {
		return EvalSkipped, nil, err
	}
	target := cfg.TargetAbs(e.SnapDir)

	size, err := statSize(target)
	if err != nil {
		return EvalSkipped, nil, err
	}

	last, hasLast := m.Last()
	if hasLast {
		if lastTime, perr := time.Parse(time.RFC3339, last.Timestamp); perr == nil &&
			now.Sub(lastTime) < time.Duration(cfg.CooldownSeconds)*time.Second {
			return EvalCooldown, nil, nil
		}

		diff := abs64(size - last.SizeBytes)
		trigger := diff >= cfg.MinDeltaBytes
		if !trigger && last.SizeBytes > 0 {
			trigger = float64(diff)/float64(last.SizeBytes)*100 >= cfg.MinDeltaPct
		}
		if !trigger {
			return EvalSkipped, nil, nil
		}
	}

	hash, err := hashFile(target)
	if err != nil {
		return EvalSkipped, nil, err
	}
	if hasLast && hash == last.ContentHash {
		return EvalSkipped, nil, nil
	}

	return e.capture(m, cfg, target, size, hash, now, manifest.TagAuto)
}

// Capture records the current target state unconditionally with the given tag.
func (e *Engine) Capture(tag string) (*manifest.Snapshot, error) {
	cfg, err := config.Load(e.SnapDir)
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(e.SnapDir)
	if err != nil {
		return nil, err
	}
	target := cfg.TargetAbs(e.SnapDir)

	size, err := statSize(target)
	if err != nil {
		return nil, err
	}
	hash, err := hashFile(target)
	if err != nil {
		return nil, err
	}

	_, snap, err := e.capture(m, cfg, target, size, hash, time.Now(), tag)
	return snap, err
}

func (e *Engine) capture(m *manifest.Manifest, cfg *config.Config, target string, size int64, hash string, now time.Time, tag string) (EvalResult, *manifest.Snapshot, error) {
	if err := store.Put(e.SnapDir, hash, target); err != nil {
		return EvalSkipped, nil, err
	}

	snap := manifest.Snapshot{
		ID:          m.NextID(now),
		Timestamp:   now.UTC().Format(time.RFC3339),
		ContentHash: hash,
		SizeBytes:   size,
		Tag:         tag,
	}
	if last, ok := m.Last(); ok {
		snap.DeltaBytes = size - last.SizeBytes
		if last.SizeBytes > 0 {
			snap.DeltaPct = round2(float64(snap.DeltaBytes) / float64(last.SizeBytes) * 100)
		}
	}
	m.Append(snap)

	if cfg.MaxSnapshots > 0 {
		if removed := m.Trim(cfg.MaxSnapshots); len(removed) > 0 {
			e.collect(removed, m.Hashes())
		}
	}

	if err := m.Save(e.SnapDir); err != nil {
		return EvalSkipped, nil, err
	}
	return EvalCaptured, &snap, nil
}

// Prune trims history to the newest keep entries and removes orphaned blobs,
// returning the number of snapshots removed.
func (e *Engine) Prune(keep int) (int, error) {
	m, err := manifest.Load(e.SnapDir)
	if err != nil {
		return 0, err
	}
	removed := m.Trim(keep)
	if len(removed) == 0 {
		return 0, nil
	}
	e.collect(removed, m.Hashes())
	if err := m.Save(e.SnapDir); err != nil {
		return 0, err
	}
	return len(removed), nil
}

// PruneAll keeps only the single most recent snapshot.
func (e *Engine) PruneAll() (int, error) {
	return e.Prune(1)
}

func (e *Engine) collect(removed []manifest.Snapshot, referenced map[string]bool) {
	for _, s := range removed {
		if !referenced[s.ContentHash] {
			store.Delete(e.SnapDir, s.ContentHash)
		}
	}
}

func statSize(path string) (int64, error) {
	var size int64
	err := withRetry(func() error {
		st, err := os.Stat(path)
		if err != nil {
			return err
		}
		size = st.Size()
		return nil
	})
	return size, err
}

func hashFile(path string) (string, error) {
	var hash string
	err := withRetry(func() error {
		var err error
		hash, err = store.HashFile(path)
		return err
	})
	return hash, err
}

// withRetry runs fn, retrying transient failures at 100/200/400ms.
func withRetry(fn func() error) error {
	var err error
	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt < len(retryDelays) {
			time.Sleep(retryDelays[attempt])
		}
	}
	return err
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
