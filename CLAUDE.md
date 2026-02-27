# CLAUDE.md

Project-level instructions for Claude Code.

## Project

emailkit is a modular Go email validation library with a fluent builder API.
Module path: `github.com/optimode/emailkit`.

## Language

- Code, comments, documentation, commit messages: English
- Communication with the user: match the user's language

## Development

- Go 1.25+
- Single runtime dependency: `golang.org/x/net/idna`
- Test dependency: `github.com/stretchr/testify`
- Run `make check` before committing (vet + lint + test)
- Run `make test-race` to verify concurrency safety

## Git Workflow

Type: GitHub Flow with RC/Stable Tags.

- **Single production branch**: `main` — always production-ready
- **No develop branch** — unnecessary for small teams
- **Short-lived feature branches** — only when needed, merge fast, minimize drift
- **RC tag first, stable tag later** — every version goes through testing before stable release
- **Explicit tags** — RC and stable versions are clearly separated (e.g. `v0.1.0-rc.1` → `v0.1.0`)

## Versioning

Semantic Versioning 2.0.0 (`MAJOR.MINOR.PATCH`).

## Commit Messages

Conventional Commits format. English language. No signatures, names, emails, or attribution in the commit message footer. Footer is reserved for `BREAKING CHANGE:` and issue/ticket references only.

```
<type>[optional scope]: <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`.

## Changelog

Every completed task must result in a CHANGELOG.md update following the [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) format.

- Update the `[Unreleased]` section during development
- Move entries to a versioned section when creating a stable tag
- Include the release date in ISO 8601 format (YYYY-MM-DD)
- Group changes by type: Added, Changed, Deprecated, Removed, Fixed, Security
- Entries must be concise, single-line, and user-focused
- Link version numbers to tags/releases

## Testing Conventions

- Black-box tests: use `package <name>_test` suffix (not `package <name>`)
- Assertions: use `github.com/stretchr/testify/assert`
- Testability via dependency injection: pass mock functions/interfaces instead of using real network
  - DNS: injectable `lookup func(string) ([]*net.MX, error)` via `NewDNSCheckerWithLookup()`
  - SMTP: injectable `Dial func(network, address string, timeout time.Duration) (net.Conn, error)` via `smtppool.Config`
  - DNS cache: injectable resolver via `dnscache.NewWithResolver()`
- Use `net.Pipe()` to simulate SMTP servers in tests (no real network needed)
- Testable Example functions (`Example*` in `_test.go`) for every exported API — these appear on pkg.go.dev and are verified by `go test`
- Handle all error return values in tests (`_ = closer.Close()`) to satisfy errcheck

## Architecture

- **`types/` package**: exists solely to break circular imports between the root `emailkit` package and the `check/` package — both need `CheckResult` and `CheckLevel`
- **`internal/` packages**: implementation details not exposed to consumers — `parse`, `dnscache`, `smtppool`, `disposable`, `levenshtein`
- **Shared resources**: the `Validator` creates a single `dnscache.Cache` and `smtppool.Pool`, shared across checkers via `ensureDNSCache()` — the DNS checker and SMTP checker reuse the same cached MX lookups
- **Dependency injection**: all network operations are injectable for testing — no checker directly calls `net.Dial` or `net.Resolver`
- **Checker interface**: every validation level implements `Check(ctx, parse.Email) types.CheckResult` — the `Validator` iterates over them in registration order
- **IDN/EAI dual representation**: `parse.Email` carries both `Domain` (ASCII/Punycode for DNS/SMTP) and `DomainUnicode` (for display/typo detection)

## Project Structure

```
emailkit.go          # package doc, re-exports from types/
validator.go         # Validator builder and pipeline
options.go           # DNSOptions, DomainOptions, SMTPOptions
result.go            # Result type with helpers
errors.go            # sentinel errors
types/               # shared types (avoids circular imports)
check/               # validation levels (syntax, dns, domain, smtp)
internal/parse/      # email parser with IDN/EAI support
internal/dnscache/   # MX lookup cache with singleflight
internal/smtppool/   # SMTP connection pool with RSET reuse
internal/disposable/ # embedded disposable domain list
internal/levenshtein/ # edit distance for typo detection
_examples/           # standalone runnable examples
```
