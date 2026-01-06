# ezshare - Go Client Library for EZ-Share WiFi SD Cards

A Go client library for accessing EZ-Share WiFi SD cards via their HTTP API.

**Primary Use Case**: This library powers the [ezshare-sync](../cmd/ezshare-sync) tool for downloading CPAP
telemetry data. It can also be used for any application that needs to access files on WiFi-enabled SD cards.

## Features

- ✅ List directory contents with metadata (timestamp, size, type)
- ✅ Download files from the SD card
- ✅ Get firmware version information
- ✅ Support for SOCKS5 proxy
- ✅ Automatic retry logic with exponential backoff
- ✅ Context support for cancellation and timeouts
- ✅ Unix-style path notation (automatically converted to DOS format)
- ✅ Minimal dependencies
## Installation

```bash
go get github.com/haimgel/ezshare-sync/ezshare
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/haimgel/ezshare-sync/ezshare"
)

func main() {
    // Create client
    client, err := ezshare.NewClient("http://192.168.4.1")
    if err != nil {
        log.Fatal(err)
    }

    // List root directory
    entries, err := client.ListDirectory(context.Background(), "/")
    if err != nil {
        log.Fatal(err)
    }

    // Print entries
    for _, entry := range entries {
        if entry.IsDir {
            fmt.Printf("[DIR]  %s\n", entry.Name)
        } else {
            fmt.Printf("[FILE] %s (%d bytes)\n", entry.Name, entry.Size)
        }
    }
}
```

### Using SOCKS5 Proxy

```go
client, err := ezshare.NewClient(
    "http://192.168.4.1",
    ezshare.WithSOCKS5Proxy("localhost:1080"),
)
```

### Downloading Files

```go
// List directory first
entries, _ := client.ListDirectory(context.Background(), "/DATALOG")

// Download using Entry
for _, entry := range entries {
    if !entry.IsDir {
        client.DownloadFile(context.Background(), entry, "/tmp/" + entry.Name)
    }
}

// Or download by path directly
err := client.DownloadFileByPath(
    context.Background(),
    "/DATALOG/20260104/data.edf",
    "/tmp/data.edf",
)

// Or get an io.ReadCloser
entries, _ = client.ListDirectory(context.Background(), "/")
for _, entry := range entries {
    if entry.Name == "STR.EDF" {
        reader, _ := client.GetFile(context.Background(), entry)
        defer reader.Close()
        // Process the reader...
    }
}
```

### Getting Firmware Version

```go
version, err := client.GetVersion(context.Background())
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Chip: %s, Firmware: %s, Date: %s, Build: %s\n",
    version.ChipModel, version.FirmwareVersion, version.Date, version.BuildNumber)
// Output: Chip: LZ1801EDPG, Firmware: 1.0.0, Date: 2016-03-19, Build: 72
```

### Custom Configuration

```go
client, err := ezshare.NewClient(
    "http://192.168.4.1",
    ezshare.WithSOCKS5Proxy("localhost:1080"),
    ezshare.WithTimeout(60 * time.Second),
    ezshare.WithRetries(5),
    ezshare.WithUserAgent("my-app/1.0"),
)
```
