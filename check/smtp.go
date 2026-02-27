package check

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/optimode/emailkit/internal/dnscache"
	"github.com/optimode/emailkit/internal/parse"
	"github.com/optimode/emailkit/internal/smtppool"
	"github.com/optimode/emailkit/types"
)

// SMTPConfig is the SMTP checker configuration.
type SMTPConfig struct {
	HeloDomain     string
	MailFrom       string
	MaxMXHosts     int
}

// SMTPChecker performs SMTP RCPT TO probes to verify email existence.
// It uses a shared DNS cache for MX lookups and an SMTP connection pool
// for efficient connection reuse via the RSET command.
type SMTPChecker struct {
	cfg      SMTPConfig
	dnsCache *dnscache.Cache
	pool     *smtppool.Pool
}

// NewSMTPChecker creates an SMTP checker with a shared DNS cache and connection pool.
func NewSMTPChecker(cfg SMTPConfig, cache *dnscache.Cache, pool *smtppool.Pool) *SMTPChecker {
	return &SMTPChecker{
		cfg:      cfg,
		dnsCache: cache,
		pool:     pool,
	}
}

func (c *SMTPChecker) Check(ctx context.Context, email parse.Email) types.CheckResult {
	level := types.LevelSMTP

	if !email.Valid {
		return types.CheckResult{Level: level, Passed: false, Details: "skipped: invalid email"}
	}

	// Use cached MX lookup (shared with DNS checker)
	mxRecords, err := c.dnsCache.LookupMX(email.Domain)
	if err != nil || len(mxRecords) == 0 {
		detail := "no MX records found"
		if err != nil {
			detail = fmt.Sprintf("MX lookup failed: %v", err)
		}
		return types.CheckResult{
			Level:   level,
			Passed:  false,
			Details: detail,
		}
	}

	sort.Slice(mxRecords, func(i, j int) bool {
		return mxRecords[i].Pref < mxRecords[j].Pref
	})

	maxHosts := c.cfg.MaxMXHosts
	if maxHosts <= 0 || maxHosts > len(mxRecords) {
		maxHosts = len(mxRecords)
	}

	var lastErr error
	for i := 0; i < maxHosts; i++ {
		// Check context cancellation before each attempt
		select {
		case <-ctx.Done():
			return types.CheckResult{
				Level:   level,
				Passed:  false,
				Details: "context cancelled",
			}
		default:
		}

		mxHost := strings.TrimSuffix(mxRecords[i].Host, ".")

		code, msg, err := c.pool.CheckRCPT(mxHost, email.Raw)
		if err != nil {
			lastErr = err
			continue
		}

		if code >= 500 {
			return types.CheckResult{
				Level:    level,
				Passed:   false,
				Details:  fmt.Sprintf("RCPT rejected: %s", msg),
				MXHost:   mxHost,
				SMTPCode: code,
			}
		}
		if code >= 400 {
			lastErr = fmt.Errorf("temporary failure %d: %s", code, msg)
			continue
		}

		return types.CheckResult{
			Level:    level,
			Passed:   true,
			Details:  "RCPT TO accepted",
			MXHost:   mxHost,
			SMTPCode: code,
		}
	}

	return types.CheckResult{
		Level:   level,
		Passed:  false,
		Details: fmt.Sprintf("SMTP probe failed on all MX hosts: %v", lastErr),
	}
}
