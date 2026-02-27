package check

import (
	"context"
	"strings"
	"unicode"

	"github.com/optimode/emailkit/internal/parse"
	"github.com/optimode/emailkit/types"
)

// SyntaxChecker validates email syntax according to RFC 5321/5322
// with RFC 6531 (SMTPUTF8) and IDNA2008 internationalization support.
type SyntaxChecker struct{}

func NewSyntaxChecker() *SyntaxChecker {
	return &SyntaxChecker{}
}

func (c *SyntaxChecker) Check(_ context.Context, email parse.Email) types.CheckResult {
	level := types.LevelSyntax

	if email.Raw == "" {
		return types.CheckResult{Level: level, Passed: false, Details: "empty email address"}
	}

	if !email.Valid {
		return types.CheckResult{Level: level, Passed: false, Details: "invalid email syntax"}
	}

	// Length checks (RFC 5321)
	if len(email.Raw) > 254 {
		return types.CheckResult{Level: level, Passed: false, Details: "email address exceeds 254 characters"}
	}
	if len(email.Local) > 64 {
		return types.CheckResult{Level: level, Passed: false, Details: "local part exceeds 64 characters"}
	}

	// Local part validation
	// net/mail.ParseAddress strips quotes from quoted local parts,
	// so we check the raw input to detect quoted form.
	quotedLocal := hasQuotedLocal(email.Raw)
	if !quotedLocal {
		if err := validateLocal(email.Local); err != "" {
			return types.CheckResult{Level: level, Passed: false, Details: err}
		}
	}

	// Domain validation (use Unicode form for user-friendly error messages;
	// IDNA2008 validation was already done during parsing)
	if err := validateDomain(email.DomainUnicode); err != "" {
		return types.CheckResult{Level: level, Passed: false, Details: err}
	}

	return types.CheckResult{Level: level, Passed: true, Details: "syntax ok"}
}

// hasQuotedLocal checks if the raw email has a quoted local part.
func hasQuotedLocal(raw string) bool {
	atIdx := strings.LastIndex(raw, "@")
	if atIdx < 1 {
		return false
	}
	local := raw[:atIdx]
	return strings.HasPrefix(local, `"`) && strings.HasSuffix(local, `"`)
}

// validateLocal validates the local part.
// Supports RFC 5321 ASCII characters and RFC 6531 (SMTPUTF8) Unicode characters.
// Returns error text, or "" if ok.
func validateLocal(local string) string {
	if local == "" {
		return "local part is empty"
	}

	// Quoted local part: "something"
	if strings.HasPrefix(local, `"`) && strings.HasSuffix(local, `"`) {
		return "" // in quoted form all printable characters are allowed
	}

	// RFC 5321 ASCII special characters (besides alphanumeric)
	asciiSpecial := "!#$%&'*+/=?^_`{|}~-."

	for _, ch := range local {
		if ch > 127 {
			// RFC 6531 (SMTPUTF8): non-ASCII Unicode characters are allowed,
			// except control characters
			if unicode.IsControl(ch) {
				return "local part contains control character"
			}
			continue
		}
		// ASCII range: letters, digits, and RFC 5321 special characters
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			continue
		}
		if !strings.ContainsRune(asciiSpecial, ch) {
			return "local part contains invalid character: " + string(ch)
		}
	}

	// Cannot start or end with a dot
	if strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") {
		return "local part cannot start or end with a dot"
	}

	// Cannot contain consecutive dots
	if strings.Contains(local, "..") {
		return "local part cannot contain consecutive dots"
	}

	return ""
}

// validateDomain validates the domain part (Unicode form).
// Returns error text, or "" if ok.
func validateDomain(domain string) string {
	if domain == "" {
		return "domain is empty"
	}

	// IP literal: [127.0.0.1] - accept but don't validate deeply
	if strings.HasPrefix(domain, "[") && strings.HasSuffix(domain, "]") {
		return ""
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return "domain must have at least two labels"
	}

	for _, label := range labels {
		if label == "" {
			return "domain contains empty label (consecutive dots)"
		}
		if len(label) > 63 {
			return "domain label exceeds 63 characters"
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "domain label cannot start or end with a hyphen"
		}
		for _, ch := range label {
			if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' {
				return "domain label contains invalid character: " + string(ch)
			}
		}
	}

	// TLD cannot be all digits
	tld := labels[len(labels)-1]
	allDigits := true
	for _, ch := range tld {
		if !unicode.IsDigit(ch) {
			allDigits = false
			break
		}
	}
	if allDigits {
		return "TLD cannot be all digits"
	}

	return ""
}
