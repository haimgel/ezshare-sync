package ezshare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestConvertUnixPathToAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/", "A:"},
		{"", "A:"},
		{"/DATALOG", "A:\\DATALOG"},
		{"/DATALOG/20260104", "A:\\DATALOG\\20260104"},
		{"/DATALOG/20260104/file.edf", "A:\\DATALOG\\20260104\\file.edf"},
		{"DATALOG/20260104", "A:\\DATALOG\\20260104"},
		{"STR.EDF", "A:\\STR.EDF"},
		{"/STR.EDF", "A:\\STR.EDF"},
	}

	for _, tt := range tests {
		result := convertUnixPathToAPI(tt.input)
		if result != tt.expected {
			t.Errorf("convertUnixPathToAPI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsRetriableError(t *testing.T) {
	wrapRequest := func(err error) error {
		return fmt.Errorf("directory listing request failed: %w",
			&url.Error{Op: "Get", URL: "http://192.168.4.1/dir?dir=A%3A", Err: err})
	}
	// Shape of the error x/net's SOCKS dialer returns when the proxy cannot reach the card.
	socksUnreachable := &net.OpError{Op: "socks connect", Net: "tcp", Err: errors.New("unknown error host unreachable")}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"SOCKS host unreachable", wrapRequest(socksUnreachable), true},
		{"connection reset", wrapRequest(&net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}), true},
		{"connection closed before response", wrapRequest(io.EOF), true},
		{"body cut short", fmt.Errorf("failed to write file: %w", io.ErrUnexpectedEOF), true},
		{"server error", fmt.Errorf("%w: HTTP 503", ErrServerError), true},
		{"deadline exceeded", wrapRequest(context.DeadlineExceeded), true},
		{"canceled", wrapRequest(context.Canceled), false},
		{"not found", fmt.Errorf("%w: /STR.EDF", ErrNotFound), false},
		{"local file error", fmt.Errorf("failed to create destination file: %w", &os.PathError{Op: "open", Path: "/x", Err: syscall.ENOSPC}), false},
		{"unexpected status", errors.New("unexpected status code: 403"), false},
	}

	for _, tt := range tests {
		if got := isRetriableError(tt.err); got != tt.want {
			t.Errorf("%s: isRetriableError(%v) = %v, want %v", tt.name, tt.err, got, tt.want)
		}
	}
}

func TestRetryBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 500 * time.Millisecond},
		{2, time.Second},
		{6, 16 * time.Second},
		{7, 30 * time.Second},
		{DefaultMaxRetries, 30 * time.Second},
		{100, 30 * time.Second},
	}

	for _, tt := range tests {
		if got := retryBackoff(tt.attempt); got != tt.want {
			t.Errorf("retryBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestListDirectory_RetriesDroppedConnections(t *testing.T) {
	listing := `<html><body><pre>
   2026- 1- 5   12:10: 0          22KB  <a href="http://192.168.4.1/download?file=STR.EDF"> STR.edf</a>
</pre></body></html>`

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) <= 2 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}
			return
		}
		_, _ = w.Write([]byte(listing))
	}))
	defer server.Close()

	client := createTestClient(t, server.URL, WithRetries(2))
	entries, err := client.ListDirectory(context.Background(), "/")
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "STR.edf" {
		t.Errorf("unexpected entries: %+v", entries)
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("expected 3 requests, got %d", got)
	}
}
