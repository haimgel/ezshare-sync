package ezshare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var (
	timestampRegex  = regexp.MustCompile(`(\d{4})-\s*(\d{1,2})-\s*(\d{1,2})\s+(\d{1,2}):\s*(\d{1,2}):\s*(\d{1,2})`)
	sizeRegex       = regexp.MustCompile(`(\d+)KB|&lt;DIR&gt;|<DIR>`)
	whitespaceRegex = regexp.MustCompile(`\s+`)
)

type rawEntry struct {
	text string
	href string
	name string
}

// ListDirectory returns the contents of a directory on the device.
func (c *Client) ListDirectory(ctx context.Context, dirPath string) ([]*Entry, error) {
	var entries []*Entry
	err := c.retryOperation(ctx, func() error {
		result, err := c.listDirectoryAttempt(ctx, dirPath)
		if err == nil {
			entries = result
		}
		return err
	})
	return entries, err
}

func (c *Client) listDirectoryAttempt(ctx context.Context, dirPath string) ([]*Entry, error) {
	apiPath := convertUnixPathToAPI(dirPath)
	listURL := c.buildURL("/dir", "dir", apiPath)

	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("directory listing request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, dirPath)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	entries, err := parseDirectoryListing(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory listing: %w", err)
	}
	return entries, nil
}

func parseDirectoryListing(r io.Reader) ([]*Entry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	preNode := findPreTag(doc)
	if preNode == nil {
		return nil, fmt.Errorf("%w: <pre> tag not found", ErrInvalidResponse)
	}

	rawEntries := extractRawEntries(preNode)

	var entries []*Entry
	for _, raw := range rawEntries {
		entry, err := parseEntry(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse entry: %w", err)
		}
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func findPreTag(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "pre" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findPreTag(c); result != nil {
			return result
		}
	}
	return nil
}

func extractRawEntries(preNode *html.Node) []rawEntry {
	var entries []rawEntry
	var currentText strings.Builder

	for child := preNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			currentText.WriteString(child.Data)
		} else if child.Type == html.ElementNode && child.Data == "a" {
			var href, name string
			for _, attr := range child.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}
			if child.FirstChild != nil && child.FirstChild.Type == html.TextNode {
				name = strings.TrimSpace(child.FirstChild.Data)
			}

			if href != "" && name != "" {
				entries = append(entries, rawEntry{
					text: currentText.String(),
					href: href,
					name: name,
				})
				currentText.Reset()
			}
		}
	}

	return entries
}

func parseEntry(raw rawEntry) (*Entry, error) {
	text := strings.TrimSpace(raw.text)
	if text == "" || strings.HasPrefix(text, "Total") {
		return nil, nil
	}

	if raw.name == "." || raw.name == ".." {
		return nil, nil
	}

	timestampMatch := timestampRegex.FindStringSubmatch(text)
	if timestampMatch == nil {
		return nil, fmt.Errorf("timestamp not found in line: %q", text)
	}
	normalized := whitespaceRegex.ReplaceAllString(timestampMatch[0], " ")
	normalized = strings.ReplaceAll(normalized, "- ", "-")
	normalized = strings.ReplaceAll(normalized, ": ", ":")
	timestamp, err := time.Parse("2006-1-2 15:4:5", normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp %q: %w", timestampMatch[0], err)
	}

	sizeMatch := sizeRegex.FindStringSubmatch(text)
	if sizeMatch == nil {
		return nil, fmt.Errorf("size not found in line: %q", text)
	}
	isDir := false
	var size int64 = 0
	if strings.Contains(sizeMatch[0], "DIR") {
		isDir = true
	} else if sizeMatch[1] != "" {
		sizeKB, err := strconv.ParseInt(sizeMatch[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size %q: %w", sizeMatch[1], err)
		}
		size = sizeKB * 1024
	} else {
		return nil, fmt.Errorf("unexpected size format: %q", sizeMatch[0])
	}

	return &Entry{
		Name:      raw.name,
		IsDir:     isDir,
		Timestamp: timestamp,
		Size:      size,
		URL:       raw.href,
	}, nil
}
