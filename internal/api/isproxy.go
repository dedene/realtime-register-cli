package api

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// IsProxy protocol (is.yoursrs.com:2001):
//
//	> STARTTLS
//	< 100 OK
//	> LOGIN <api-key>
//	< 100 Login ok
//	> IS example.com
//	< example.com available
//	> QUIT
const (
	IsProxyHost        = "is.yoursrs.com"
	IsProxySandboxHost = "is.yoursrs-ote.com"
	IsProxyPort        = 2001
	IsProxyTimeout     = 30 * time.Second
)

// IsProxyClient handles bulk domain availability checks via the IsProxy protocol.
type IsProxyClient struct {
	conn   net.Conn
	reader *bufio.Reader
	apiKey string
}

// IsProxyResult represents a single domain check result.
type IsProxyResult struct {
	Domain    string `json:"domain"`
	TLD       string `json:"tld"`
	Available bool   `json:"available"`
}

// NewIsProxyClient creates a new IsProxy client.
func NewIsProxyClient(apiKey string) *IsProxyClient {
	return &IsProxyClient{apiKey: apiKey}
}

// Connect establishes a connection to the IsProxy server and upgrades to TLS.
func (c *IsProxyClient) Connect() error {
	conn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", IsProxyHost, IsProxyPort),
		IsProxyTimeout)
	if err != nil {
		return fmt.Errorf("connect to IsProxy: %w", err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)

	if err := c.startTLS(); err != nil {
		_ = c.conn.Close()
		return err
	}

	if err := c.auth(); err != nil {
		_ = c.Close()
		return err
	}

	return nil
}

// Close closes the connection after sending QUIT.
func (c *IsProxyClient) Close() error {
	if c.conn == nil {
		return nil
	}
	_, _ = fmt.Fprintf(c.conn, "QUIT\r\n")
	return c.conn.Close()
}

func (c *IsProxyClient) startTLS() error {
	if _, err := fmt.Fprintf(c.conn, "STARTTLS\r\n"); err != nil {
		return fmt.Errorf("send STARTTLS: %w", err)
	}

	resp, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read STARTTLS response: %w", err)
	}

	if !strings.HasPrefix(strings.TrimSpace(resp), "100") {
		return fmt.Errorf("STARTTLS failed: %s", strings.TrimSpace(resp))
	}

	tlsConn := tls.Client(c.conn, &tls.Config{
		ServerName: IsProxyHost,
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake: %w", err)
	}

	c.conn = tlsConn
	c.reader = bufio.NewReader(tlsConn)
	return nil
}

func (c *IsProxyClient) auth() error {
	if _, err := fmt.Fprintf(c.conn, "LOGIN %s\r\n", c.apiKey); err != nil {
		return fmt.Errorf("send LOGIN: %w", err)
	}

	resp, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read LOGIN response: %w", err)
	}

	resp = strings.TrimSpace(resp)
	if !strings.HasPrefix(resp, "100") {
		return fmt.Errorf("IsProxy auth failed: %s", resp)
	}

	return nil
}

// Check checks a single domain availability.
func (c *IsProxyClient) Check(domain string) (*IsProxyResult, error) {
	if _, err := fmt.Fprintf(c.conn, "IS %s\r\n", domain); err != nil {
		return nil, fmt.Errorf("send IS: %w", err)
	}

	resp, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read IS response: %w", err)
	}

	return parseCheckResponse(resp)
}

// CheckMany checks multiple domains efficiently.
func (c *IsProxyClient) CheckMany(domains []string) ([]IsProxyResult, error) {
	results := make([]IsProxyResult, 0, len(domains))

	for _, domain := range domains {
		result, err := c.Check(domain)
		if err != nil {
			return results, err
		}
		results = append(results, *result)
	}

	return results, nil
}

func parseCheckResponse(resp string) (*IsProxyResult, error) {
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return nil, fmt.Errorf("empty IsProxy response")
	}

	// Response format: "example.com available" or "example.com not available"
	var domain string
	var available bool

	switch {
	case strings.HasSuffix(resp, " not available"):
		domain = strings.TrimSuffix(resp, " not available")
	case strings.HasSuffix(resp, " available"):
		domain = strings.TrimSuffix(resp, " available")
		available = true
	default:
		return nil, fmt.Errorf("unexpected IsProxy response: %s", resp)
	}

	parts := strings.SplitN(domain, ".", 2)
	name := parts[0]
	tld := ""
	if len(parts) > 1 {
		tld = parts[1]
	}

	return &IsProxyResult{
		Domain:    name,
		TLD:       tld,
		Available: available,
	}, nil
}
