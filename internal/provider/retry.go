package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/namecheap/go-spaceship-sdk/client"
)

const (
	// defaultRetryWait applies when a 429 arrives without a Retry-After header.
	defaultRetryWait = 30 * time.Second
	// retryWaitMargin lands the follow-up just after the window resets rather
	// than on its edge.
	retryWaitMargin = 1 * time.Second
)

// retrySleep is a package variable so tests can replace the wait with an
// instant recorder.
var retrySleep = sleepContext

// withRetry runs fn, retrying while it fails with HTTP 429 and sleeping the
// server-requested Retry-After between attempts. The ctx deadline (set from
// the resource timeouts block) is the only retry budget: when the requested
// wait cannot fit before the deadline, withRetry fails immediately instead of
// sleeping into a guaranteed timeout. Only 429s are retried — the server
// rejects those before execution, so writes are safe to repeat.
func withRetry(ctx context.Context, opName string, fn func() error) error {
	for {
		err := fn()
		if !client.IsRateLimitError(err) {
			return err
		}

		wait := retryWait(err)

		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if wait > remaining {
				return fmt.Errorf(
					"%s: rate limited, and the requested wait of %s exceeds the %s left of the operation timeout — raise the resource timeouts block or retry later: %w",
					opName, wait, remaining.Round(time.Second), err,
				)
			}
		}

		tflog.Warn(ctx, "rate limited by the Spaceship API, waiting before retry", map[string]any{
			"operation": opName,
			"wait":      wait.String(),
		})

		if sleepErr := retrySleep(ctx, wait); sleepErr != nil {
			return sleepErr
		}
	}
}

// retryWait converts a rate-limit error into the sleep duration: the
// server-requested Retry-After when present, defaultRetryWait otherwise,
// plus retryWaitMargin either way.
func retryWait(err error) time.Duration {
	var apiErr *client.SpaceshipApiError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter <= 0 {
		return defaultRetryWait + retryWaitMargin
	}
	return apiErr.RetryAfter + retryWaitMargin
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
