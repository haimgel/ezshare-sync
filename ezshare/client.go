package ezshare

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// Client provides access to an EZ-Share WiFi SD card.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	proxyAddr  string
	timeout    time.Duration
	maxRetries int
	userAgent  string
}

// NewClient creates a new EZ-Share client with the given base URL and options.
func NewClient(baseURL string, opts ...Option) (*Client, error) {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	c := &Client{
		baseURL:    parsedURL,
		timeout:    3 * time.Minute,
		maxRetries: 3,
		userAgent:  "ezshare-go/1.0",
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		transport := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		if c.proxyAddr != "" {
			dialer, err := proxy.SOCKS5("tcp", c.proxyAddr, nil, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
			}
			if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
				transport.DialContext = contextDialer.DialContext
			} else {
				transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				}
			}
		}

		c.httpClient = &http.Client{
			Transport: transport,
			Timeout:   c.timeout,
		}
	}
	return c, nil
}

func (c *Client) buildURL(path, paramName, paramValue string) string {
	u := *c.baseURL
	u.Path = path
	q := u.Query()
	q.Set(paramName, paramValue)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	reqWithCtx := req.Clone(ctx)
	if c.userAgent != "" {
		reqWithCtx.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(reqWithCtx)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: HTTP %d", ErrServerError, resp.StatusCode)
	}

	return resp, nil
}

func (c *Client) retryOperation(ctx context.Context, operation func() error) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		if !isRetriableError(err) {
			return err
		}

		lastErr = err
	}
	return fmt.Errorf("operation failed after %d retries: %w", c.maxRetries, lastErr)
}

func isRetriableError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, ErrServerError) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

func convertUnixPathToAPI(path string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "A:"
	}
	return "A:\\" + strings.ReplaceAll(path, "/", "\\")
}
