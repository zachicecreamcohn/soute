package watcher

import "time"

// Debouncer collapses a burst of triggers into a single signal after a quiet
// interval. The timer factory is injectable for deterministic tests.
type Debouncer struct {
	interval time.Duration
	newTimer func(time.Duration) <-chan time.Time

	trigger chan struct{} // request to (re)arm the timer
	acked   chan struct{} // timer armed (makes Trigger synchronous)
	out     chan struct{} // the "quiet interval elapsed" signal
	done    chan struct{}
}

// NewDebounce returns a Debouncer that fires on C after interval of no
// triggers. nt is the timer factory; nil uses time.After.
func NewDebounce(interval time.Duration, nt func(time.Duration) <-chan time.Time) *Debouncer {
	if nt == nil {
		nt = time.After
	}
	d := &Debouncer{
		interval: interval,
		newTimer: nt,
		trigger:  make(chan struct{}),
		acked:    make(chan struct{}),
		out:      make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	go d.loop()
	return d
}

func (d *Debouncer) loop() {
	var c <-chan time.Time
	for {
		select {
		case <-d.done:
			return
		case <-d.trigger:
			c = d.newTimer(d.interval)
			d.acked <- struct{}{}
		case <-c:
			c = nil
			select {
			case d.out <- struct{}{}:
			default:
			}
		}
	}
}

// Trigger records an event, resetting the quiet window. It blocks until the
// timer has been re-armed.
func (d *Debouncer) Trigger() {
	d.trigger <- struct{}{}
	<-d.acked
}

// C returns the channel that fires once after the quiet interval.
func (d *Debouncer) C() <-chan struct{} { return d.out }

// Stop terminates the debouncer goroutine.
func (d *Debouncer) Stop() { close(d.done) }
