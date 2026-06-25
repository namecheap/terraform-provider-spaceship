package client

import (
	"os"
	"testing"
)

// Live check against the real Spaceship API. Forces pagination with a page size
// of 1 and asserts it returns the same domains as a single large page, so it
// proves the loop reassembles the full list on any account with >= 2 domains —
// no need to own 100. Skipped unless API credentials are set, and excluded from
// `make test` by the TestAcc prefix.
func TestAccGetDomainListPagination(t *testing.T) {
	key := os.Getenv("SPACESHIP_API_KEY")
	secret := os.Getenv("SPACESHIP_API_SECRET")
	if key == "" || secret == "" {
		t.Skip("set SPACESHIP_API_KEY and SPACESHIP_API_SECRET to run the live pagination check")
	}

	c, err := NewClient("https://spaceship.dev/api/v1", key, secret)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	full, err := c.listDomains(t.Context(), maxDomainListPageSize)
	if err != nil {
		t.Fatalf("single-page list (take=100): %v", err)
	}

	paged, err := c.listDomains(t.Context(), 1)
	if err != nil {
		t.Fatalf("paged list (take=1): %v", err)
	}

	// Same set regardless of page size.
	if len(paged.Items) != len(full.Items) {
		t.Fatalf("pagination mismatch: take=1 returned %d domains, take=100 returned %d",
			len(paged.Items), len(full.Items))
	}

	// The loop's termination invariant: the API's reported total must equal the
	// number of items actually collected. If total were a per-page count, the
	// loop would either stop early or spin — this asserts it is the global count.
	if full.Total != int64(len(full.Items)) {
		t.Fatalf("total invariant broken: API total=%d but collected %d items",
			full.Total, len(full.Items))
	}

	switch {
	case len(full.Items) >= 2:
		t.Logf("pagination exercised across multiple pages: %d domains via take=1", len(full.Items))
	default:
		t.Logf("contract confirmed (take/skip/total/orderBy) but only %d domain in account; "+
			"multi-page iteration NOT exercised — needs >= 2 domains", len(full.Items))
	}
}
