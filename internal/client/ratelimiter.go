package client

import (
	"fmt"
	"sync"
	"time"
)

// RateLimitTimeoutError is returned when the context deadline or cancellation
// fires while waiting for an API rate-limit window to clear. Callers can use
// errors.As to detect this specific condition and provide appropriate messaging.
type RateLimitTimeoutError struct {
	Cause error
}

func (e *RateLimitTimeoutError) Error() string {
	return fmt.Sprintf("timed out waiting for API rate limit to clear: %v", e.Cause)
}

func (e *RateLimitTimeoutError) Unwrap() error {
	return e.Cause
}

// rateLimiter coordinates a shared wait period across all concurrent goroutines
// when a 429 Too Many Requests response is received. Instead of each goroutine
// waiting independently, the first goroutine to hit a rate limit starts a single
// timer; all subsequent goroutines join the same wait channel.
type rateLimiter struct {
	mu     sync.Mutex
	waitCh chan struct{} // nil = not waiting; closed when wait is done (wakes all waiters)
	clock  Clock
}

// peek returns the active wait channel without blocking, or nil if no rate-limit
// wait is currently in progress.
func (rl *rateLimiter) peek() <-chan struct{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.waitCh
}

// activate starts a global wait of duration d if none is in progress, or joins
// the existing wait if one was already started by another goroutine.
// Returns a channel that is closed when the wait period is over.
func (rl *rateLimiter) activate(d time.Duration) <-chan struct{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.waitCh != nil {
		// Another goroutine already started the wait — join it.
		return rl.waitCh
	}

	ch := make(chan struct{})
	rl.waitCh = ch

	go func() {
		<-rl.clock.After(d)
		rl.mu.Lock()
		rl.waitCh = nil
		close(ch)
		rl.mu.Unlock()
	}()

	return ch
}
