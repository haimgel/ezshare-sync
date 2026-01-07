package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/haimgel/ezshare-sync/ezshare"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func buildVersion(version, commit, date string) string {
	result := fmt.Sprintf("ez-share v%s", version)
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	result = fmt.Sprintf("%s\ngoos: %s\ngoarch: %s", result, runtime.GOOS, runtime.GOARCH)
	return result
}

func main() {
	var (
		baseURL      = flag.String("url", "http://192.168.4.1", "EZ-Share base URL")
		proxyAddr    = flag.String("proxy", "", "SOCKS5 proxy address (e.g., localhost:1080)")
		targetDir    = flag.String("target", "", "Target directory for sync (required)")
		dryRun       = flag.Bool("dry-run", false, "Preview what would be synced without actually doing it")
		printVersion = flag.Bool("version", false, "Print version information and exit")
	)
	flag.Parse()

	if *printVersion {
		fmt.Println(buildVersion(version, commit, date))
		return
	}

	if *targetDir == "" {
		log.Fatal("Error: --target flag is required")
	}

	var opts []ezshare.Option
	if *proxyAddr != "" {
		opts = append(opts, ezshare.WithSOCKS5Proxy(*proxyAddr))
	}

	client, err := ezshare.NewClient(*baseURL, opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	if *dryRun {
		log.Println("DRY RUN MODE - No files will be modified")
	}

	log.Printf("Syncing from %s to %s", *baseURL, *targetDir)

	stats := &syncStats{}
	if err := syncDirectory(ctx, client, "/", *targetDir, *dryRun, stats); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	log.Printf("Sync complete: %d files synced, %d skipped, %d errors",
		stats.synced, stats.skipped, stats.errors)

	if stats.errors > 0 {
		os.Exit(1)
	}
}

type syncStats struct {
	synced  int
	skipped int
	errors  int
}

func syncDirectory(ctx context.Context, client *ezshare.Client, remotePath, localBase string, dryRun bool, stats *syncStats) error {
	entries, err := client.ListDirectory(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("failed to list directory %s: %w", remotePath, err)
	}

	for _, entry := range entries {
		var fullRemotePath string
		if remotePath == "/" {
			fullRemotePath = "/" + entry.Name
		} else {
			fullRemotePath = remotePath + "/" + entry.Name
		}

		localPath := filepath.Join(localBase, filepath.FromSlash(fullRemotePath))

		if entry.IsDir {
			if !dryRun {
				if err := os.MkdirAll(localPath, 0755); err != nil {
					log.Printf("ERROR: Failed to create directory %s: %v", localPath, err)
					stats.errors++
					continue
				}
			}
			if err := syncDirectory(ctx, client, fullRemotePath, localBase, dryRun, stats); err != nil {
				log.Printf("ERROR: Failed to sync directory %s: %v", fullRemotePath, err)
				stats.errors++
			}
		} else {
			if err := syncFile(ctx, client, entry, fullRemotePath, localPath, dryRun, stats); err != nil {
				log.Printf("ERROR: Failed to sync file %s: %v", fullRemotePath, err)
				stats.errors++
			}
		}
	}

	return nil
}

func syncFile(ctx context.Context, client *ezshare.Client, entry *ezshare.Entry, remotePath, localPath string, dryRun bool, stats *syncStats) error {
	needsSync, reason := fileNeedsSync(entry, localPath)

	if !needsSync {
		stats.skipped++
		return nil
	}

	if dryRun {
		log.Printf("WOULD SYNC: %s (%s)", remotePath, reason)
		stats.synced++
		return nil
	}

	log.Printf("Syncing: %s (%s)", remotePath, reason)

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	tempPath := localPath + ".tmp"
	if err := client.DownloadFile(ctx, entry, tempPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to download: %w", err)
	}

	if err := os.Chtimes(tempPath, entry.Timestamp, entry.Timestamp); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to set timestamp: %w", err)
	}

	if err := os.Rename(tempPath, localPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	stats.synced++
	return nil
}

func fileNeedsSync(entry *ezshare.Entry, localPath string) (bool, string) {
	info, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		return true, "new file"
	}
	if err != nil {
		return true, fmt.Sprintf("stat error: %v", err)
	}

	// The API returns sizes rounded up to KB (base-2: 1024 bytes)
	// Check if local file rounds to the same KB value as remote
	localSizeKB := (info.Size() + 1023) / 1024
	remoteSizeKB := (entry.Size + 1023) / 1024
	if localSizeKB != remoteSizeKB {
		return true, "size mismatch"
	}

	timeDiff := info.ModTime().Sub(entry.Timestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 10*time.Second {
		return true, "timestamp mismatch"
	}

	return false, ""
}
