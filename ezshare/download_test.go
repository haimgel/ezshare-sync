package ezshare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestServer(t *testing.T, content string) (*httptest.Server, *Entry) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")

		if rangeHeader == "" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(content))
			return
		}

		var start, end int
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil {
			http.Error(w, "Invalid Range header", http.StatusBadRequest)
			return
		}

		if start >= len(content) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		end = len(content) - 1
		rangeContent := content[start:]

		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(rangeContent)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte(rangeContent))
	}))

	entry := &Entry{
		Name: "test.txt",
		URL:  server.URL + "/download?file=test.txt",
		Size: int64(len(content)),
	}

	return server, entry
}

func TestDownloadFile_SmallFile_NoResume(t *testing.T) {
	content := "Small file content"
	server, entry := setupTestServer(t, content)
	defer server.Close()

	entry.Size = int64(len(content))

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.txt")

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(downloaded), content)
	}
}

func TestDownloadFile_BelowThreshold_NoResume(t *testing.T) {
	content := strings.Repeat("x", 90*1024)
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "below.txt")

	partialSize := int64(len(content) / 2)
	if err := os.WriteFile(destPath, []byte(content[:partialSize]), 0644); err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != content {
		t.Errorf("Content mismatch: got length %d, want %d", len(downloaded), len(content))
	}
}

func TestDownloadFile_AboveThreshold_ResumesFromPartial(t *testing.T) {
	content := strings.Repeat("x", 110*1024)
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "above.txt")

	partialSize := int64(len(content) / 2)
	if err := os.WriteFile(destPath, []byte(content[:partialSize]), 0644); err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != content {
		t.Errorf("Content mismatch: got length %d, want %d", len(downloaded), len(content))
	}
}

func TestDownloadFile_LargeFile_FirstAttempt(t *testing.T) {
	content := strings.Repeat("Large file content. ", 100000)
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "large.txt")

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != content {
		t.Errorf("Content length mismatch: got %d, want %d", len(downloaded), len(content))
	}
}

func TestDownloadFile_ResumeFromPartial(t *testing.T) {
	content := strings.Repeat("Resumable content. ", 100000)
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "resume.txt")

	partialSize := int64(len(content) / 2)
	if err := os.WriteFile(destPath, []byte(content[:partialSize]), 0644); err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err != nil {
		t.Fatalf("Resume download failed: %v", err)
	}

	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != content {
		t.Errorf("Content mismatch after resume: got length %d, want %d", len(downloaded), len(content))
	}
}

func TestDownloadFile_CorruptedPartial_Restart(t *testing.T) {
	content := strings.Repeat("Content for corruption test. ", 100000)
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "corrupted.txt")

	corruptedSize := entry.Size + 1000
	if err := os.WriteFile(destPath, make([]byte, corruptedSize), 0644); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err != nil {
		t.Fatalf("Download after corruption failed: %v", err)
	}

	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != content {
		t.Errorf("Content mismatch: got length %d, want %d", len(downloaded), len(content))
	}
}

func TestDownloadFile_EmptyPartial_Restart(t *testing.T) {
	content := strings.Repeat("Content for empty test. ", 100000)
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "empty.txt")

	if err := os.WriteFile(destPath, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(downloaded) != content {
		t.Errorf("Content mismatch: got length %d, want %d", len(downloaded), len(content))
	}
}

func TestGetFileWithRange(t *testing.T) {
	content := strings.Repeat("Range request test content. ", 10000)
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	offset := int64(1000)
	reader, err := client.getFileWithRange(context.Background(), entry, offset)
	if err != nil {
		t.Fatalf("Range request failed: %v", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Errorf("Failed to close reader: %v", err)
		}
	}()

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	expected := content[offset:]
	if string(result) != expected {
		t.Errorf("Range content mismatch: got length %d, want %d", len(result), len(expected))
	}
}

func TestGetFileWithRange_InvalidOffset(t *testing.T) {
	content := "Short content for range test"
	server, entry := setupTestServer(t, content)
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	offset := int64(len(content) + 100)
	_, err = client.getFileWithRange(context.Background(), entry, offset)
	if err == nil {
		t.Error("Expected error for invalid offset, got nil")
	}
}

func TestValidatePartialFile(t *testing.T) {
	tests := []struct {
		name         string
		fileSize     int64
		expectedSize int64
		wantResume   bool
	}{
		{"No file", 0, 1000, false},
		{"Empty file", 0, 1000, false},
		{"Partial valid", 500, 1000, true},
		{"Equal size", 1000, 1000, false},
		{"Larger than expected", 1500, 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "test.txt")

			if tt.name != "No file" {
				content := make([]byte, tt.fileSize)
				if err := os.WriteFile(destPath, content, 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			partialSize, shouldResume := validatePartialFile(destPath, tt.expectedSize)

			if shouldResume != tt.wantResume {
				t.Errorf("shouldResume = %v, want %v", shouldResume, tt.wantResume)
			}

			if shouldResume && partialSize != tt.fileSize {
				t.Errorf("partialSize = %d, want %d", partialSize, tt.fileSize)
			}
		})
	}
}

func TestDownloadFile_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	entry := &Entry{
		Name: "error.txt",
		URL:  server.URL + "/download?file=error.txt",
		Size: 1000,
	}

	client, err := NewClient(server.URL, WithRetries(1))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "error.txt")

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
}

func TestDownloadFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	entry := &Entry{
		Name: "notfound.txt",
		URL:  server.URL + "/download?file=notfound.txt",
		Size: 1000,
	}

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "notfound.txt")

	err = client.DownloadFile(context.Background(), entry, destPath)
	if err == nil {
		t.Error("Expected error for 404, got nil")
	}
}
