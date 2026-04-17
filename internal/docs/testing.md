# Testing Conventions

## Layer Responsibilities

### Client-layer unit tests (`internal/client/`, `internal/client/records/`)

These test the HTTP client and record validation logic. They are the **single source of truth** for:

- API request/response serialization (JSON payloads, query parameters)
- HTTP error handling (status codes, error mapping, retry/fallback logic)
- Pagination
- Record validation (field formats, boundary values, required fields)
- DNS record filtering (custom vs product vs personalNS groups)

Use `httptest.Server` to mock the Spaceship API at the HTTP level.

**Examples of good client-layer tests:**
- `TestGetDNSRecords_FiltersOutNonCustomGroups` — verifies group filtering logic
- `TestARecord_ValidateAddress` — verifies IP format validation
- `TestSRVRecord_ValidateTarget_EdgeCases` — boundary testing at 253 chars
- `TestDeleteDNSRecords_NotFoundIgnored` — verifies 404 is swallowed

### Provider-layer unit tests (`internal/provider/`)

These test Terraform-specific behavior that **only exists at the provider layer**. They are the single source of truth for:

- Schema validation using Terraform types (`types.String`, `types.Object`, null/unknown handling)
- Diff and reconciliation logic (`diffDNSRecords`, `orderDNSRecordsLike`)
- State mapping (expand/flatten between Terraform models and client structs)
- Import lifecycle (`ImportState`)
- Provider configuration (credential resolution, missing credentials)
- Custom validators that use the Terraform validator framework (`ValidateObject`, `ValidateString`)

**Examples of good provider-layer tests:**
- `TestDiffDNSRecords_AddressChange` — tests reconciliation logic unique to the provider
- `TestNameserversValidator_ValidateObject` — tests validation with Terraform types (null, unknown, sets)
- `TestDomainResource_ImportState` — tests import lifecycle not covered elsewhere
- `TestExpandDNSRecords_AllTypes` — tests Terraform-to-client model conversion
- `TestConfigure_MissingAPIKey` — tests provider-level credential errors

### Acceptance tests (`*_acc_test.go`)

These test full Terraform lifecycle (plan → apply → read → update → destroy → import) against the real Spaceship API. They are the authoritative tests for end-to-end correctness.

- Run only when explicitly requested (require API credentials)
- Prefixed with `TestAcc`
- Cover real API behavior including eventual consistency

## What NOT to test at the provider layer

Do not create provider-layer tests that merely re-test client-layer logic through a Terraform wrapper. Common anti-patterns:

- **Mock server to test `if err != nil`**: If a data source or resource Read method calls `client.GetSomething()` and the error path is just `resp.Diagnostics.AddError(...)`, the client-layer test for `GetSomething` already covers the error. Adding an httptest server + full Terraform provider setup to verify a 3-line error branch adds maintenance cost with no value.

- **Happy-path mock tests that duplicate acceptance tests**: If you're building a mock API server that handles GET/PUT/DELETE to test create-read-update-delete, you're reimplementing what acceptance tests already do — except with a fake API that can diverge from reality.

- **Testing framework plumbing**: Don't test that `Configure()` fails when `ProviderData` is the wrong Go type (Terraform controls this), that `Schema()` returns non-nil attributes (compile-time guarantees), or that interface assertions hold (the provider factory enforces this).

- **Re-testing validation through wrappers**: If an `ObjectValidator` delegates to a client-layer `Validate()` method, test the validation logic at the client layer. Only add provider-layer validator tests when there's meaningful Terraform-type handling (null, unknown, type assertions) in the wrapper itself.

## Decision checklist for new tests

Before writing a test, ask:

1. **Is this testing something unique to this layer?** If the logic is in `internal/client/`, test it there.
2. **Does this test a real branching decision?** A 3-line `if err != nil` block doesn't need its own test.
3. **Is there already a test at another layer?** Check client tests and acceptance tests first.
4. **Would a bug here be caught by existing tests?** If yes, the new test is redundant.
5. **Does this require Terraform-specific types or lifecycle?** If yes, it belongs at the provider layer.
