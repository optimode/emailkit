# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Fluent builder API for composable email validation pipeline
- RFC 5321/5322 syntax validation with local part and domain checks
- Internationalized Domain Names (IDNA2008) with automatic Punycode conversion
- Internationalized email local parts (RFC 6531 / SMTPUTF8 / EAI)
- DNS validation with MX record lookup and optional A record fallback
- Domain validation with disposable email detection (~100 domains)
- Domain typo detection using Levenshtein distance with configurable threshold
- SMTP RCPT TO probe with multi-MX host support
- SMTP connection pool with RSET-based reuse for bulk validation
- DNS MX lookup cache with singleflight deduplication and configurable TTL
- Domain-sorted processing in `ValidateMany()` for optimal cache/pool locality
- `Validator.Close()` for releasing pooled SMTP connections
- `ValidateAll()` for non-short-circuit validation
- `ValidateMany()` for concurrent bulk email validation with domain sorting
- Context support for timeout and cancellation on all network operations
- Testable example functions for `pkg.go.dev` documentation
- Makefile with `check`, `test`, `lint`, `cover`, and `tidy` targets
- GitHub Actions CI workflow (test matrix, lint, coverage)
- BSD 3-Clause license
- Single runtime dependency: `golang.org/x/net/idna` (Go official extended library)
