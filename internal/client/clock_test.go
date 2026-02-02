package client

import (
	"sync"
	"time"
)

type FakeClock struct {
	mu      sync.Mutex
	current time.Time
	waiters []waiter
}

type waiter struct {
	until time.Time
	ch    chan time.Time
}

func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{current: start}
}

func (f *FakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current
}

func (f *FakeClock) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch := make(chan time.Time, 1)
	target := f.current.Add(d)

	if d <= 0 {
		ch <- f.current
		return ch
	}

	f.waiters = append(f.waiters, waiter{until: target, ch: ch})
	return ch
}

func (f *FakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.current = f.current.Add(d)

	var remaining []waiter
	for _, w := range f.waiters {
		if !f.current.Before(w.until) {
			w.ch <- f.current
			close(w.ch)
		} else {
			remaining = append(remaining, w)
		}
	}
	f.waiters = remaining
}

func (f *FakeClock) WaitForWaiters(n int) {
	for {
		f.mu.Lock()
		count := len(f.waiters)
		f.mu.Unlock()

		if count >= n {
			return
		}
		time.Sleep(1 * time.Millisecond) // smal real sleep for sync
	}
}
