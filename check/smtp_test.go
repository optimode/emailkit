package check_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit/check"
	"github.com/optimode/emailkit/internal/dnscache"
	"github.com/optimode/emailkit/internal/parse"
	"github.com/optimode/emailkit/internal/smtppool"
	"github.com/optimode/emailkit/types"
)

// mockMXResolver implements the resolver interface for dnscache tests.
type mockMXResolver struct {
	records []*net.MX
	err     error
}

func (m *mockMXResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return m.records, m.err
}

// testSMTPServer simulates an SMTP server on one end of a net.Pipe.
func testSMTPServer(server net.Conn, banner string, responses map[string]string) {
	defer func() { _ = server.Close() }()

	_, _ = fmt.Fprintf(server, "%s\r\n", banner)

	buf := make([]byte, 4096)
	for {
		n, err := server.Read(buf)
		if err != nil {
			return
		}
		cmd := string(buf[:n])

		for prefix, resp := range responses {
			if len(cmd) >= len(prefix) && cmd[:len(prefix)] == prefix {
				_, _ = fmt.Fprintf(server, "%s\r\n", resp)
				break
			}
		}

		if len(cmd) >= 4 && cmd[:4] == "QUIT" {
			_, _ = fmt.Fprintf(server, "221 Bye\r\n")
			return
		}
	}
}

func newTestSMTPChecker(mxRecords []*net.MX, dial func(string, string, time.Duration) (net.Conn, error)) (*check.SMTPChecker, func()) {
	cache := dnscache.NewWithResolver(2*time.Second, 1*time.Minute, &mockMXResolver{
		records: mxRecords,
	})

	pool := smtppool.New(smtppool.Config{
		HeloDomain:      "test.com",
		MailFrom:        "verify@test.com",
		ConnectTimeout:  5 * time.Second,
		CommandTimeout:  5 * time.Second,
		Port:            "25",
		MaxConnsPerHost: 2,
		Dial:            dial,
	})

	checker := check.NewSMTPChecker(check.SMTPConfig{
		HeloDomain: "test.com",
		MailFrom:   "verify@test.com",
		MaxMXHosts: 1,
	}, cache, pool)

	cleanup := func() { _ = pool.Close() }
	return checker, cleanup
}

func TestSMTPChecker_SuccessfulRCPT(t *testing.T) {
	mxRecords := []*net.MX{{Host: "mx.example.com.", Pref: 10}}
	c, cleanup := newTestSMTPChecker(mxRecords, func(network, address string, timeout time.Duration) (net.Conn, error) {
		client, server := net.Pipe()
		responses := map[string]string{
			"EHLO": "250 OK", "RSET": "250 OK",
			"MAIL FROM": "250 OK", "RCPT TO": "250 OK",
		}
		go testSMTPServer(server, "220 smtp.example.com ESMTP", responses)
		return client, nil
	})
	defer cleanup()

	parsed := parse.NewEmail("test@example.com")
	result := c.Check(context.Background(), parsed)

	assert.Equal(t, types.LevelSMTP, result.Level)
	assert.True(t, result.Passed)
	assert.Contains(t, result.Details, "RCPT TO accepted")
}

func TestSMTPChecker_RejectedRCPT(t *testing.T) {
	mxRecords := []*net.MX{{Host: "mx.example.com.", Pref: 10}}
	c, cleanup := newTestSMTPChecker(mxRecords, func(network, address string, timeout time.Duration) (net.Conn, error) {
		client, server := net.Pipe()
		responses := map[string]string{
			"EHLO": "250 OK", "MAIL FROM": "250 OK",
			"RCPT TO": "550 User not found",
		}
		go testSMTPServer(server, "220 smtp.example.com ESMTP", responses)
		return client, nil
	})
	defer cleanup()

	parsed := parse.NewEmail("test@example.com")
	result := c.Check(context.Background(), parsed)

	assert.Equal(t, types.LevelSMTP, result.Level)
	assert.False(t, result.Passed)
	assert.Equal(t, 550, result.SMTPCode)
}

func TestSMTPChecker_ConnectionError(t *testing.T) {
	mxRecords := []*net.MX{{Host: "mx.example.com.", Pref: 10}}
	c, cleanup := newTestSMTPChecker(mxRecords, func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, fmt.Errorf("connection refused")
	})
	defer cleanup()

	parsed := parse.NewEmail("test@example.com")
	result := c.Check(context.Background(), parsed)

	assert.Equal(t, types.LevelSMTP, result.Level)
	assert.False(t, result.Passed)
}

func TestSMTPChecker_InvalidEmail(t *testing.T) {
	mxRecords := []*net.MX{{Host: "mx.example.com.", Pref: 10}}
	c, cleanup := newTestSMTPChecker(mxRecords, func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, fmt.Errorf("should not be called")
	})
	defer cleanup()

	parsed := parse.NewEmail("invalid")
	result := c.Check(context.Background(), parsed)

	assert.False(t, result.Passed)
	assert.Contains(t, result.Details, "skipped")
}

func TestSMTPChecker_NoMXRecords(t *testing.T) {
	cache := dnscache.NewWithResolver(2*time.Second, 1*time.Minute, &mockMXResolver{
		err: &net.DNSError{Err: "no such host"},
	})

	pool := smtppool.New(smtppool.Config{
		HeloDomain:     "test.com",
		MailFrom:       "verify@test.com",
		ConnectTimeout: 5 * time.Second,
		CommandTimeout: 5 * time.Second,
		Port:           "25",
	})
	defer func() { _ = pool.Close() }()

	checker := check.NewSMTPChecker(check.SMTPConfig{
		HeloDomain: "test.com",
		MailFrom:   "verify@test.com",
		MaxMXHosts: 1,
	}, cache, pool)

	parsed := parse.NewEmail("test@example.com")
	result := checker.Check(context.Background(), parsed)

	assert.False(t, result.Passed)
	assert.Contains(t, result.Details, "MX lookup failed")
}

func TestSMTPChecker_TemporaryFailure(t *testing.T) {
	mxRecords := []*net.MX{{Host: "mx.example.com.", Pref: 10}}
	c, cleanup := newTestSMTPChecker(mxRecords, func(network, address string, timeout time.Duration) (net.Conn, error) {
		client, server := net.Pipe()
		responses := map[string]string{
			"EHLO": "250 OK", "MAIL FROM": "250 OK",
			"RCPT TO": "450 Try again later",
		}
		go testSMTPServer(server, "220 smtp.example.com ESMTP", responses)
		return client, nil
	})
	defer cleanup()

	parsed := parse.NewEmail("test@example.com")
	result := c.Check(context.Background(), parsed)

	assert.Equal(t, types.LevelSMTP, result.Level)
	assert.False(t, result.Passed)
	assert.Contains(t, result.Details, "SMTP probe failed")
}

func TestSMTPChecker_ConnectionReuse(t *testing.T) {
	dialCount := 0
	mxRecords := []*net.MX{{Host: "mx.example.com.", Pref: 10}}

	c, cleanup := newTestSMTPChecker(mxRecords, func(network, address string, timeout time.Duration) (net.Conn, error) {
		dialCount++
		client, server := net.Pipe()
		responses := map[string]string{
			"EHLO": "250 OK", "RSET": "250 OK",
			"MAIL FROM": "250 OK", "RCPT TO": "250 OK",
		}
		go testSMTPServer(server, "220 smtp.example.com ESMTP", responses)
		return client, nil
	})
	defer cleanup()

	ctx := context.Background()
	parsed1 := parse.NewEmail("user1@example.com")
	parsed2 := parse.NewEmail("user2@example.com")

	result1 := c.Check(ctx, parsed1)
	assert.True(t, result1.Passed)

	result2 := c.Check(ctx, parsed2)
	assert.True(t, result2.Passed)

	// Should have reused the connection (only 1 dial)
	assert.Equal(t, 1, dialCount)
}
