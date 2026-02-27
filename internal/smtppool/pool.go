// Package smtppool provides a thread-safe SMTP connection pool that reuses
// TCP connections via the RSET command for efficient bulk email validation.
package smtppool

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Config configures the SMTP connection pool.
type Config struct {
	HeloDomain      string
	MailFrom        string
	ConnectTimeout  time.Duration
	CommandTimeout  time.Duration
	Port            string
	MaxConnsPerHost int           // max idle connections per MX host (default: 3)
	MaxUsesPerConn  int           // max RCPT checks per connection before reconnect (default: 100)
	MaxConnAge      time.Duration // max lifetime of a connection (default: 5m)
	// Dial is injectable for testing. Defaults to net.DialTimeout.
	Dial func(network, address string, timeout time.Duration) (net.Conn, error)
}

// Pool manages SMTP connections per MX host.
type Pool struct {
	cfg    Config
	mu     sync.Mutex
	hosts  map[string][]*conn
	closed bool
}

type conn struct {
	netConn   net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	createdAt time.Time
	uses      int
}

// New creates a new SMTP connection pool.
func New(cfg Config) *Pool {
	if cfg.Dial == nil {
		cfg.Dial = net.DialTimeout
	}
	if cfg.MaxConnsPerHost <= 0 {
		cfg.MaxConnsPerHost = 3
	}
	if cfg.MaxUsesPerConn <= 0 {
		cfg.MaxUsesPerConn = 100
	}
	if cfg.MaxConnAge <= 0 {
		cfg.MaxConnAge = 5 * time.Minute
	}
	return &Pool{
		cfg:   cfg,
		hosts: make(map[string][]*conn),
	}
}

// CheckRCPT performs an SMTP RCPT TO check using a pooled connection.
// For new connections: Banner → EHLO → MAIL FROM → RCPT TO
// For reused connections: RSET → MAIL FROM → RCPT TO
// Returns the RCPT TO response code and message.
func (p *Pool) CheckRCPT(mxHost, email string) (code int, msg string, err error) {
	c, isNew, err := p.get(mxHost)
	if err != nil {
		return 0, "", err
	}

	code, msg, err = p.doCheck(c, mxHost, email, isNew)
	if err != nil {
		// Connection is broken, discard it
		_ = c.netConn.Close()
		return 0, "", err
	}

	p.put(mxHost, c)
	return code, msg, nil
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	for host, conns := range p.hosts {
		for _, c := range conns {
			sendQuit(c)
			_ = c.netConn.Close()
		}
		delete(p.hosts, host)
	}
	return nil
}

// get retrieves an existing connection from the pool or creates a new one.
func (p *Pool) get(mxHost string) (*conn, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, false, errors.New("smtppool: pool is closed")
	}

	conns := p.hosts[mxHost]

	// Try to find a reusable connection (LIFO for better locality)
	for i := len(conns) - 1; i >= 0; i-- {
		c := conns[i]
		if c.uses >= p.cfg.MaxUsesPerConn || time.Since(c.createdAt) > p.cfg.MaxConnAge {
			// Too old or too many uses, close and remove
			sendQuit(c)
			_ = c.netConn.Close()
			conns = append(conns[:i], conns[i+1:]...)
			continue
		}
		// Take this connection out of the pool
		conns = append(conns[:i], conns[i+1:]...)
		p.hosts[mxHost] = conns
		return c, false, nil
	}
	p.hosts[mxHost] = conns

	// No reusable connection, create a new one
	c, err := p.dial(mxHost)
	if err != nil {
		return nil, false, err
	}
	return c, true, nil
}

// put returns a connection to the pool for reuse.
func (p *Pool) put(mxHost string, c *conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || len(p.hosts[mxHost]) >= p.cfg.MaxConnsPerHost {
		sendQuit(c)
		_ = c.netConn.Close()
		return
	}

	p.hosts[mxHost] = append(p.hosts[mxHost], c)
}

// dial creates a new TCP connection to the MX host.
func (p *Pool) dial(mxHost string) (*conn, error) {
	address := net.JoinHostPort(mxHost, p.cfg.Port)
	netConn, err := p.cfg.Dial("tcp", address, p.cfg.ConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", address, err)
	}

	return &conn{
		netConn:   netConn,
		reader:    bufio.NewReader(netConn),
		writer:    bufio.NewWriter(netConn),
		createdAt: time.Now(),
	}, nil
}

// doCheck performs the SMTP check on a connection.
func (p *Pool) doCheck(c *conn, mxHost, email string, isNew bool) (int, string, error) {
	deadline := time.Now().Add(p.cfg.CommandTimeout)
	if err := c.netConn.SetDeadline(deadline); err != nil {
		return 0, "", fmt.Errorf("set deadline: %w", err)
	}

	if isNew {
		// Read banner
		code, msg, err := readResponse(c.reader)
		if err != nil {
			return 0, "", fmt.Errorf("read banner: %w", err)
		}
		if code >= 500 {
			return 0, "", fmt.Errorf("server rejected connection: %d %s", code, msg)
		}

		// EHLO
		code, msg, err = command(c, fmt.Sprintf("EHLO %s\r\n", p.cfg.HeloDomain))
		if err != nil {
			return 0, "", fmt.Errorf("EHLO failed: %w", err)
		}
		if code >= 400 {
			return 0, "", fmt.Errorf("EHLO rejected: %d %s", code, msg)
		}
	} else {
		// RSET to start a fresh transaction on the reused connection
		code, msg, err := command(c, "RSET\r\n")
		if err != nil {
			return 0, "", fmt.Errorf("RSET failed: %w", err)
		}
		if code >= 400 {
			return 0, "", fmt.Errorf("RSET rejected: %d %s", code, msg)
		}
	}

	// MAIL FROM
	code, msg, err := command(c, fmt.Sprintf("MAIL FROM:<%s>\r\n", p.cfg.MailFrom))
	if err != nil {
		return 0, "", fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if code >= 500 {
		return code, msg, nil
	}
	if code >= 400 {
		return 0, "", fmt.Errorf("MAIL FROM temporary failure: %d %s", code, msg)
	}

	// RCPT TO
	code, msg, err = command(c, fmt.Sprintf("RCPT TO:<%s>\r\n", email))
	if err != nil {
		return 0, "", fmt.Errorf("RCPT TO failed: %w", err)
	}

	c.uses++
	return code, msg, nil
}

// command sends an SMTP command and reads the response.
func command(c *conn, cmd string) (int, string, error) {
	if _, err := c.writer.WriteString(cmd); err != nil {
		return 0, "", err
	}
	if err := c.writer.Flush(); err != nil {
		return 0, "", err
	}
	return readResponse(c.reader)
}

// sendQuit sends a QUIT command (best-effort, ignores errors).
func sendQuit(c *conn) {
	_ = c.netConn.SetDeadline(time.Now().Add(2 * time.Second))
	_, _ = c.writer.WriteString("QUIT\r\n")
	_ = c.writer.Flush()
}

// readResponse reads a (possibly multi-line) SMTP response.
func readResponse(r *bufio.Reader) (code int, full string, err error) {
	var lines []string
	for {
		line, readErr := r.ReadString('\n')
		if readErr != nil {
			return 0, "", fmt.Errorf("read SMTP response: %w", readErr)
		}
		line = strings.TrimRight(line, "\r\n")
		if len(line) < 3 {
			return 0, "", errors.New("SMTP response line too short")
		}
		lines = append(lines, line)
		// If the 4th character is not '-', this is the last line
		if len(line) < 4 || line[3] != '-' {
			break
		}
	}

	lastLine := lines[len(lines)-1]
	if _, err := fmt.Sscanf(lastLine[:3], "%d", &code); err != nil {
		return 0, "", fmt.Errorf("invalid SMTP response code %q: %w", lastLine[:3], err)
	}
	return code, strings.Join(lines, " | "), nil
}
