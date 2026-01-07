# Development Workflow Guide

This document captures the workflow and principles used to develop this project. Use this as a reference for future work.

## Project Context

See [README.md](README.md) for project overview and [API.md](API.md) for the HTTP API specification.

## Development Methodology

### 1. Research Phase
- Started by researching existing GitHub repositories to understand the API
- Documented findings in markdown format
- Searched online for additional documentation and blog posts

### 2. Validation Against Real Hardware
- Access to actual EZ-Share device via SOCKS5 proxy (`usb-pve01:1080`)
- Tested all documented endpoints against real hardware
- Discovered documentation from repos was incorrect for the actual firmware version
- Updated all documentation based on real responses

### 3. Implementation Requirements

**Go Library Design:**
- Client struct with functional options pattern
- Context-first API design (all methods accept `context.Context`)
- Unix-style paths (`/` as root, not `A:`)
- SOCKS5 proxy support
- Retry logic with exponential backoff
- Both `DownloadFile` (save to file) and `GetFile` (return `io.ReadCloser`)

**Testing:**
- Test against real hardware responses
- Run tests after every significant change
- Use actual HTML/XML responses in test cases

## Code Style Preferences

### What to Include
- One-liner documentation for all public API elements
- Only comments that add non-obvious context
- Meaningful logical separations in code

### What to Remove
- Self-evident comments (e.g., `// Trim leading slash` before `strings.TrimPrefix()`)
- Unnecessary empty lines
- Step-by-step comments for obvious operations
- Redundant documentation files

### Example of Good vs Bad Comments

**Bad (self-evident):**
```go
// Trim leading slash
path = strings.TrimPrefix(path, "/")

// Convert forward slashes to backslashes
return strings.ReplaceAll(path, "/", "\\")
```

**Good (adds context):**
```go
// Client provides access to an EZ-Share WiFi SD card.
type Client struct { ... }

// WithSOCKS5Proxy configures the client to use a SOCKS5 proxy (e.g., "localhost:1080").
func WithSOCKS5Proxy(proxyAddr string) Option { ... }
```

## Testing Workflow

1. Run tests after every change: `go test ./ezshare/... -v`
2. Use real hardware responses in test cases
3. Ensure all tests pass before committing

## Commit Style

- Clear, descriptive commit messages
- Include bullet points for multiple changes
- Add Claude Code footer with co-authorship
- Example: See `git log` for established pattern

## Key Learnings

1. **Don't trust documentation**: Validate against real hardware
2. **Firmware variations exist**: LZ1001 vs LZ1801 have different APIs
3. **Variable-spaced timestamps**: Regex must handle spaces in unexpected places (`5: 8:56`)
4. **Path abstraction**: Internal conversion layer allows clean Unix-style API
5. **Keep it simple**: Prefer minimal, clean code over verbose documentation
