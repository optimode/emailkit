# Examples

Standalone runnable examples for emailkit.

## Basic

Demonstrates syntax-only, DNS, and domain validation without any SMTP network calls.

```sh
go run ./_examples/basic
```

## Advanced

Full validation pipeline with SMTP probe and concurrent bulk validation.
Requires network access and a valid SMTP configuration.

```sh
go run ./_examples/advanced
```
