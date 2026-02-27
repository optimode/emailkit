package emailkit_test

import (
	"context"
	"fmt"

	"github.com/optimode/emailkit"
)

func ExampleNew() {
	v := emailkit.New()
	result, _ := v.Validate(context.Background(), "user@example.com")
	fmt.Println(result.Valid)
	// Output: true
}

func ExampleValidator_Validate() {
	v := emailkit.New()

	result, _ := v.Validate(context.Background(), "user@example.com")
	fmt.Println(result.Valid, result.Checks[0].Details)

	result, _ = v.Validate(context.Background(), "invalid")
	fmt.Println(result.Valid, result.Checks[0].Details)
	// Output:
	// true syntax ok
	// false invalid email syntax
}

func ExampleValidator_Validate_idn() {
	v := emailkit.New()

	// Internationalized Domain Name (German)
	result, _ := v.Validate(context.Background(), "user@münchen.de")
	fmt.Println(result.Valid)

	// Email Address Internationalization / RFC 6531 (Chinese local part)
	result, _ = v.Validate(context.Background(), "用户@example.com")
	fmt.Println(result.Valid)
	// Output:
	// true
	// true
}

func ExampleValidator_ValidateAll() {
	v := emailkit.New()
	result, _ := v.ValidateAll(context.Background(), "bad email")

	for _, c := range result.FailedChecks() {
		fmt.Printf("[%s] %s\n", c.Level, c.Details)
	}
	// Output:
	// [syntax] invalid email syntax
}

func ExampleValidator_ValidateMany() {
	v := emailkit.New()
	emails := []string{"alice@example.com", "invalid", "bob@example.com"}

	results, _ := v.ValidateMany(context.Background(), emails, emailkit.ConcurrencyOptions{
		Workers: 2,
	})

	for _, r := range results {
		fmt.Printf("%-20s valid=%v\n", r.Email, r.Valid)
	}
	// Output:
	// alice@example.com    valid=true
	// invalid              valid=false
	// bob@example.com      valid=true
}

func ExampleResult_CheckFor() {
	v := emailkit.New()
	result, _ := v.Validate(context.Background(), "user@example.com")

	if syntax, ok := result.CheckFor(emailkit.LevelSyntax); ok {
		fmt.Println(syntax.Passed, syntax.Details)
	}
	// Output: true syntax ok
}

func ExampleResult_FailedChecks() {
	v := emailkit.New()
	result, _ := v.Validate(context.Background(), "missing-at-sign")

	for _, c := range result.FailedChecks() {
		fmt.Printf("[%s] %s\n", c.Level, c.Details)
	}
	// Output:
	// [syntax] invalid email syntax
}

func ExampleValidator_WithDomain() {
	v := emailkit.New().WithDomain()

	// Typo detection (does not fail, populates Suggestion)
	result, _ := v.Validate(context.Background(), "user@gmial.com")
	domain, _ := result.CheckFor(emailkit.LevelDomain)
	fmt.Println(result.Valid, domain.Suggestion)
	// Output: true gmail.com
}

func ExampleValidator_Close() {
	v := emailkit.New().WithSMTP(emailkit.SMTPOptions{
		HeloDomain: "myapp.com",
		MailFrom:   "verify@myapp.com",
	})
	defer func() { _ = v.Close() }()

	fmt.Println("validator created with SMTP pool")
	// Output: validator created with SMTP pool
}
