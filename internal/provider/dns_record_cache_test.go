package provider

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"terraform-provider-spaceship/internal/client"
)

// newCountingRecordCache returns a cache backed by a mock API plus a counter of
// how many record-list fetches reached the server. For record sets under one
// page, one GetDNSRecords call == one GET, so the counter measures cache misses.
func newCountingRecordCache(t *testing.T, items []map[string]any) (*dnsRecordCache, *int64) {
	t.Helper()

	var gets int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		atomic.AddInt64(&gets, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "total": len(items)})
	}))
	t.Cleanup(server.Close)

	c, err := client.NewClient(server.URL, "k", "s")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return newDNSRecordCache(c), &gets
}

// A warm cache serves repeated Find calls for the same domain from a single fetch.
func TestDNSRecordCache_CachesRepeatedReads(t *testing.T) {
	cache, gets := newCountingRecordCache(t, []map[string]any{
		{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
		{"type": "A", "name": "www", "ttl": 3600, "address": "5.6.7.8"},
	})

	for range 5 {
		rec, err := cache.Find(t.Context(), "example.com", "A", "@", "1.2.3.4")
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		if rec.Address != "1.2.3.4" {
			t.Fatalf("expected 1.2.3.4, got %q", rec.Address)
		}
	}

	if got := atomic.LoadInt64(gets); got != 1 {
		t.Fatalf("expected 1 underlying fetch, got %d", got)
	}
}

// Invalidate forces the next Find to re-fetch.
func TestDNSRecordCache_InvalidateForcesRefetch(t *testing.T) {
	cache, gets := newCountingRecordCache(t, []map[string]any{
		{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
	})

	if _, err := cache.Find(t.Context(), "example.com", "A", "@", "1.2.3.4"); err != nil {
		t.Fatalf("first Find: %v", err)
	}
	cache.Invalidate("example.com")
	if _, err := cache.Find(t.Context(), "example.com", "A", "@", "1.2.3.4"); err != nil {
		t.Fatalf("second Find: %v", err)
	}

	if got := atomic.LoadInt64(gets); got != 2 {
		t.Fatalf("expected 2 fetches after invalidation, got %d", got)
	}
}

// A missing record returns client.ErrRecordNotFound, not an error.
func TestDNSRecordCache_FindMissingReturnsNotFound(t *testing.T) {
	cache, _ := newCountingRecordCache(t, []map[string]any{
		{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
	})

	_, err := cache.Find(t.Context(), "example.com", "A", "missing", "9.9.9.9")
	if !errors.Is(err, client.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

// A failed fetch is not cached — the next Find retries and succeeds.
func TestDNSRecordCache_ErrorNotCached(t *testing.T) {
	var gets int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt64(&gets, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"}},
			"total": 1,
		})
	}))
	t.Cleanup(server.Close)

	c, err := client.NewClient(server.URL, "k", "s")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cache := newDNSRecordCache(c)

	if _, err := cache.Find(t.Context(), "example.com", "A", "@", "1.2.3.4"); err == nil {
		t.Fatal("expected first Find to fail")
	}
	if _, err := cache.Find(t.Context(), "example.com", "A", "@", "1.2.3.4"); err != nil {
		t.Fatalf("second Find: %v", err)
	}
}

// Invalidate evicts only the given domain; other domains stay cached.
func TestDNSRecordCache_InvalidateIsPerDomain(t *testing.T) {
	cache, gets := newCountingRecordCache(t, []map[string]any{
		{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
	})

	for _, domain := range []string{"a.com", "b.com"} {
		if _, err := cache.Find(t.Context(), domain, "A", "@", "1.2.3.4"); err != nil {
			t.Fatalf("warm Find %s: %v", domain, err)
		}
	}
	cache.Invalidate("a.com")
	for _, domain := range []string{"a.com", "b.com"} {
		if _, err := cache.Find(t.Context(), domain, "A", "@", "1.2.3.4"); err != nil {
			t.Fatalf("post-invalidate Find %s: %v", domain, err)
		}
	}

	// 2 warm-up fetches + 1 re-fetch for a.com; b.com stays cached.
	if got := atomic.LoadInt64(gets); got != 3 {
		t.Fatalf("expected 3 fetches, got %d", got)
	}
}

// Concurrent cold-cache Find calls collapse into a single fetch (singleflight).
func TestDNSRecordCache_ConcurrentReadsCollapse(t *testing.T) {
	cache, gets := newCountingRecordCache(t, []map[string]any{
		{"type": "A", "name": "@", "ttl": 3600, "address": "1.2.3.4"},
	})

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cache.Find(t.Context(), "example.com", "A", "@", "1.2.3.4")
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt64(gets); got != 1 {
		t.Fatalf("expected concurrent reads to collapse into 1 fetch, got %d", got)
	}
}
