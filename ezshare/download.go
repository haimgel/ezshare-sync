package ezshare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

// DownloadFile downloads a file from the device and saves it to the specified destination path.
func (c *Client) DownloadFile(ctx context.Context, entry *Entry, destPath string) (err error) {
	reader, err := c.GetFile(ctx, entry)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close reader: %w", closeErr)
		}
	}()

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
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
