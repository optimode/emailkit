// Package dnscache provides a thread-safe, TTL-based cache for DNS MX lookups
// with singleflight deduplication for concurrent requests to the same domain.
package dnscache

import (
	"context"
	"net"
	"sync"
	"time"
)

// Cache is a thread-safe DNS MX lookup cache.
// Concurrent lookups for the same domain are deduplicated:
// only one actual DNS query is performed, and all waiters receive the result.
type Cache struct {
	mu            sync.Mutex
	entries       map[string]*entry
	cacheTTL      time.Duration
	lookupTimeout time.Duration
	// resolver is injectable for testing
	resolver interface {
		LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	}
}

type entry struct {
	records []*net.MX
	err     error
	expires time.Time
	done    chan struct{} // closed when lookup is complete
}

// New creates a DNS cache with the given lookup timeout and cache TTL.
func New(lookupTimeout, cacheTTL time.Duration) *Cache {
	return &Cache{
		entries:       make(map[string]*entry),
		cacheTTL:      cacheTTL,
		lookupTimeout: lookupTimeout,
		resolver:      &net.Resolver{},
	}
}

// NewWithResolver creates a DNS cache with a custom resolver (for testing).
func NewWithResolver(lookupTimeout, cacheTTL time.Duration, r interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}) *Cache {
	c := New(lookupTimeout, cacheTTL)
	c.resolver = r
	return c
}

// LookupMX returns MX records for the domain, using the cache when possible.
// Concurrent lookups for the same domain are deduplicated via singleflight.
func (c *Cache) LookupMX(domain string) ([]*net.MX, error) {
	c.mu.Lock()

	if e, ok := c.entries[domain]; ok {
		select {
		case <-e.done:
			// Completed entry - check if still valid
			if time.Now().Before(e.expires) {
				c.mu.Unlock()
				return copyMX(e.records), e.err
			}
			// Expired, fall through to refresh
		default:
			// Lookup in progress - wait for it
			c.mu.Unlock()
			<-e.done
			return copyMX(e.records), e.err
		}
	}

	// Start new lookup
	e := &entry{done: make(chan struct{})}
	c.entries[domain] = e
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), c.lookupTimeout)
	defer cancel()

	e.records, e.err = c.resolver.LookupMX(ctx, domain)
	e.expires = time.Now().Add(c.cacheTTL)
	close(e.done)

	return copyMX(e.records), e.err
}

// Len returns the number of entries in the cache (for diagnostics).
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// copyMX returns a deep copy of MX records to prevent callers from
// mutating cached data (e.g., via sort.Slice).
func copyMX(records []*net.MX) []*net.MX {
	if records == nil {
		return nil
	}
	out := make([]*net.MX, len(records))
	for i, r := range records {
		cp := *r
		out[i] = &cp
	}
	return out
}
