package ezshare

import (
	"net/http"
	"time"
)

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client for the EZ-Share client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithSOCKS5Proxy configures the client to use a SOCKS5 proxy (e.g., "localhost:1080").
func WithSOCKS5Proxy(proxyAddr string) Option {
	return func(c *Client) {
		c.proxyAddr = proxyAddr
	}
}

// WithTimeout sets the HTTP request timeout for the client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithRetries sets the maximum number of retry attempts for failed requests.
func WithRetries(maxRetries int) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithUserAgent sets a custom User-Agent header for HTTP requests.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}
