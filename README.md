# emailkit

[![CI](https://github.com/optimode/emailkit/actions/workflows/ci.yml/badge.svg)](https://github.com/optimode/emailkit/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/optimode/emailkit.svg)](https://pkg.go.dev/github.com/optimode/emailkit)
[![Go Report Card](https://goreportcard.com/badge/github.com/optimode/emailkit)](https://goreportcard.com/report/github.com/optimode/emailkit)
[![License](https://img.shields.io/github/license/optimode/emailkit)](LICENSE)

Modular Go email validation library with a fluent builder API.
Validate email addresses from basic syntax checks through DNS verification to SMTP mailbox probing — pick only the levels you need.

## Features

- **Fluent builder API** — compose your validation pipeline with `New().WithDNS().WithDomain().WithSMTP()`
- **RFC 5321/5322 syntax validation** with local part and domain checks
- **Internationalized Domain Names (IDN)** — automatic IDNA2008 Punycode conversion
- **Internationalized email local parts (EAI)** — RFC 6531 / SMTPUTF8 support
- **DNS validation** with MX record lookup and optional A record fallback
- **Disposable email detection** — built-in list of ~100 known throwaway domains
- **Domain typo detection** — Levenshtein distance matching against major providers
- **SMTP RCPT TO probe** with multi-MX host support
- **SMTP connection pool** — RSET-based connection reuse for bulk validation
- **DNS MX cache** — singleflight deduplication and configurable TTL
- **Bulk validation** — concurrent processing with domain-sorted ordering for optimal cache/pool locality
- **Context support** — timeout and cancellation on all network operations
- **Single runtime dependency** — `golang.org/x/net/idna` (Go official extended library)

## Requirements

- Go 1.25 or later

## Installation

```sh
go get github.com/optimode/emailkit
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/optimode/emailkit"
)

func main() {
    ctx := context.Background()

    result, err := emailkit.New().
        WithDNS().
        WithDomain().
        WithSMTP(emailkit.SMTPOptions{
            HeloDomain: "myapp.com",
            MailFrom:   "verify@myapp.com",
        }).
        Validate(ctx, "user@example.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Valid)    // true or false
    for _, c := range result.Checks {
        fmt.Printf("  [%s] passed=%v  %s\n", c.Level, c.Passed, c.Details)
    }
}
```

## Usage

Every validation level is optional except syntax (which always runs as a prerequisite).
Add levels with the `With*` methods — order doesn't matter, but the pipeline executes them in registration order and **short-circuits on the first failure**.

### Syntax Validation

Validates email format according to RFC 5321/5322 with full internationalization support.
Catches malformed addresses, overlong parts, invalid characters, and structural errors before any network calls are made.

This level always runs — it's the foundation for every other check.

```go
result, err := emailkit.New().Validate(ctx, "user@example.com")
// result.Valid == true
// result.Checks[0].Details == "syntax ok"

result, _ = emailkit.New().Validate(ctx, "missing-at-sign")
// result.Valid == false
// result.Checks[0].Details == "invalid email syntax"

result, _ = emailkit.New().Validate(ctx, "user@münchen.de")
// result.Valid == true (IDN domain, converted to Punycode internally)

result, _ = emailkit.New().Validate(ctx, "用户@example.com")
// result.Valid == true (EAI / RFC 6531 Unicode local part)
```

### DNS Validation

Verifies that the email domain has valid MX records — confirming it can actually receive mail.
Prevents accepting addresses at domains with no mail infrastructure.

```go
v := emailkit.New().WithDNS()

result, _ := v.Validate(ctx, "user@example.com")
// result.Checks[1].MXHost == "mx.example.com" (primary MX host)

// With A record fallback for domains that use A records instead of MX:
v = emailkit.New().WithDNS(emailkit.DNSOptions{
    Timeout:     10 * time.Second, // default: 5s
    FallbackToA: true,             // default: false
})
```

### Domain Validation

Detects disposable (throwaway) email domains and typos in common provider names.
Useful for catching `user@gmial.com` or blocking `user@mailinator.com` at the form level.

Disposable detection fails the check. Typo detection **never fails** — it only populates the `Suggestion` field so your application can prompt the user ("Did you mean gmail.com?").

```go
v := emailkit.New().WithDomain()

// Disposable domain:
result, _ := v.Validate(ctx, "user@mailinator.com")
// result.Valid == false
// result.Checks[1].Details == "disposable email domain detected"

// Typo detection:
result, _ = v.Validate(ctx, "user@gmial.com")
// result.Valid == true (typo doesn't fail)
// result.Checks[1].Suggestion == "gmail.com"

// Configure sensitivity:
v = emailkit.New().WithDomain(emailkit.DomainOptions{
    CheckDisposable: true, // default: true
    CheckTypos:      true, // default: true
    TypoThreshold:   2,    // default: 2 (Levenshtein distance)
})
```

### SMTP Validation

Performs an SMTP RCPT TO probe against the domain's mail servers to check whether the mailbox actually exists.
This is the most thorough validation level — it catches addresses that look valid and have working DNS but where the mailbox doesn't exist.

Connections are pooled and reused via the SMTP RSET command, making bulk validation efficient.
**Always call `Close()` when done** to release pooled connections.

```go
v := emailkit.New().
    WithDNS().
    WithSMTP(emailkit.SMTPOptions{
        HeloDomain: "myapp.com",        // required: your domain
        MailFrom:   "verify@myapp.com", // required: envelope sender
    })
defer v.Close()

result, _ := v.Validate(ctx, "user@example.com")
// result.Checks[2].SMTPCode == 250 (accepted)
// result.Checks[2].MXHost == "mx.example.com"

// Full options:
v = emailkit.New().WithSMTP(emailkit.SMTPOptions{
    HeloDomain:      "myapp.com",
    MailFrom:        "verify@myapp.com",
    ConnectTimeout:  5 * time.Second,  // default: 5s
    CommandTimeout:  10 * time.Second, // default: 10s
    MaxMXHosts:      2,                // default: 2 (MX hosts to try)
    Port:            "25",             // default: "25"
    MaxConnsPerHost: 3,                // default: 3 (pooled connections per MX host)
})
defer v.Close()
```

### Non-Short-Circuit Validation

By default, `Validate()` stops at the first failing level. Use `ValidateAll()` when you need to know exactly which levels pass and which fail — useful for diagnostics or detailed user feedback.

```go
result, _ := v.ValidateAll(ctx, "user@nonexistent-domain.example")
// result.Valid == false
// result.Checks contains results for ALL configured levels, not just the first failure
for _, c := range result.FailedChecks() {
    fmt.Printf("failed: [%s] %s\n", c.Level, c.Details)
}
```

### Bulk Validation

`ValidateMany()` validates a slice of emails concurrently. Internally, emails are sorted by domain for optimal DNS cache and SMTP connection pool utilization. Result order always matches input order.

```go
v := emailkit.New().
    WithDNS().
    WithDomain().
    WithSMTP(emailkit.SMTPOptions{
        HeloDomain: "myapp.com",
        MailFrom:   "verify@myapp.com",
    })
defer v.Close()

emails := []string{
    "alice@example.com",
    "bob@gmail.com",
    "carol@example.com", // same domain as alice — processed together
}

results, err := v.ValidateMany(ctx, emails, emailkit.ConcurrencyOptions{
    Workers: 10, // default: 5
})
// results[0] corresponds to alice, results[1] to bob, etc.
```

### Inspecting Results

The `Result` struct provides helpers for examining validation outcomes.

```go
result, _ := v.Validate(ctx, "user@example.com")

// Check overall validity:
result.Valid // true if all checks passed

// Get a specific level's result:
if dns, ok := result.CheckFor(emailkit.LevelDNS); ok {
    fmt.Println(dns.MXHost) // primary MX host
}

// Get all failures:
for _, c := range result.FailedChecks() {
    fmt.Printf("[%s] %s\n", c.Level, c.Details)
}

// JSON serialization (all fields have json tags):
data, _ := json.Marshal(result)
```

## Contributing

Contributions are welcome. Please follow these guidelines:

1. **Fork and branch** — create a feature branch from `main`
2. **Code style** — run `make check` (vet + lint + test) before submitting
3. **Tests** — add tests for new functionality; aim to maintain or improve coverage (`make cover`)
4. **Commits** — use [Conventional Commits](https://www.conventionalcommits.org/) format (e.g. `feat: add rate limiting`, `fix: handle nil MX response`)
5. **One concern per PR** — keep pull requests focused on a single change

### Development

```sh
make check      # run vet + lint + tests
make test-race  # run tests with race detector
make cover      # show test coverage report
make tidy       # tidy and verify module dependencies
```

## License

This project is licensed under the [BSD 3-Clause License](LICENSE).
