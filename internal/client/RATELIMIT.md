# Rate-Limit Handling

The Spaceship API enforces per-endpoint rate limits and returns `429 Too Many Requests` with a `Retry-After` header (seconds).

## Two strategies

### 1. Shared retry (`doJSONWithRetry`)

Used by all mutating and list endpoints (DNS CRUD, auto-renew, nameservers, domain list).

```
request  --->  429?  --->  parse Retry-After
                |                   |
                no                  v
                |           rateLimiter.activate(duration)
                v                   |
             success         wait (shared across goroutines)
                                    |
                                    v
                                  retry
```

Key behaviors:

- **Shared wait**: A single `rateLimiter` per `Client` coordinates all concurrent goroutines. When any goroutine gets a 429, the others see the active wait via `peek()` and hold off without making their own requests. All resume together when the timer fires.
- **Context-aware**: Both the wait and the HTTP request respect `ctx.Done()`, so callers can control cancellation via context deadlines or `context.WithCancel`.
- **Retry-After fallback**: If the header is missing or unparseable, defaults to 60 seconds to avoid a busy-retry loop.

### 2. List fallback (`GetDomainInfo`)

The single-domain endpoint (`GET /domains/{name}`) has very low rate limits. On 429 it falls back to `GetDomainList` (`GET /domains`), which has ~60x higher limits, and searches for the requested domain in the response.

This path uses `doJSON` (no retry) for the initial request so it can inspect the 429 status code and branch. The fallback `GetDomainList` call itself uses `doJSONWithRetry`.

## Timeout budget

The retry loop is bounded by two mechanisms:

1. **Caller-provided context deadline** — if the caller passes a context with a deadline (e.g. `context.WithTimeout(ctx, 12*time.Minute)`), the loop respects it via `ctx.Done()`.
2. **Client-level `maxRetryWait`** (default 10 min) — if the caller's context has no deadline (e.g. `context.Background()`), the client wraps it with `maxRetryWait` as a safety net. Override via `WithMaxRetryWait()` option.

Caller-provided deadlines always take precedence. The client-level budget only applies when no deadline is set.

## Testing

- **`clock.go` / `clock_test.go`**: `Clock` interface with `FakeClock` for deterministic time control. `FakeClock.Advance()` fires pending timers; `WaitForWaiters()` synchronizes test goroutines.
- **`domains_acc_test.go`**: Client-level tests — single retry, concurrent rate-limit coordination, max retry wait budget.

## Files

| File             | Role                                                        |
| ---------------- | ----------------------------------------------------------- |
| `ratelimiter.go` | Shared `rateLimiter` type with `peek()` / `activate()`      |
| `request.go`     | `doJSON` (no retry) and `doJSONWithRetry` (with retry loop) |
| `clock.go`       | `Clock` interface, `RealClock` implementation               |
| `clock_test.go`  | `FakeClock` for testing                                     |
| `client.go`      | `maxRetryWait` field, `WithMaxRetryWait` option             |
| `domains.go`     | `GetDomainInfo` with list fallback                          |
