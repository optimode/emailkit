package emailkit

import "errors"

var (
	// ErrNoChecksConfigured is returned when Validate() is called
	// but no validation level is configured (not even syntax).
	ErrNoChecksConfigured = errors.New("emailkit: no validation checks configured")

	// ErrInvalidSMTPOptions is returned when WithSMTP is called
	// but HeloDomain or MailFrom is missing.
	ErrInvalidSMTPOptions = errors.New("emailkit: SMTPOptions requires HeloDomain and MailFrom")
)
