package ezshare

import "time"

// Entry represents a file or directory on the EZ-Share device.
type Entry struct {
	Name      string
	IsDir     bool
	Timestamp time.Time
	Size      int64
	URL       string
}

// Version represents the firmware version information from the EZ-Share device.
type Version struct {
	ChipModel        string
	FirmwareVersion  string
	Date             string
	BuildNumber      string
	Raw              string
}
