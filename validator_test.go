package emailkit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit"
)

func TestNew_SyntaxOnly(t *testing.T) {
	v := emailkit.New()
	ctx := context.Background()

	res, err := v.Validate(ctx, "user@example.com")
	assert.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Len(t, res.Checks, 1)
	assert.Equal(t, emailkit.LevelSyntax, res.Checks[0].Level)

	res, err = v.Validate(ctx, "invalid")
	assert.NoError(t, err)
	assert.False(t, res.Valid)
}

func TestNew_InvalidSMTPOptions(t *testing.T) {
	v := emailkit.New().WithSMTP(emailkit.SMTPOptions{
		// HeloDomain and MailFrom are missing
	})
	_, err := v.Validate(context.Background(), "user@example.com")
	assert.ErrorIs(t, err, emailkit.ErrInvalidSMTPOptions)
}

func TestValidateMany(t *testing.T) {
	v := emailkit.New()
	ctx := context.Background()

	emails := []string{"a@example.com", "b@example.com", "invalid"}
	results, err := v.ValidateMany(ctx, emails)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	assert.True(t, results[0].Valid)
	assert.True(t, results[1].Valid)
	assert.False(t, results[2].Valid)
}

func TestResult_FailedChecks(t *testing.T) {
	v := emailkit.New()
	res, _ := v.Validate(context.Background(), "bad email")
	assert.Len(t, res.FailedChecks(), 1)
	assert.Equal(t, emailkit.LevelSyntax, res.FailedChecks()[0].Level)
}

func TestResult_CheckFor(t *testing.T) {
	v := emailkit.New()
	res, _ := v.Validate(context.Background(), "user@example.com")

	check, found := res.CheckFor(emailkit.LevelSyntax)
	assert.True(t, found)
	assert.True(t, check.Passed)

	_, found = res.CheckFor(emailkit.LevelDNS)
	assert.False(t, found) // DNS was not configured
}

func TestValidateAll(t *testing.T) {
	v := emailkit.New()
	ctx := context.Background()

	res, err := v.ValidateAll(ctx, "user@example.com")
	assert.NoError(t, err)
	assert.True(t, res.Valid)

	res, err = v.ValidateAll(ctx, "invalid")
	assert.NoError(t, err)
	assert.False(t, res.Valid)
}
