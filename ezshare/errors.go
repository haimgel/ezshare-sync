package ezshare

import "errors"

var (
	// ErrNotFound is returned when a file or directory does not exist on the device.
	ErrNotFound = errors.New("file or directory not found")
	// ErrInvalidResponse is returned when the device returns an unexpected response format.
	ErrInvalidResponse = errors.New("invalid response from device")
	// ErrServerError is returned when the device returns a 5xx HTTP status code.
	ErrServerError = errors.New("server error")
)
