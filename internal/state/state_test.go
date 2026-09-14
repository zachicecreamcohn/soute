package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	s := DaemonState{
		PID:       1234,
		IPCPath:   "/x/soute.sock",
		Status:    StatusRunning,
		StartedAt: time.Now().UTC(),
	}
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != s.PID || got.IPCPath != s.IPCPath || got.Status != s.Status {
		t.Errorf("mismatch: %+v", got)
	}
	if got.StartedAt.UnixNano() != s.StartedAt.UnixNano() {
		t.Errorf("StartedAt = %v want %v", got.StartedAt, s.StartedAt)
	}
}

func TestRemoveIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	Remove(path) // no error when missing

	if err := (DaemonState{PID: 1, Status: StatusRunning}).Save(path); err != nil {
		t.Fatal(err)
	}
	Remove(path)
	if _, err := Load(path); err == nil {
		t.Error("expected missing after remove")
	}
	Remove(path) // idempotent second remove
}
