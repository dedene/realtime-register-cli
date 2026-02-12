package api

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// IsProxy protocol endpoints per SPEC.md Appendix B:
// Connection: isapi.yoursrs.com:5443 (TLS)
// Protocol:
//   > AUTH <api-key>
//   < OK
//   > CHECK example com
//   < example.com AVAILABLE 9.95
//   > QUIT

const (
	IsProxyHost    = "isapi.yoursrs.com"
	IsProxyPort    = 5443
	IsProxyTimeout = 30 * time.Second
)

// IsProxyClient handles bulk domain availability checks via the IsProxy protocol.
type IsProxyClient struct {
	conn   net.Conn
	reader *bufio.Reader
	apiKey string
}

// IsProxyResult represents a single domain check result.
type IsProxyResult struct {
	Domain    string  `json:"domain"`
	TLD       string  `json:"tld"`
	Available bool    `json:"available"`
	Price     float64 `json:"price,omitempty"`
}

// NewIsProxyClient creates a new IsProxy client.
func NewIsProxyClient(apiKey string) *IsProxyClient {
	return &IsProxyClient{apiKey: apiKey}
}

// Connect establishes a TLS connection to the IsProxy server.
func (c *IsProxyClient) Connect() error {
	dialer := &net.Dialer{Timeout: IsProxyTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp",
		fmt.Sprintf("%s:%d", IsProxyHost, IsProxyPort),
		&tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		return fmt.Errorf("connect to IsProxy: %w", err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)

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

func (c *IsProxyClient) auth() error {
	if _, err := fmt.Fprintf(c.conn, "AUTH %s\r\n", c.apiKey); err != nil {
		return fmt.Errorf("send AUTH: %w", err)
	}

	resp, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read AUTH response: %w", err)
	}

	resp = strings.TrimSpace(resp)
	if resp != "OK" {
		return fmt.Errorf("IsProxy auth failed: %s", resp)
	}

	return nil
}

// Check checks a single domain availability.
func (c *IsProxyClient) Check(domain, tld string) (*IsProxyResult, error) {
	if _, err := fmt.Fprintf(c.conn, "CHECK %s %s\r\n", domain, tld); err != nil {
		return nil, fmt.Errorf("send CHECK: %w", err)
	}

	resp, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read CHECK response: %w", err)
	}

	return parseCheckResponse(resp)
}

// CheckMany checks multiple domains efficiently.
func (c *IsProxyClient) CheckMany(domains []string) ([]IsProxyResult, error) {
	results := make([]IsProxyResult, 0, len(domains))

	for _, full := range domains {
		parts := strings.SplitN(full, ".", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid domain format: %q (expected name.tld)", full)
		}
		domain, tld := parts[0], parts[1]

		result, err := c.Check(domain, tld)
		if err != nil {
			return results, err
		}
		results = append(results, *result)
	}

	return results, nil
}

func parseCheckResponse(resp string) (*IsProxyResult, error) {
	resp = strings.TrimSpace(resp)
	parts := strings.Fields(resp)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid IsProxy response: %s", resp)
	}

	fullDomain := parts[0]
	status := parts[1]

	domainParts := strings.SplitN(fullDomain, ".", 2)
	domain := domainParts[0]
	tld := ""
	if len(domainParts) > 1 {
		tld = domainParts[1]
	}

	result := &IsProxyResult{
		Domain:    domain,
		TLD:       tld,
		Available: status == "AVAILABLE",
	}

	if len(parts) >= 3 && result.Available {
		if price, err := strconv.ParseFloat(parts[2], 64); err == nil {
			result.Price = price
		}
	}

	return result, nil
}
