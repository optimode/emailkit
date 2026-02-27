// Package emailkit is an email validation library that validates email
// addresses at the syntax, DNS, domain and SMTP levels.
//
// Basic usage:
//
//	result, err := emailkit.New().Validate(ctx, "user@example.com")
//
// Full pipeline:
//
//	result, err := emailkit.New().
//	    WithDNS().
//	    WithDomain().
//	    WithSMTP(emailkit.SMTPOptions{
//	        HeloDomain: "myapp.com",
//	        MailFrom:   "verify@myapp.com",
//	    }).
//	    Validate(ctx, "user@example.com")
package emailkit

import "github.com/optimode/emailkit/types"

// CheckResult is a re-export from the types package so that consumers
// don't need to import the types package directly.
type CheckResult = types.CheckResult

// CheckLevel is a re-export.
type CheckLevel = types.CheckLevel

// Level constants re-exported.
const (
	LevelSyntax = types.LevelSyntax
	LevelDNS    = types.LevelDNS
	LevelDomain = types.LevelDomain
	LevelSMTP   = types.LevelSMTP
)
