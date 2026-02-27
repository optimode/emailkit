package parse_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit/internal/parse"
)

func TestNewEmail_ASCII(t *testing.T) {
	e := parse.NewEmail("user@example.com")
	assert.True(t, e.Valid)
	assert.Equal(t, "user", e.Local)
	assert.Equal(t, "example.com", e.Domain)
	assert.Equal(t, "example.com", e.DomainUnicode)
}

func TestNewEmail_Whitespace(t *testing.T) {
	e := parse.NewEmail("  user@example.com  ")
	assert.True(t, e.Valid)
	assert.Equal(t, "user", e.Local)
}

func TestNewEmail_Invalid(t *testing.T) {
	tests := []string{
		"",
		"noatsign",
		"@nodomain",
		"nolocal@",
	}
	for _, raw := range tests {
		e := parse.NewEmail(raw)
		assert.False(t, e.Valid, "expected invalid for %q", raw)
	}
}

func TestNewEmail_IDN_UnicodeDomain(t *testing.T) {
	// Unicode domain should be converted to Punycode in Domain,
	// and kept as Unicode in DomainUnicode
	e := parse.NewEmail("user@münchen.de")
	assert.True(t, e.Valid)
	assert.Equal(t, "user", e.Local)
	assert.Equal(t, "xn--mnchen-3ya.de", e.Domain)
	assert.Equal(t, "münchen.de", e.DomainUnicode)
}

func TestNewEmail_IDN_PunycodeDomain(t *testing.T) {
	// Already-Punycode domain should be kept as-is in Domain,
	// and decoded to Unicode in DomainUnicode
	e := parse.NewEmail("user@xn--mnchen-3ya.de")
	assert.True(t, e.Valid)
	assert.Equal(t, "xn--mnchen-3ya.de", e.Domain)
	assert.Equal(t, "münchen.de", e.DomainUnicode)
}

func TestNewEmail_EAI_UnicodeLocal(t *testing.T) {
	// Unicode local part (RFC 6531 SMTPUTF8)
	e := parse.NewEmail("用户@example.com")
	assert.True(t, e.Valid)
	assert.Equal(t, "用户", e.Local)
	assert.Equal(t, "example.com", e.Domain)
}

func TestNewEmail_EAI_BothUnicode(t *testing.T) {
	// Both Unicode local and domain
	e := parse.NewEmail("用户@münchen.de")
	assert.True(t, e.Valid)
	assert.Equal(t, "用户", e.Local)
	assert.Equal(t, "xn--mnchen-3ya.de", e.Domain)
	assert.Equal(t, "münchen.de", e.DomainUnicode)
}

func TestNewEmail_IDN_JapaneseDomain(t *testing.T) {
	e := parse.NewEmail("user@例え.jp")
	assert.True(t, e.Valid)
	assert.Equal(t, "xn--r8jz45g.jp", e.Domain)
	assert.Equal(t, "例え.jp", e.DomainUnicode)
}

func TestNewEmail_IDN_CyrillicDomain(t *testing.T) {
	e := parse.NewEmail("user@почта.рф")
	assert.True(t, e.Valid)
	assert.Equal(t, "xn--80a1acny.xn--p1ai", e.Domain)
	assert.Equal(t, "почта.рф", e.DomainUnicode)
}

func TestNewEmail_DomainCaseNormalization(t *testing.T) {
	e := parse.NewEmail("user@EXAMPLE.COM")
	assert.True(t, e.Valid)
	assert.Equal(t, "example.com", e.Domain)
}
