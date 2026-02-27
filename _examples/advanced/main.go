package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/optimode/emailkit"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Full pipeline with SMTP probe
	validator := emailkit.New().
		WithDNS(emailkit.DNSOptions{
			Timeout:     5 * time.Second,
			FallbackToA: false,
		}).
		WithDomain(emailkit.DomainOptions{
			CheckDisposable: true,
			CheckTypos:      true,
			TypoThreshold:   2,
		}).
		WithSMTP(emailkit.SMTPOptions{
			HeloDomain:     "example.com",
			MailFrom:       "noreply@example.com",
			ConnectTimeout: 5 * time.Second,
			CommandTimeout: 15 * time.Second,
			MaxMXHosts:     2,
		})

	// Single validation
	fmt.Println("=== Single validation ===")
	res, err := validator.Validate(ctx, "test@gmail.com")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	printJSON(res)

	// Bulk validation
	fmt.Println("\n=== Bulk validation ===")
	emails := []string{
		"user1@gmail.com",
		"user2@yahoo.com",
		"invalid@",
		"user@mailinator.com",
		"user@gmial.com",
	}

	results, err := validator.ValidateMany(ctx, emails, emailkit.ConcurrencyOptions{
		Workers: 3,
	})
	if err != nil {
		fmt.Printf("Partial error: %v\n", err)
	}
	for _, r := range results {
		fmt.Printf("%-30s valid=%v\n", r.Email, r.Valid)
		for _, c := range r.Checks {
			icon := "  PASS"
			if !c.Passed {
				icon = "  FAIL"
			}
			fmt.Printf("%s [%s] %s", icon, c.Level, c.Details)
			if c.Suggestion != "" {
				fmt.Printf(" (did you mean: %s?)", c.Suggestion)
			}
			fmt.Println()
		}
	}
}

func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
