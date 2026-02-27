.PHONY: build test vet lint cover check clean

# Run all checks (CI entry point)
check: vet lint test

# Verify the library compiles
build:
	go build ./...

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-v:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run go vet
vet:
	go vet ./...

# Run golangci-lint (install: https://golangci-lint.run/welcome/install/)
lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping (see https://golangci-lint.run/welcome/install/)"

# Generate test coverage report
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm -f coverage.out

# Generate HTML coverage report and open in browser
cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html
	@rm -f coverage.out

# Tidy and verify module dependencies
tidy:
	go mod tidy
	go mod verify

# Remove generated files
clean:
	@rm -f coverage.out coverage.html
