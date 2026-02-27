package emailkit

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/optimode/emailkit/check"
	"github.com/optimode/emailkit/internal/dnscache"
	"github.com/optimode/emailkit/internal/parse"
	"github.com/optimode/emailkit/internal/smtppool"
	"github.com/optimode/emailkit/types"
)

// checker is the internal interface for all validation levels.
// Every check/ package type implements this.
type checker interface {
	Check(ctx context.Context, email parse.Email) types.CheckResult
}

// Validator is the main fluent builder struct.
// Instantiate with the New() function.
// When using SMTP validation, call Close() when done to release pooled connections.
type Validator struct {
	checkers []checker
	err      error // configuration error, returned on Validate()
	dnsCache *dnscache.Cache
	smtpPool *smtppool.Pool
}

// New creates a new Validator. By default it only performs syntax checking.
// Syntax checking always runs and cannot be disabled, because a valid email
// address is a prerequisite for the other levels.
func New() *Validator {
	return &Validator{
		checkers: []checker{
			check.NewSyntaxChecker(),
		},
	}
}

// WithDNS adds MX lookup validation to the pipeline.
// Optionally overrides the default DNSOptions.
// MX lookup results are cached and shared with the SMTP checker.
func (v *Validator) WithDNS(opts ...DNSOptions) *Validator {
	o := defaultDNSOptions()
	if len(opts) > 0 {
		o = opts[0]
	}
	v.ensureDNSCache(o.Timeout)
	v.checkers = append(v.checkers, check.NewDNSCheckerWithLookup(
		check.DNSConfig{
			Timeout:     o.Timeout,
			FallbackToA: o.FallbackToA,
		},
		v.dnsCache.LookupMX,
	))
	return v
}

// WithDomain adds domain-level validation (disposable + typo).
func (v *Validator) WithDomain(opts ...DomainOptions) *Validator {
	o := defaultDomainOptions()
	if len(opts) > 0 {
		o = opts[0]
	}
	v.checkers = append(v.checkers, check.NewDomainChecker(check.DomainConfig{
		CheckDisposable: o.CheckDisposable,
		CheckTypos:      o.CheckTypos,
		TypoThreshold:   o.TypoThreshold,
	}))
	return v
}

// WithSMTP adds the SMTP RCPT TO probe to the pipeline.
// SMTPOptions.HeloDomain and MailFrom are required.
// Uses a connection pool for efficient bulk validation (connections reused via RSET).
// Call Close() when done to release pooled connections.
func (v *Validator) WithSMTP(opts SMTPOptions) *Validator {
	if opts.HeloDomain == "" || opts.MailFrom == "" {
		v.err = ErrInvalidSMTPOptions
		return v
	}
	// Apply defaults for unset values
	def := defaultSMTPOptions()
	if opts.ConnectTimeout == 0 {
		opts.ConnectTimeout = def.ConnectTimeout
	}
	if opts.CommandTimeout == 0 {
		opts.CommandTimeout = def.CommandTimeout
	}
	if opts.MaxMXHosts == 0 {
		opts.MaxMXHosts = def.MaxMXHosts
	}
	if opts.Port == "" {
		opts.Port = def.Port
	}
	if opts.MaxConnsPerHost == 0 {
		opts.MaxConnsPerHost = def.MaxConnsPerHost
	}

	// Ensure DNS cache exists (SMTP checker shares it for MX lookups)
	v.ensureDNSCache(5 * opts.ConnectTimeout)

	// Create SMTP connection pool
	v.smtpPool = smtppool.New(smtppool.Config{
		HeloDomain:      opts.HeloDomain,
		MailFrom:        opts.MailFrom,
		ConnectTimeout:  opts.ConnectTimeout,
		CommandTimeout:  opts.CommandTimeout,
		Port:            opts.Port,
		MaxConnsPerHost: opts.MaxConnsPerHost,
	})

	v.checkers = append(v.checkers, check.NewSMTPChecker(
		check.SMTPConfig{
			HeloDomain: opts.HeloDomain,
			MailFrom:   opts.MailFrom,
			MaxMXHosts: opts.MaxMXHosts,
		},
		v.dnsCache,
		v.smtpPool,
	))
	return v
}

// Close releases resources held by the Validator.
// Must be called when using SMTP validation to close pooled connections.
// Safe to call multiple times. No-op if no pooled resources exist.
func (v *Validator) Close() error {
	if v.smtpPool != nil {
		return v.smtpPool.Close()
	}
	return nil
}

// ensureDNSCache creates a shared DNS cache if one doesn't exist yet.
func (v *Validator) ensureDNSCache(lookupTimeout time.Duration) {
	if v.dnsCache == nil {
		v.dnsCache = dnscache.New(lookupTimeout, 5*time.Minute)
	}
}

// Validate runs all configured checks on the given email.
// The pipeline short-circuits: if a level fails, subsequent levels are skipped.
// Context can be used for timeout or cancellation.
func (v *Validator) Validate(ctx context.Context, email string) (Result, error) {
	if v.err != nil {
		return Result{}, v.err
	}

	parsed := parse.NewEmail(email)
	result := Result{Email: email}

	for _, c := range v.checkers {
		cr := c.Check(ctx, parsed)
		result.Checks = append(result.Checks, cr)

		if !cr.Passed {
			result.Valid = false
			return result, nil // short-circuit
		}
	}

	result.Valid = true
	return result, nil
}

// ValidateAll runs all checks without short-circuiting.
// Useful when you want to know exactly which levels fail.
func (v *Validator) ValidateAll(ctx context.Context, email string) (Result, error) {
	if v.err != nil {
		return Result{}, v.err
	}

	parsed := parse.NewEmail(email)
	result := Result{Email: email, Valid: true}

	for _, c := range v.checkers {
		cr := c.Check(ctx, parsed)
		result.Checks = append(result.Checks, cr)
		if !cr.Passed {
			result.Valid = false
			// don't stop, continue
		}
	}

	return result, nil
}

// ConcurrencyOptions configures concurrent processing for ValidateMany.
type ConcurrencyOptions struct {
	// Workers is the number of concurrent goroutines. Default: 5
	Workers int
}

// ValidateMany validates multiple emails concurrently.
// The result order matches the input slice order.
// Emails are sorted by domain internally for optimal DNS cache and
// SMTP connection pool utilization.
func (v *Validator) ValidateMany(ctx context.Context, emails []string, opts ...ConcurrencyOptions) ([]Result, error) {
	if v.err != nil {
		return nil, v.err
	}

	workers := 5
	if len(opts) > 0 && opts[0].Workers > 0 {
		workers = opts[0].Workers
	}

	results := make([]Result, len(emails))
	type job struct {
		idx    int
		email  string
		domain string
	}

	// Build and sort jobs by domain for cache/pool locality
	jobSlice := make([]job, len(emails))
	for i, e := range emails {
		domain := ""
		if atIdx := strings.LastIndex(e, "@"); atIdx >= 0 {
			domain = strings.ToLower(e[atIdx+1:])
		}
		jobSlice[i] = job{idx: i, email: e, domain: domain}
	}
	sort.Slice(jobSlice, func(i, j int) bool {
		return jobSlice[i].domain < jobSlice[j].domain
	})

	// Feed sorted jobs into bounded channel
	bufSize := len(emails)
	if bufSize > 1000 {
		bufSize = 1000
	}
	jobs := make(chan job, bufSize)
	go func() {
		for _, j := range jobSlice {
			jobs <- j
		}
		close(jobs)
	}()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				res, err := v.Validate(ctx, j.email)
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("validating %q: %w", j.email, err)
					}
					mu.Unlock()
					continue
				}
				results[j.idx] = res
			}
		}()
	}

	wg.Wait()
	return results, firstErr
}
