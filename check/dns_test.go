package check_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit/check"
	"github.com/optimode/emailkit/internal/parse"
)

func TestDNSChecker_WithMockLookup(t *testing.T) {
	tests := []struct {
		name    string
		records []*net.MX
		lookErr error
		wantOK  bool
	}{
		{
			name:    "has MX records",
			records: []*net.MX{{Host: "mx.example.com.", Pref: 10}},
			wantOK:  true,
		},
		{
			name:    "no MX records",
			records: []*net.MX{},
			wantOK:  false,
		},
		{
			name:    "lookup error",
			lookErr: &net.DNSError{Err: "no such host"},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := check.DNSConfig{Timeout: 2 * time.Second}
			c := check.NewDNSCheckerWithLookup(cfg, func(domain string) ([]*net.MX, error) {
				return tt.records, tt.lookErr
			})
			parsed := parse.NewEmail("test@example.com")
			result := c.Check(context.Background(), parsed)
			assert.Equal(t, tt.wantOK, result.Passed)
		})
	}
}

func TestDNSChecker_MXHostTrimsDot(t *testing.T) {
	cfg := check.DNSConfig{Timeout: 2 * time.Second}
	c := check.NewDNSCheckerWithLookup(cfg, func(domain string) ([]*net.MX, error) {
		return []*net.MX{{Host: "mx.example.com.", Pref: 10}}, nil
	})
	parsed := parse.NewEmail("test@example.com")
	result := c.Check(context.Background(), parsed)
	assert.True(t, result.Passed)
	assert.Equal(t, "mx.example.com", result.MXHost)
}

func TestDNSChecker_SortsByPreference(t *testing.T) {
	cfg := check.DNSConfig{Timeout: 2 * time.Second}
	c := check.NewDNSCheckerWithLookup(cfg, func(domain string) ([]*net.MX, error) {
		return []*net.MX{
			{Host: "mx2.example.com.", Pref: 20},
			{Host: "mx1.example.com.", Pref: 10},
		}, nil
	})
	parsed := parse.NewEmail("test@example.com")
	result := c.Check(context.Background(), parsed)
	assert.True(t, result.Passed)
	assert.Equal(t, "mx1.example.com", result.MXHost)
}

func TestDNSChecker_InvalidEmail(t *testing.T) {
	cfg := check.DNSConfig{Timeout: 2 * time.Second}
	c := check.NewDNSCheckerWithLookup(cfg, func(domain string) ([]*net.MX, error) {
		return nil, nil
	})
	parsed := parse.NewEmail("invalid")
	result := c.Check(context.Background(), parsed)
	assert.False(t, result.Passed)
	assert.Contains(t, result.Details, "skipped")
}
