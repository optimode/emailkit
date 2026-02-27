package smtppool_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit/internal/smtppool"
)

// mockSMTPServer simulates an SMTP server on a net.Pipe connection.
func mockSMTPServer(server net.Conn, responses map[string]string) {
	defer func() { _ = server.Close() }()

	// Send banner
	_, _ = fmt.Fprintf(server, "220 mock.smtp ESMTP\r\n")

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

func TestPool_NewConnectionAndReuse(t *testing.T) {
	dialCount := 0

	cfg := smtppool.Config{
		HeloDomain:      "test.com",
		MailFrom:        "verify@test.com",
		ConnectTimeout:  5 * time.Second,
		CommandTimeout:  5 * time.Second,
		Port:            "25",
		MaxConnsPerHost: 2,
		MaxUsesPerConn:  10,
		MaxConnAge:      1 * time.Minute,
		Dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			dialCount++
			client, server := net.Pipe()
			responses := map[string]string{
				"EHLO":      "250 OK",
				"RSET":      "250 OK",
				"MAIL FROM": "250 OK",
				"RCPT TO":   "250 OK",
			}
			go mockSMTPServer(server, responses)
			return client, nil
		},
	}

	pool := smtppool.New(cfg)
	defer func() { _ = pool.Close() }()

	// First check: creates new connection
	code, _, err := pool.CheckRCPT("mx.example.com", "user1@example.com")
	assert.NoError(t, err)
	assert.Equal(t, 250, code)
	assert.Equal(t, 1, dialCount)

	// Second check: should reuse the connection (RSET)
	code, _, err = pool.CheckRCPT("mx.example.com", "user2@example.com")
	assert.NoError(t, err)
	assert.Equal(t, 250, code)
	assert.Equal(t, 1, dialCount) // still 1, connection was reused
}

func TestPool_DifferentHosts(t *testing.T) {
	dialCount := 0

	cfg := smtppool.Config{
		HeloDomain:      "test.com",
		MailFrom:        "verify@test.com",
		ConnectTimeout:  5 * time.Second,
		CommandTimeout:  5 * time.Second,
		Port:            "25",
		MaxConnsPerHost: 2,
		Dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			dialCount++
			client, server := net.Pipe()
			responses := map[string]string{
				"EHLO": "250 OK", "RSET": "250 OK",
				"MAIL FROM": "250 OK", "RCPT TO": "250 OK",
			}
			go mockSMTPServer(server, responses)
			return client, nil
		},
	}

	pool := smtppool.New(cfg)
	defer func() { _ = pool.Close() }()

	_, _, _ = pool.CheckRCPT("mx1.example.com", "user@example.com")
	_, _, _ = pool.CheckRCPT("mx2.example.com", "user@other.com")
	assert.Equal(t, 2, dialCount) // different hosts, different connections
}

func TestPool_RejectedRCPT(t *testing.T) {
	cfg := smtppool.Config{
		HeloDomain:     "test.com",
		MailFrom:       "verify@test.com",
		ConnectTimeout: 5 * time.Second,
		CommandTimeout: 5 * time.Second,
		Port:           "25",
		Dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			client, server := net.Pipe()
			responses := map[string]string{
				"EHLO":      "250 OK",
				"MAIL FROM": "250 OK",
				"RCPT TO":   "550 User not found",
			}
			go mockSMTPServer(server, responses)
			return client, nil
		},
	}

	pool := smtppool.New(cfg)
	defer func() { _ = pool.Close() }()

	code, _, err := pool.CheckRCPT("mx.example.com", "nobody@example.com")
	assert.NoError(t, err)
	assert.Equal(t, 550, code)
}

func TestPool_ConnectionError(t *testing.T) {
	cfg := smtppool.Config{
		HeloDomain:     "test.com",
		MailFrom:       "verify@test.com",
		ConnectTimeout: 1 * time.Second,
		CommandTimeout: 1 * time.Second,
		Port:           "25",
		Dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	pool := smtppool.New(cfg)
	defer func() { _ = pool.Close() }()

	_, _, err := pool.CheckRCPT("mx.example.com", "user@example.com")
	assert.Error(t, err)
}

func TestPool_CloseAndReject(t *testing.T) {
	cfg := smtppool.Config{
		HeloDomain:     "test.com",
		MailFrom:       "verify@test.com",
		ConnectTimeout: 5 * time.Second,
		CommandTimeout: 5 * time.Second,
		Port:           "25",
		Dial: func(network, address string, timeout time.Duration) (net.Conn, error) {
			client, server := net.Pipe()
			responses := map[string]string{
				"EHLO": "250 OK", "RSET": "250 OK",
				"MAIL FROM": "250 OK", "RCPT TO": "250 OK",
			}
			go mockSMTPServer(server, responses)
			return client, nil
		},
	}

	pool := smtppool.New(cfg)
	_ = pool.Close()

	_, _, err := pool.CheckRCPT("mx.example.com", "user@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}
