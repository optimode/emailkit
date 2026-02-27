package emailkit

import "time"

// DNSOptions configures the DNS validation level.
type DNSOptions struct {
	// Timeout is the maximum time for MX lookup. Default: 5s
	Timeout time.Duration
	// FallbackToA when true accepts A records when no MX record is found.
	// Default: false (strict MX requirement)
	FallbackToA bool
}

func defaultDNSOptions() DNSOptions {
	return DNSOptions{
		Timeout:     5 * time.Second,
		FallbackToA: false,
	}
}

// DomainOptions configures the domain-level validation.
type DomainOptions struct {
	// CheckDisposable when true fails on known disposable domains. Default: true
	CheckDisposable bool
	// CheckTypos when true suggests corrections for close-match domains. Default: true
	// This never fails an email, only provides a suggestion (Suggestion field).
	CheckTypos bool
	// TypoThreshold is the Levenshtein distance threshold for typo detection. Default: 2
	TypoThreshold int
}

func defaultDomainOptions() DomainOptions {
	return DomainOptions{
		CheckDisposable: true,
		CheckTypos:      true,
		TypoThreshold:   2,
	}
}

// SMTPOptions configures the SMTP probe level.
type SMTPOptions struct {
	// HeloDomain is the domain sent in the EHLO command. Required, e.g. "myapp.com"
	HeloDomain string
	// MailFrom is the address sent in the MAIL FROM command. Required, e.g. "verify@myapp.com"
	MailFrom string
	// ConnectTimeout is the maximum time for TCP connection. Default: 5s
	ConnectTimeout time.Duration
	// CommandTimeout is the maximum response time for SMTP commands. Default: 10s
	CommandTimeout time.Duration
	// MaxMXHosts is how many MX hosts to try sequentially. Default: 2
	MaxMXHosts int
	// Port is the SMTP port. Default: 25
	Port string
	// MaxConnsPerHost is the max pooled SMTP connections per MX host. Default: 3
	MaxConnsPerHost int
}

func defaultSMTPOptions() SMTPOptions {
	return SMTPOptions{
		ConnectTimeout:  5 * time.Second,
		CommandTimeout:  10 * time.Second,
		MaxMXHosts:      2,
		Port:            "25",
		MaxConnsPerHost: 3,
	}
}
