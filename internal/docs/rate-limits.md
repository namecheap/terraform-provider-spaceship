# Rate Limits & Retry — Design Notes

## Context

Several Spaceship API endpoints are aggressively rate limited per domain (as of
July 2026; server-side limits can change without notice):

| Endpoint | Limit |
|---|---|
| Domain info (GET) | 5 req / domain / 300s |
| Nameserver update (PUT) | 5 req / domain / 300s |
| Personal nameservers (GET) | 5 req / domain / 300s |
| Domain list (GET) | 300 req / user / 300s |
| DNS records list (GET) | 300 req / user / 300s |

An ordinary plan → apply → re-plan cycle issues several domain-info reads and
nameserver writes within minutes, so the 5/300s buckets exhaust in normal use.
On HTTP 429 the API sends a `Retry-After` header: the number of **seconds** the
client ought to wait before a follow-up request (delta-seconds only, never an
HTTP date; observed values run up to ~300 — a full window).

A 429 means the server rejected the request *before executing it*, so retrying
is safe for writes as well as reads — there is no idempotency concern (unlike a
timeout, where the write may have landed).

## Layering: SDK reports facts, provider owns policy

- **SDK** (`go-spaceship-sdk`): `errorFromResponse` parses the `Retry-After`
  header into `SpaceshipApiError.RetryAfter` (`time.Duration`; zero when absent
  or unparsable). `IsRateLimitError(err)` mirrors the existing
  `IsNotFoundError` idiom. The SDK never sleeps — a general-purpose client that
  silently blocks for minutes would ambush non-Terraform consumers.
- **Provider**: `withRetry` (`internal/provider/retry.go`) implements the wait
  policy — sleeping, logging, deadline accounting, cancellation. Terraform UX
  decisions belong in the Terraform layer.

The SDK's existing `GetDomainInfo` 429 fallback (retry via the higher-limit
domain-list endpoint) is unchanged and runs *inside* the wrapped call; provider
retry fires only when both paths are exhausted. Do not defeat the fallback.

## Retry policy

`withRetry(ctx, opName, fn)` loops until the call stops returning 429:

```
for {
    err := fn()
    if err is not a 429 rate-limit error → return err
    wait := err.RetryAfter (30s when the header is missing) + 1s margin
    if wait > time remaining before ctx deadline
        → fail immediately: report the server-requested wait and the
          operation timeout so the user knows which knob to turn
    tflog.Warn(ctx, "rate limited", op, wait, remaining)
    sleep wait, aborting instantly if ctx is cancelled
}
```

Deliberate choices:

- **The ctx deadline is the only budget.** There is no max-attempts counter —
  two limits that can disagree produce "failed after N retries with minutes
  still on the clock". Same principle as `retry.RetryContext` in
  terraform-plugin-sdk.
- **Fail fast on an unfittable wait.** Sleeping into a deadline that cannot be
  met converts a bounded failure into "hang, then fail anyway".
- **Only HTTP 429 is retried.** Other errors (4xx, 5xx, transport) return
  unchanged; retrying them here would mask real failures.

## Operation timeouts

`spaceship_domain` exposes a `timeouts` block
(`terraform-plugin-framework-timeouts`):

| Operation | Default | Rationale |
|---|---|---|
| `create` | 15m | up to 3 rate-limitable calls × 300s worst case |
| `update` | 15m | same call shape as create |
| `read` | 5m | single domain-info read, one window + margin |

Delete is a state-only no-op (no API call), so it takes no timeout. Each CRUD
method resolves its timeout, wraps the request context with
`context.WithTimeout`, and passes it down; the retry loop inherits the deadline.

## Cancellation (Ctrl-C)

Terraform cancels the operation context on interrupt. Every wait in the retry
loop must select on `ctx.Done()` as well as the sleep timer, so a user pressing
Ctrl-C during a 280s rate-limit wait gets an immediate, clean abort — never an
orphaned sleep that holds the apply hostage. `ctx.Err()` is returned so
Terraform reports the operation as cancelled, not failed.

## Visibility

- `tflog.Warn` on every wait (operation, wait duration, remaining budget) —
  visible with `TF_LOG=WARN` and in HCP Terraform run logs.
- During apply, Terraform core natively prints `Still modifying... [Xs elapsed]`
  every ~10s, so long waits do not look frozen.
- During plan/refresh there is no native heartbeat (a Terraform core
  limitation); the `read` timeout keeps the quiet period bounded.

## Testing

Two-layer split, as everywhere in this codebase:

- **SDK repo**: unit tests for `Retry-After` parsing — present, absent,
  garbage, large values (httptest).
- **Provider unit tests** (`retry_test.go`): 429-then-success retries and
  returns the success; non-429 errors pass through without retry; missing
  `RetryAfter` uses the 30s default; a wait larger than the remaining deadline
  fails immediately (no sleep); ctx cancellation mid-sleep aborts promptly and
  returns `ctx.Err()`. The sleep is injected so tests never actually wait.
- **Provider unit tests** (domain resource): `timeouts` attribute present in
  schema; defaults resolve when the block is omitted.
- **Acceptance tests**: the existing `spaceship_domain` acc tests exercise the
  wrapped paths end-to-end; deliberately provoking a real 429 is not worth the
  5-minute lockout per run.

## Registry docs

Per the registry docs style rules: the `timeouts` block is documented via
`make docs`; wording never mentions rate-limit numbers ("operations retry
transient API throttling until the operation timeout elapses"). Rate-limit
specifics live only in this internal note.

## Rollout

`spaceship_domain` first. `withRetry` is resource-agnostic; wiring it into
`spaceship_personal_nameserver` (also 5/300s) and the data sources is
follow-up work, one resource per change.
