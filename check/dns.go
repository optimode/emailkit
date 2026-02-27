package check

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/optimode/emailkit/internal/parse"
	"github.com/optimode/emailkit/types"
)

// DNSConfig is the DNS checker configuration.
type DNSConfig struct {
	Timeout     time.Duration
	FallbackToA bool
}

// DNSChecker verifies the existence of MX records.
type DNSChecker struct {
	cfg    DNSConfig
	lookup func(domain string) ([]*net.MX, error) // injectable for testability
}

func NewDNSChecker(cfg DNSConfig) *DNSChecker {
	return &DNSChecker{
		cfg: cfg,
		lookup: func(domain string) ([]*net.MX, error) {
			ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
			defer cancel()
			r := &net.Resolver{}
			return r.LookupMX(ctx, domain)
		},
	}
}

// NewDNSCheckerWithLookup is a test-oriented constructor that overrides the MX lookup function.
func NewDNSCheckerWithLookup(cfg DNSConfig, fn func(string) ([]*net.MX, error)) *DNSChecker {
	c := NewDNSChecker(cfg)
	c.lookup = fn
	return c
}

func (c *DNSChecker) Check(ctx context.Context, email parse.Email) types.CheckResult {
	level := types.LevelDNS

	if !email.Valid {
		return types.CheckResult{Level: level, Passed: false, Details: "skipped: invalid email"}
	}

	mxRecords, err := c.lookup(email.Domain)
	if err != nil {
		// If FallbackToA is enabled, try A record
		if c.cfg.FallbackToA {
			addrs, aErr := net.LookupHost(email.Domain)
			if aErr == nil && len(addrs) > 0 {
				return types.CheckResult{
					Level:   level,
					Passed:  true,
					Details: "no MX record, but A record found (fallback)",
					MXHost:  addrs[0],
				}
			}
		}
		return types.CheckResult{
			Level:   level,
			Passed:  false,
			Details: fmt.Sprintf("MX lookup failed: %v", err),
		}
	}

	if len(mxRecords) == 0 {
		return types.CheckResult{Level: level, Passed: false, Details: "no MX records found"}
	}

	sort.Slice(mxRecords, func(i, j int) bool {
		return mxRecords[i].Pref < mxRecords[j].Pref
	})

	primaryMX := strings.TrimSuffix(mxRecords[0].Host, ".")
	return types.CheckResult{
		Level:   level,
		Passed:  true,
		Details: fmt.Sprintf("%d MX record(s) found", len(mxRecords)),
		MXHost:  primaryMX,
	}
}
