package watcher

import (
	"testing"
	"time"
)

func TestDebounceCollapsesBurst(t *testing.T) {
	timerCh := make(chan time.Time, 1)
	d := NewDebounce(time.Second, func(time.Duration) <-chan time.Time { return timerCh })
	defer d.Stop()

	for i := 0; i < 5; i++ {
		d.Trigger()
	}
	select {
	case <-d.C():
		t.Fatal("fired before quiet interval")
	default:
	}

	timerCh <- time.Now()
	select {
	case <-d.C():
	case <-time.After(time.Second):
		t.Fatal("did not fire after timer")
	}
}

func TestDebounceResets(t *testing.T) {
	var chans []chan time.Time
	d := NewDebounce(time.Second, func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		chans = append(chans, ch)
		return ch
	})
	defer d.Stop()

	d.Trigger() // arms chans[0]
	d.Trigger() // resets to chans[1]
	if len(chans) != 2 {
		t.Fatalf("expected 2 timers, got %d", len(chans))
	}

	// The stale timer must be ignored.
	chans[0] <- time.Now()
	select {
	case <-d.C():
		t.Fatal("stale timer should not fire")
	case <-time.After(20 * time.Millisecond):
	}

	// The current timer must fire.
	chans[1] <- time.Now()
	select {
	case <-d.C():
	case <-time.After(time.Second):
		t.Fatal("current timer did not fire")
	}
}
