package watcher

import "time"

// DefaultRetryDelays is the exponential backoff for lock retries (spec §5.2):
// 100ms, 200ms, 400ms (max 3 retries).
var DefaultRetryDelays = []time.Duration{
	100 * time.Millisecond,
	200 * time.Millisecond,
	400 * time.Millisecond,
}

// Doer retries a function with injectable sleep (for deterministic tests).
type Doer struct {
	Sleep func(time.Duration)
}

// Do retries fn using DefaultRetryDelays (1 initial attempt + 3 retries).
func (r Doer) Do(fn func() error) error {
	return r.DoWithDelays(DefaultRetryDelays, fn)
}

// DoWithDelays retries fn using the provided delays. The total attempts are
// len(delays)+1.
func (r Doer) DoWithDelays(delays []time.Duration, fn func() error) error {
	sleep := r.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	var err error
	for attempt := 0; ; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt >= len(delays) {
			return err
		}
		sleep(delays[attempt])
	}
}
