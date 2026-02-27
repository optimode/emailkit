package dnscache_test

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit/internal/dnscache"
)

// mockResolver tracks how many times LookupMX was called.
type mockResolver struct {
	records []*net.MX
	err     error
	calls   atomic.Int64
}

func (m *mockResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	m.calls.Add(1)
	return m.records, m.err
}

func TestCache_BasicCaching(t *testing.T) {
	r := &mockResolver{
		records: []*net.MX{{Host: "mx.example.com.", Pref: 10}},
	}
	c := dnscache.NewWithResolver(2*time.Second, 1*time.Minute, r)

	// First call: actual lookup
	recs, err := c.LookupMX("example.com")
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, int64(1), r.calls.Load())

	// Second call: cached
	recs, err = c.LookupMX("example.com")
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, int64(1), r.calls.Load()) // still 1, no new lookup
}

func TestCache_DifferentDomains(t *testing.T) {
	r := &mockResolver{
		records: []*net.MX{{Host: "mx.test.", Pref: 10}},
	}
	c := dnscache.NewWithResolver(2*time.Second, 1*time.Minute, r)

	_, _ = c.LookupMX("a.com")
	_, _ = c.LookupMX("b.com")
	assert.Equal(t, int64(2), r.calls.Load())
	assert.Equal(t, 2, c.Len())
}

func TestCache_TTLExpiry(t *testing.T) {
	r := &mockResolver{
		records: []*net.MX{{Host: "mx.test.", Pref: 10}},
	}
	c := dnscache.NewWithResolver(2*time.Second, 50*time.Millisecond, r) // short TTL

	_, _ = c.LookupMX("example.com")
	assert.Equal(t, int64(1), r.calls.Load())

	time.Sleep(100 * time.Millisecond) // wait for expiry

	_, _ = c.LookupMX("example.com")
	assert.Equal(t, int64(2), r.calls.Load()) // refreshed
}

func TestCache_Singleflight(t *testing.T) {
	r := &mockResolver{
		records: []*net.MX{{Host: "mx.test.", Pref: 10}},
	}
	c := dnscache.NewWithResolver(2*time.Second, 1*time.Minute, r)

	// Launch many concurrent lookups for the same domain
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs, err := c.LookupMX("example.com")
			assert.NoError(t, err)
			assert.Len(t, recs, 1)
		}()
	}
	wg.Wait()

	// Should have only performed 1 actual lookup
	assert.Equal(t, int64(1), r.calls.Load())
}

func TestCache_CachesErrors(t *testing.T) {
	r := &mockResolver{
		err: &net.DNSError{Err: "no such host"},
	}
	c := dnscache.NewWithResolver(2*time.Second, 1*time.Minute, r)

	_, err := c.LookupMX("bad.com")
	assert.Error(t, err)

	_, err = c.LookupMX("bad.com")
	assert.Error(t, err)
	assert.Equal(t, int64(1), r.calls.Load()) // error was cached
}

func TestCache_ReturnsCopy(t *testing.T) {
	r := &mockResolver{
		records: []*net.MX{
			{Host: "mx2.", Pref: 20},
			{Host: "mx1.", Pref: 10},
		},
	}
	c := dnscache.NewWithResolver(2*time.Second, 1*time.Minute, r)

	recs1, _ := c.LookupMX("example.com")
	recs2, _ := c.LookupMX("example.com")

	// Mutating one copy should not affect the other
	recs1[0].Host = "modified."
	assert.NotEqual(t, recs1[0].Host, recs2[0].Host)
}
