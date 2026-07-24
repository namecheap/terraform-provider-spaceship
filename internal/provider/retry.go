package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/namecheap/go-spaceship-sdk/client"
)

const (
	// rateLimitWindow is the API's throttling window: a Retry-After can request
	// up to a full window's wait. The per-operation timeout defaults next to
	// each resource's schema are expressed as multiples of it — one window per
	// rate-limitable call, plus slack. See internal/docs/rate-limits.md.
	rateLimitWindow = 5 * time.Minute
	// defaultRetryWait applies when a 429 arrives without a Retry-After header.
	defaultRetryWait = 30 * time.Second
	// retryWaitMargin lands the follow-up just after the window resets rather
	// than on its edge.
	retryWaitMargin = 1 * time.Second
	// retryDeadlineHeadroom is the budget the retried call itself needs after
	// the wait. A wait that fits the deadline but leaves less than this would
	// sleep into a guaranteed "context deadline exceeded" instead of failing
	// fast with an actionable message.
	retryDeadlineHeadroom = 5 * time.Second
)

// Test seams: tests swap the sleep for an instant recorder that advances a
// fake clock, keeping the limiter's bucket arithmetic deterministic, and
// reset the limiter so buckets cannot leak between tests.
var (
	retrySleep   = sleepContext
	retryNow     = time.Now
	retryLimiter = newRateLimiter()
)

// rateLimiter shares rate-limit wait state across concurrent operations.
// Terraform runs resources in parallel (user-configurable via -parallelism),
// so when one request exhausts an API bucket every sibling goroutine would
// otherwise burn a request discovering the same thing and then wait on its
// own. Recording "bucket blocked until T" lets later callers join the wait
// before their first attempt.
//
// Keys are "operation|domain", matching the API's per-domain buckets: a
// throttled domain never pauses work on other domains. Distinct operations
// that share one API bucket under-coordinate, which is safe — the second
// caller's 429 simply joins it to the wait.
type rateLimiter struct {
	mu    sync.Mutex
	until map[string]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{until: make(map[string]time.Time)}
}

// block records that the bucket stays exhausted for wait; a later deadline
// wins so concurrent 429s never shorten an existing wait. Expired entries are
// swept here so buckets that are never queried again (e.g. after a fail-fast)
// don't accumulate for the process lifetime.
func (rl *rateLimiter) block(key string, wait time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := retryNow()
	for k, until := range rl.until {
		if !until.After(now) {
			delete(rl.until, k)
		}
	}
	until := now.Add(wait)
	if until.After(rl.until[key]) {
		rl.until[key] = until
	}
}

// remaining reports how long the bucket stays blocked; zero or negative means
// clear. Cleanup of expired entries stays in block(), the only place entries
// are added.
func (rl *rateLimiter) remaining(key string) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.until[key].Sub(retryNow())
}

// withRetry runs fn, retrying while it fails with HTTP 429. The wait honors
// the server-requested Retry-After, and is shared through retryLimiter so
// concurrent operations on the same bucket join one wait instead of each
// discovering the 429 themselves. The ctx deadline (set from the resource
// timeouts block) is the only retry budget: when the wait cannot fit before
// the deadline, withRetry fails immediately instead of sleeping into a
// guaranteed timeout. Only 429s are retried — the server rejects those before
// execution, so writes are safe to repeat.
//
// scope identifies the rate-limit bucket the call draws from: the domain for
// per-domain endpoints, or perUserBucket(c) for per-user endpoints.
func withRetry(ctx context.Context, opName, scope string, fn func() error) error {
	key := opName + "|" + scope

	// The first waitTurn joins a wait another goroutine may already have
	// started for this bucket; cause is nil until fn has returned a 429.
	var cause error
	for {
		if err := waitTurn(ctx, opName, key, cause); err != nil {
			return err
		}
		if cause = fn(); !client.IsRateLimitError(cause) {
			return cause
		}
		retryLimiter.block(key, retryWait(cause))
	}
}

// withRetryValue is withRetry for calls that return a value alongside the
// error, sparing call sites the declare-outside-assign-inside closure dance.
func withRetryValue[T any](ctx context.Context, opName, scope string, fn func() (T, error)) (T, error) {
	var result T
	err := withRetry(ctx, opName, scope, func() error {
		var fnErr error
		result, fnErr = fn()
		return fnErr
	})
	return result, err
}

// perUserBucket keys a limiter bucket for endpoints whose rate limit is per
// user rather than per domain. The client pointer stands in for the account:
// aliased providers (different accounts) get distinct clients, so they never
// wait on each other's throttling.
func perUserBucket(c *client.Client) string {
	return fmt.Sprintf("%p", c)
}

// waitTurn sleeps out any active wait on the bucket, failing fast when the
// wait (plus headroom for the retried call itself) cannot fit before the ctx
// deadline. It re-checks the bucket after waking: another goroutine's 429 may
// have re-blocked it during the sleep, and attempting then would burn a
// request the limiter already knows is doomed. cause is the 429 that
// triggered the wait when the caller has one; it is wrapped into the
// fail-fast error so errors.As still surfaces the API error.
func waitTurn(ctx context.Context, opName, key string, cause error) error {
	for {
		wait := retryLimiter.remaining(key)
		if wait <= 0 {
			return nil
		}

		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if wait+retryDeadlineHeadroom > remaining {
				msg := fmt.Sprintf(
					"%s: rate limited, and the requested wait of %s exceeds the %s left of the operation timeout — raise the resource timeouts block or retry later",
					opName, wait.Round(time.Second), max(remaining, 0).Round(time.Second),
				)
				if cause != nil {
					return fmt.Errorf("%s: %w", msg, cause)
				}
				return errors.New(msg)
			}
		}

		tflog.Warn(ctx, "rate limited by the Spaceship API, waiting before retry", map[string]any{
			"operation": opName,
			"wait":      wait.String(),
		})

		if err := retrySleep(ctx, wait); err != nil {
			return err
		}
	}
}

// retryWait converts a rate-limit error into the wait duration: the
// server-requested Retry-After when present, defaultRetryWait otherwise,
// plus retryWaitMargin either way.
func retryWait(err error) time.Duration {
	wait := defaultRetryWait
	var apiErr *client.SpaceshipApiError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		wait = apiErr.RetryAfter
	}
	return wait + retryWaitMargin
}

// operationContext bounds ctx by the operation's timeout — the retry loop's
// only budget. resolve is the timeouts.Value method for the operation (e.g.
// plan.Timeouts.Create); its diagnostics are appended to diags, and the caller
// must defer cancel and return early when diags has errors.
func operationContext(ctx context.Context, resolve func(context.Context, time.Duration) (time.Duration, diag.Diagnostics), fallback time.Duration, diags *diag.Diagnostics) (context.Context, context.CancelFunc) {
	timeout, timeoutDiags := resolve(ctx, fallback)
	diags.Append(timeoutDiags...)
	if diags.HasError() {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// sleepContext waits for d, aborting immediately with ctx.Err() when ctx is
// cancelled (the user pressed Ctrl-C) or its deadline passes.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
