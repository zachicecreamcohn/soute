package watcher

import (
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsAfterFailures(t *testing.T) {
	var slept []time.Duration
	d := Doer{Sleep: func(x time.Duration) { slept = append(slept, x) }}
	calls := 0
	err := d.Do(func() error {
		calls++
		if calls < 3 {
			return errors.New("fail")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d want 3", calls)
	}
	if len(slept) != 2 {
		t.Errorf("slept %d times, want 2", len(slept))
	}
}

func TestRetryExhausts(t *testing.T) {
	d := Doer{Sleep: func(time.Duration) {}}
	calls := 0
	err := d.Do(func() error { calls++; return errors.New("fail") })
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 4 {
		t.Errorf("calls = %d want 4 (1 initial + 3 retries)", calls)
	}
}

func TestRetryDefaultDelays(t *testing.T) {
	want := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	if len(DefaultRetryDelays) != len(want) {
		t.Fatalf("len = %d want %d", len(DefaultRetryDelays), len(want))
	}
	for i := range want {
		if DefaultRetryDelays[i] != want[i] {
			t.Errorf("delays[%d] = %v want %v", i, DefaultRetryDelays[i], want[i])
		}
	}
}
