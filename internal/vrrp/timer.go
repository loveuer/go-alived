package vrrp

import (
	"sync"
	"time"
)

// Timer provides a thread-safe timer with callback support.
type Timer struct {
	duration time.Duration
	timer    *time.Timer
	callback func()
	mu       sync.Mutex
}

// NewTimer creates a new Timer with the specified duration and callback.
func NewTimer(duration time.Duration, callback func()) *Timer {
	return &Timer{
		duration: duration,
		callback: callback,
	}
}

// Start starts or restarts the timer.
func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
	}

	t.timer = time.AfterFunc(t.duration, t.callback)
}

// Stop stops the timer if it's running.
func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
}

// Reset stops the current timer and starts a new one with the same duration.
func (t *Timer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
	}

	t.timer = time.AfterFunc(t.duration, t.callback)
}

// SetDuration updates the timer's duration for future starts.
func (t *Timer) SetDuration(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.duration = duration
}
