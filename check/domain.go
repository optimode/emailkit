package check

import (
	"context"
	"strings"

	"github.com/optimode/emailkit/internal/disposable"
	"github.com/optimode/emailkit/internal/levenshtein"
	"github.com/optimode/emailkit/internal/parse"
	"github.com/optimode/emailkit/types"
)

// DomainConfig is the domain checker configuration.
type DomainConfig struct {
	CheckDisposable bool
	CheckTypos      bool
	TypoThreshold   int
}

// DomainChecker detects disposable domains and typos.
type DomainChecker struct {
	cfg            DomainConfig
	knownProviders []string // known major email providers for typo detection
}

// defaultKnownProviders is the list of known major email providers.
// If the user's domain is within TypoThreshold distance from one of these,
// a warning is given (but the check does not fail).
var defaultKnownProviders = []string{
	"gmail.com", "googlemail.com",
	"yahoo.com", "yahoo.co.uk", "yahoo.fr", "yahoo.de",
	"outlook.com", "hotmail.com", "hotmail.co.uk", "live.com",
	"icloud.com", "me.com", "mac.com",
	"protonmail.com", "proton.me",
	"aol.com",
	"zoho.com",
	"yandex.com", "yandex.ru",
	"mail.com",
	"gmx.com", "gmx.net", "gmx.de",
	"fastmail.com",
	"tutanota.com",
	// Hungarian providers
	"freemail.hu", "citromail.hu", "t-online.hu", "invitel.hu",
}

func NewDomainChecker(cfg DomainConfig) *DomainChecker {
	return &DomainChecker{
		cfg:            cfg,
		knownProviders: defaultKnownProviders,
	}
}

func (c *DomainChecker) Check(_ context.Context, email parse.Email) types.CheckResult {
	level := types.LevelDomain

	if !email.Valid {
		return types.CheckResult{Level: level, Passed: false, Details: "skipped: invalid email"}
	}

	// Use ASCII/Punycode domain for disposable check (list is ASCII)
	asciiDomain := strings.ToLower(email.Domain)
	// Use Unicode domain for typo detection (better Levenshtein matching)
	unicodeDomain := strings.ToLower(email.DomainUnicode)

	// Disposable check
	if c.cfg.CheckDisposable {
		if disposable.IsDisposable(asciiDomain) {
			return types.CheckResult{
				Level:   level,
				Passed:  false,
				Details: "disposable email domain detected",
			}
		}
	}

	// Typo detection (warning only, does not fail)
	if c.cfg.CheckTypos {
		suggestion := c.findTypoSuggestion(unicodeDomain)
		if suggestion != "" {
			return types.CheckResult{
				Level:      level,
				Passed:     true, // typo suspicion does not fail
				Details:    "possible typo in domain",
				Suggestion: suggestion,
			}
		}
	}

	return types.CheckResult{Level: level, Passed: true, Details: "domain ok"}
}

// findTypoSuggestion finds the closest known provider.
// If the distance is <= TypoThreshold and the domain is not an exact match,
// it returns the suggested domain. Otherwise returns an empty string.
func (c *DomainChecker) findTypoSuggestion(domain string) string {
	bestDist := c.cfg.TypoThreshold + 1
	bestMatch := ""

	for _, provider := range c.knownProviders {
		if domain == provider {
			return "" // exact match, no typo
		}
		dist := levenshtein.Distance(domain, provider)
		if dist <= c.cfg.TypoThreshold && dist < bestDist {
			bestDist = dist
			bestMatch = provider
		}
	}

	return bestMatch
}
