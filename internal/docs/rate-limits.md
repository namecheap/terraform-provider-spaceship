# Rate Limits & Retry — Design Notes

## Context

Several Spaceship endpoints are aggressively rate limited (as of July 2026;
limits can change server-side): domain info, nameserver updates, and personal
nameservers are 5–10 requests per domain per 300s; domain list and DNS record
endpoints are 300 per user per 300s. On HTTP 429 the API sends `Retry-After`:
integer **seconds** to wait (never an HTTP date), observed up to ~300. A 429
is rejected *before execution*, so retrying is safe for writes as well as
reads.

## Layering

The SDK reports facts and never sleeps: `SpaceshipApiError.RetryAfter` (zero
when the header is absent or unparsable) and `IsRateLimitError`. All policy —
waiting, logging, deadline accounting, cancellation — lives in the provider's
`withRetry` (`internal/provider/retry.go`). The SDK's `GetDomainInfo`
list-fallback runs inside the wrapped call; provider retry fires only when
both paths are exhausted.

## Retry invariants

- Only HTTP 429 is retried; every other error returns unchanged.
- The wait is the server's `Retry-After` plus a 1s margin; 30s + margin when
  the header is missing.
- The ctx deadline (from the resource `timeouts` block) is the only budget —
  no attempt counters. A wait that cannot fit (including a few seconds of
  headroom for the retried call itself) fails immediately with the requested
  wait, the remaining time, and the knob to turn, wrapping the original API
  error. After every sleep the bucket is re-checked before attempting, so a
  429 recorded by a sibling goroutine mid-sleep extends the wait instead of
  burning a doomed request.
- Waits are shared: a limiter keyed `operation|domain` (matching the API's
  per-domain buckets) lets concurrent goroutines join one wait instead of each
  burning a request — correct at any `-parallelism`. A throttled domain never
  pauses other domains; operations sharing one API bucket under-coordinate,
  which self-heals.
- Every wait selects on `ctx.Done()`: Ctrl-C aborts mid-sleep and returns
  `ctx.Err()` so Terraform reports "cancelled", not "failed".
- Each wait is logged via `tflog.Warn`; Terraform's native "Still modifying…"
  heartbeat covers apply. Plan/refresh has no heartbeat (core limitation), so
  the read timeout bounds the quiet period.

## Operation timeouts

Every resource and both data sources expose a `timeouts` block
(`terraform-plugin-framework-timeouts`). Defaults = rate-limitable calls per
operation × one full 300s window, plus at least a minute of slack — the last
window's wait is Retry-After (≤300s) + 1s margin, and the deadline must also
fit the retried call: domain 16/6/16m (delete is a state-only no-op);
personal nameserver 10/6/10/6m; `dns_records` 21/6/21/11m (create/update make
four calls, clear makes two); `dns_record` 10/6/11/6m; data source reads 6m.
Each CRUD method resolves its timeout and wraps ctx via
`context.WithTimeout`; the singular `dns_record` retries around the shared
cache's `Find` (not inside its detached singleflight fetch) so waits stay
bounded by each caller's own deadline.

## Testing

Header parsing is tested in the SDK (mocked responses). The provider unit
tests own the policy: retry/join/independent-bucket behavior, fail-fast,
cancellation (`retry_test.go`, with an injected sleep that advances a fake
clock), plus an end-to-end mock-server test that a rate-limited read retries
after the exact requested wait. Do not try to provoke real 429s in acceptance
tests — whitelisted accounts never see them.

## Registry docs

Never mention rate-limit numbers in `docs/` or `templates/` — describe the
behavior ("throttled requests are retried until the operation timeout
elapses"). Specifics live only in this note.

## Future

If more SDK consumers need retries, the policy can move into the SDK as an
opt-in client option; `withRetry` is the single seam to swap. Reactive retry
only — add proactive client-side throttling only if real usage shows
sustained 429 storms.
