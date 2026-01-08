package ezshare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const minResumableSize = 100 * 1024 // 100KB

// DownloadFile downloads a file from the device and saves it to the specified destination path.
func (c *Client) DownloadFile(ctx context.Context, entry *Entry, destPath string) error {
	return c.retryOperation(ctx, func() error {
		return c.downloadFileAttempt(ctx, entry, destPath)
	})
}

// DownloadFileByPath downloads a file by its Unix-style path. This is a convenience method
// that constructs the download URL directly. For files obtained from ListDirectory, use DownloadFile instead.
func (c *Client) DownloadFileByPath(ctx context.Context, filePath, destPath string) error {
	apiPath := convertUnixPathToAPI(filePath)
	downloadURL := c.buildURL("/download", "file", apiPath)

	entry := &Entry{
		Name: filePath,
		URL:  downloadURL,
	}

	return c.DownloadFile(ctx, entry, destPath)
}

// GetFile opens a file from the device and returns a ReadCloser for streaming the contents.
func (c *Client) GetFile(ctx context.Context, entry *Entry) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", entry.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}

	if resp.StatusCode == 404 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: %s", ErrNotFound, entry.Name)
	}
	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (c *Client) downloadFileAttempt(ctx context.Context, entry *Entry, destPath string) (err error) {
	if entry.Size < minResumableSize {
		return c.downloadFull(ctx, entry, destPath)
	}

	partialSize, shouldResume := validatePartialFile(destPath, entry.Size)
	if !shouldResume {
		_ = os.Remove(destPath)
		return c.downloadFull(ctx, entry, destPath)
	}

	if c.logger != nil {
		c.logger.Printf("Resuming download from byte %d/%d (%.1f%% complete): %s",
			partialSize, entry.Size, float64(partialSize)/float64(entry.Size)*100, entry.Name)
	}

	return c.downloadResume(ctx, entry, destPath, partialSize)
}

func (c *Client) downloadFull(ctx context.Context, entry *Entry, destPath string) error {
	reader, err := c.GetFile(ctx, entry)
	if err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		_ = reader.Close()
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	return c.downloadToFile(reader, out)
}

func (c *Client) downloadResume(ctx context.Context, entry *Entry, destPath string, partialSize int64) error {
	reader, err := c.getFileWithRange(ctx, entry, partialSize)
	if err != nil {
		return err
	}

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		_ = reader.Close()
		return fmt.Errorf("failed to open file for append: %w", err)
	}

	return c.downloadToFile(reader, out)
}

func (c *Client) downloadToFile(reader io.ReadCloser, out *os.File) (err error) {
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close reader: %w", closeErr)
		}
	}()
	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", closeErr)
		}
	}()

	if _, err = io.Copy(out, reader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func (c *Client) getFileWithRange(ctx context.Context, entry *Entry, byteOffset int64) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", entry.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", byteOffset))

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("range request failed: %w", err)
	}

	if resp.StatusCode == 206 {
		contentRange := resp.Header.Get("Content-Range")
		if contentRange == "" {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("206 response missing Content-Range header")
		}

		expectedPrefix := fmt.Sprintf("bytes %d-", byteOffset)
		if !strings.HasPrefix(contentRange, expectedPrefix) {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("unexpected Content-Range: %s (expected start at %d)", contentRange, byteOffset)
		}

		return resp.Body, nil
	}

	if resp.StatusCode == 200 {
		return resp.Body, nil
	}

	if resp.StatusCode == 404 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: %s", ErrNotFound, entry.Name)
	}

	_ = resp.Body.Close()
	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

func validatePartialFile(destPath string, expectedSize int64) (partialSize int64, shouldResume bool) {
	info, err := os.Stat(destPath)
	if err != nil {
		return 0, false
	}

	size := info.Size()
	if size == 0 || size >= expectedSize {
		return 0, false
	}

	return size, true
}
