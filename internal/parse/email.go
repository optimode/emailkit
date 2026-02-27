package parse

import (
	"net/mail"
	"strings"

	"golang.org/x/net/idna"
)

// Email is the internal representation of a parsed email address.
// The check/ packages receive this as parameter.
type Email struct {
	Raw           string // the original, trimmed input
	Local         string // the part before @
	Domain        string // the part after @, ASCII/Punycode form (for DNS/SMTP)
	DomainUnicode string // the part after @, Unicode form (for display/typo detection)
	Valid         bool   // false if Raw cannot be parsed
}

// NewEmail attempts to parse the given email string.
// If parsing fails, Valid=false but Raw is always populated.
// Supports internationalized email addresses (RFC 6531 / EAI) and
// internationalized domain names (IDNA2008).
func NewEmail(raw string) Email {
	raw = strings.TrimSpace(raw)

	// Try standard parsing first (handles most ASCII emails)
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		addr, err = mail.ParseAddress("<" + raw + ">")
		if err != nil {
			// Fallback: manual parsing for internationalized local parts
			// that net/mail doesn't support (RFC 6531 / SMTPUTF8)
			return parseManual(raw)
		}
	}

	parts := strings.SplitN(addr.Address, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Email{Raw: raw, Valid: false}
	}

	return buildEmail(raw, parts[0], parts[1])
}

// parseManual handles email addresses that net/mail.ParseAddress rejects,
// such as those with Unicode local parts (RFC 6531 SMTPUTF8).
func parseManual(raw string) Email {
	atIdx := strings.LastIndex(raw, "@")
	if atIdx < 1 || atIdx >= len(raw)-1 {
		return Email{Raw: raw, Valid: false}
	}
	local := raw[:atIdx]
	domain := raw[atIdx+1:]
	if local == "" || domain == "" {
		return Email{Raw: raw, Valid: false}
	}
	return buildEmail(raw, local, domain)
}

// buildEmail constructs an Email with proper IDNA domain handling.
// The Domain field is always ASCII/Punycode (for DNS/SMTP),
// DomainUnicode is the human-readable Unicode form.
func buildEmail(raw, local, domain string) Email {
	domainLower := strings.ToLower(domain)

	asciiDomain, unicodeDomain, ok := convertDomain(domainLower)
	if !ok {
		return Email{Raw: raw, Valid: false}
	}

	return Email{
		Raw:           raw,
		Local:         local,
		Domain:        asciiDomain,
		DomainUnicode: unicodeDomain,
		Valid:         true,
	}
}

// convertDomain converts a domain to both ASCII/Punycode and Unicode forms.
// Returns (ascii, unicode, ok). ok is false if the domain contains
// non-ASCII characters that fail IDNA2008 validation.
func convertDomain(domain string) (ascii, unicode string, ok bool) {
	hasNonASCII := false
	for _, r := range domain {
		if r > 127 {
			hasNonASCII = true
			break
		}
	}

	if hasNonASCII {
		// Internationalized domain: convert to Punycode via IDNA2008
		a, err := idna.Lookup.ToASCII(domain)
		if err != nil {
			return "", "", false
		}
		return a, domain, true
	}

	// Pure ASCII domain: try to get Unicode display form
	// (handles existing Punycode like xn--mnchen-3ya.de → münchen.de)
	u, err := idna.Display.ToUnicode(domain)
	if err != nil {
		u = domain
	}
	return domain, u, true
}
