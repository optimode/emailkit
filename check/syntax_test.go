package check_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit/check"
	"github.com/optimode/emailkit/internal/parse"
)

func TestSyntaxChecker(t *testing.T) {
	c := check.NewSyntaxChecker()
	ctx := context.Background()

	tests := []struct {
		name   string
		email  string
		wantOK bool
	}{
		{"valid simple", "user@example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"valid with dots", "first.last@example.com", true},
		{"valid quoted local", `"user name"@example.com`, true},
		{"valid subdomain", "user@mail.example.co.uk", true},
		{"empty", "", false},
		{"no at sign", "userexample.com", false},
		{"no domain", "user@", false},
		{"no local", "@example.com", false},
		{"double dot local", "user..name@example.com", false},
		{"leading dot local", ".user@example.com", false},
		{"trailing dot local", "user.@example.com", false},
		{"consecutive dots domain", "user@exam..ple.com", false},
		{"too long total", string(make([]byte, 255)) + "@example.com", false},
		{"numeric TLD", "user@example.123", false},
		{"label starts with hyphen", "user@-example.com", false},
		{"label ends with hyphen", "user@example-.com", false},

		// IDN (Internationalized Domain Names)
		{"valid IDN german", "user@münchen.de", true},
		{"valid IDN japanese", "user@例え.jp", true},
		{"valid IDN cyrillic", "user@почта.рф", true},
		{"valid Punycode", "user@xn--mnchen-3ya.de", true},

		// EAI (Email Address Internationalization / RFC 6531)
		{"valid EAI chinese local", "用户@example.com", true},
		{"valid EAI arabic local", "معلومات@example.com", true},
		{"valid EAI both unicode", "用户@münchen.de", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parse.NewEmail(tt.email)
			result := c.Check(ctx, parsed)
			assert.Equal(t, tt.wantOK, result.Passed, "Details: %s", result.Details)
		})
	}
}
