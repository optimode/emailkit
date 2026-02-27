// Package types contains the shared types for emailkit.
// This package does not import anything from other emailkit packages
// to avoid circular imports.
package types

// CheckLevel identifies the validation level.
type CheckLevel = string

const (
	LevelSyntax CheckLevel = "syntax"
	LevelDNS    CheckLevel = "dns"
	LevelDomain CheckLevel = "domain"
	LevelSMTP   CheckLevel = "smtp"
)

// CheckResult is the outcome of a single validation level.
type CheckResult struct {
	Level      CheckLevel `json:"level"`
	Passed     bool       `json:"passed"`
	Details    string     `json:"details,omitempty"`
	MXHost     string     `json:"mxHost,omitempty"`
	SMTPCode   int        `json:"smtpCode,omitempty"`
	Suggestion string     `json:"suggestion,omitempty"`
}
