package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/optimode/emailkit"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Syntax check only
	fmt.Println("=== Syntax check ===")
	syntaxOnly := emailkit.New()

	emails := []string{
		"valid@example.com",
		"invalid-email",
		"missing@domain",
		"too..many..dots@example.com",
	}

	for _, e := range emails {
		res, err := syntaxOnly.Validate(ctx, e)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}
		status := "PASS"
		if !res.Valid {
			status = "FAIL"
		}
		fmt.Printf("%s %s -- %s\n", status, e, res.Checks[0].Details)
	}

	// 2. DNS + syntax
	fmt.Println("\n=== DNS check ===")
	dnsValidator := emailkit.New().WithDNS()

	res, _ := dnsValidator.Validate(ctx, "test@nonexistent-domain-xyz123.com")
	printResult(res)

	res, _ = dnsValidator.Validate(ctx, "test@gmail.com")
	printResult(res)

	// 3. Full pipeline (without SMTP so the example runs fast)
	fmt.Println("\n=== Domain check ===")
	fullValidator := emailkit.New().
		WithDNS().
		WithDomain(emailkit.DomainOptions{
			CheckDisposable: true,
			CheckTypos:      true,
		})

	testEmails := []string{
		"user@gmail.com",
		"user@gmial.com",      // typo
		"user@mailinator.com", // disposable
	}

	for _, e := range testEmails {
		res, _ := fullValidator.Validate(ctx, e)
		printResult(res)
	}

	_ = ctx
	_ = cancel
	os.Exit(0)
}

func printResult(r emailkit.Result) {
	b, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(b))
}
