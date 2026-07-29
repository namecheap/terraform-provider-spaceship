package provider

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/namecheap/go-spaceship-sdk/client"
)

// dnsRecordCacheFetchTimeout bounds the detached singleflight fetch. The
// fetch is a plain paginated zone read with no retries, so a couple of
// minutes is generous; without a cap an abandoned fetch (every waiter gave
// up) would run unbounded, detached from any caller's deadline.
const dnsRecordCacheFetchTimeout = 2 * time.Minute

// dnsRecordCache memoizes per-domain DNS record fetches for the lifetime of a
// provider process (i.e. a single Terraform command). The singular
// spaceship_dns_record resource calls Find once per managed record during a
// refresh; without this cache each call re-fetches and paginates the whole
// zone, so N records in one domain cost N full zone reads. The cache collapses
// that into one read per domain.
//
// Correctness rests on write-invalidation: every resource that reads through
// the cache must call Invalidate(domain) after mutating that domain's records,
// so a later Find re-fetches instead of serving stale data. The plural
// spaceship_dns_records resource deliberately does not participate (neither
// reading nor invalidating): mixing it with the singular resource on one
// domain is documented as unsupported, so the cache does not defend against
// that configuration. The cache lives in the provider layer (not the client)
// so the client stays a cache-free, reusable API surface — which means the
// client cannot invalidate on its own, and callers own that responsibility.
type dnsRecordCache struct {
	client *client.Client

	// sf collapses a cold-start stampede. Terraform refreshes resources
	// concurrently (default parallelism 10), so the first wave of Find calls
	// for a domain would otherwise each launch a full fetch. singleflight runs
	// a single fetch per key; the rest wait on it and share the result.
	sf singleflight.Group

	mu      sync.Mutex
	entries map[string][]client.DNSRecord
	// gen counts invalidations per domain. A fetch snapshots the generation
	// when it starts and only stores its result if the generation is unchanged
	// when it finishes — a detached fetch that raced a write can therefore
	// never re-cache pre-write data after the write's Invalidate.
	gen map[string]uint64
}

func newDNSRecordCache(c *client.Client) *dnsRecordCache {
	return &dnsRecordCache{
		client:  c,
		entries: make(map[string][]client.DNSRecord),
		gen:     make(map[string]uint64),
	}
}

// Find returns the custom-group record matching the API identity (type, name,
// signature) for the domain, serving from cache when warm. It returns
// client.ErrRecordNotFound when no record matches — same contract as
// client.FindDNSRecord, so callers can swap one for the other.
func (c *dnsRecordCache) Find(ctx context.Context, domain, recordType, name, signature string) (client.DNSRecord, error) {
	records, err := c.records(ctx, domain)
	if err != nil {
		return client.DNSRecord{}, err
	}
	if record, ok := client.MatchDNSRecord(records, recordType, name, signature); ok {
		return record, nil
	}
	return client.DNSRecord{}, client.ErrRecordNotFound
}

// Invalidate drops a domain's cached records so the next Find re-fetches. Call
// it after every successful write (create/update/delete) to that domain.
func (c *dnsRecordCache) Invalidate(domain string) {
	c.mu.Lock()
	delete(c.entries, domain)
	c.gen[domain]++
	c.mu.Unlock()
	// Forget any in-flight fetch so callers arriving after the write start a
	// fresh flight instead of joining one that snapshotted pre-write data.
	c.sf.Forget(domain)
}

// records returns the domain's custom-group records, serving the cached slice
// on a hit and fetching via client.GetDNSRecords on a miss.
func (c *dnsRecordCache) records(ctx context.Context, domain string) ([]client.DNSRecord, error) {
	c.mu.Lock()
	if records, ok := c.entries[domain]; ok {
		c.mu.Unlock()
		return records, nil
	}
	c.mu.Unlock()

	// Cache miss: fetch once across concurrent callers. singleflight collapses
	// the cold-start stampede so a whole refresh wave shares one fetch. The mutex
	// is held only around the map, never across the network call. A caller that
	// arrives just after a flight completes (and the key is forgotten) simply
	// runs one extra fetch — never stale, just an occasional redundant read.
	//
	// DoChan + select lets each caller observe its own context cancellation,
	// and context.WithoutCancel detaches the shared fetch from any single
	// caller's ctx so one caller's cancellation can't fail waiters that still
	// need the result; the fetch keeps its own deadline so it cannot outlive
	// every waiter indefinitely.
	ch := c.sf.DoChan(domain, func() (any, error) {
		c.mu.Lock()
		startGen := c.gen[domain]
		c.mu.Unlock()

		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dnsRecordCacheFetchTimeout)
		defer cancel()
		records, err := c.client.GetDNSRecords(fetchCtx, domain)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		if c.gen[domain] == startGen {
			c.entries[domain] = records
		}
		c.mu.Unlock()
		return records, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.([]client.DNSRecord), nil
	}
}
