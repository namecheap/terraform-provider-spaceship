package provider

import (
	"context"
	"sync"

	"golang.org/x/sync/singleflight"

	"terraform-provider-spaceship/internal/client"
)

// dnsRecordCache memoizes per-domain DNS record fetches for the lifetime of a
// provider process (i.e. a single Terraform command). The singular
// spaceship_dns_record resource calls Find once per managed record during a
// refresh; without this cache each call re-fetches and paginates the whole
// zone, so N records in one domain cost N full zone reads. The cache collapses
// that into one read per domain.
//
// Correctness rests on write-invalidation: any mutation of a domain's records
// must call Invalidate(domain) so a later Find re-fetches instead of serving
// stale data. The cache deliberately lives in the provider layer (not the
// client) so the client stays a cache-free, reusable API surface — which means
// the client cannot invalidate on its own, and callers own that responsibility.
type dnsRecordCache struct {
	client *client.Client

	// sf collapses a cold-start stampede. Terraform refreshes resources
	// concurrently (default parallelism 10), so the first wave of Find calls
	// for a domain would otherwise each launch a full fetch. singleflight runs
	// a single fetch per key; the rest wait on it and share the result.
	sf singleflight.Group

	mu      sync.Mutex
	entries map[string][]client.DNSRecord
}

func newDNSRecordCache(c *client.Client) *dnsRecordCache {
	return &dnsRecordCache{
		client:  c,
		entries: make(map[string][]client.DNSRecord),
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
	c.mu.Unlock()
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
	result, err, _ := c.sf.Do(domain, func() (any, error) {
		records, err := c.client.GetDNSRecords(ctx, domain)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.entries[domain] = records
		c.mu.Unlock()
		return records, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]client.DNSRecord), nil
}
