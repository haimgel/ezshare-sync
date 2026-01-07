package ezshare

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type versionResponse struct {
	XMLName xml.Name      `xml:"response"`
	Device  versionDevice `xml:"device"`
}

type versionDevice struct {
	Version string `xml:"version"`
}

// GetVersion retrieves the firmware version information from the device.
func (c *Client) GetVersion(ctx context.Context) (*Version, error) {
	var version *Version
	err := c.retryOperation(ctx, func() error {
		result, err := c.getVersionAttempt(ctx)
		if err == nil {
			version = result
		}
		return err
	})
	return version, err
}

func (c *Client) getVersionAttempt(ctx context.Context) (*Version, error) {
	versionURL := c.buildURL("/client", "command", "version")

	req, err := http.NewRequest("GET", versionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("version request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var versionResp versionResponse
	if err := xml.Unmarshal(body, &versionResp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	if versionResp.Device.Version == "" {
		return nil, fmt.Errorf("%w: version string is empty", ErrInvalidResponse)
	}

	version, err := parseVersionString(versionResp.Device.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version string: %w", err)
	}

	return version, nil
}

func parseVersionString(versionStr string) (*Version, error) {
	parts := strings.Fields(versionStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: empty version string", ErrInvalidResponse)
	}

	components := strings.Split(parts[0], ":")
	if len(components) != 4 {
		return nil, fmt.Errorf("%w: expected 4 components, got %d", ErrInvalidResponse, len(components))
	}

	return &Version{
		ChipModel:       components[0],
		FirmwareVersion: components[1],
		Date:            components[2],
		BuildNumber:     components[3],
		Raw:             versionStr,
	}, nil
}
