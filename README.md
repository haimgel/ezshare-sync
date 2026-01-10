# ezshare-sync - EZ Share WiFi SD Card Sync Tool

[![Release](https://img.shields.io/github/release/haimgel/ezshare-sync.svg?style=flat)](https://github.com/haimgel/mqtt2cmd/releases/latest)
[![Software license](https://img.shields.io/github/license/haimgel/ezshare-sync.svg?style=flat)](/LICENSE)
[![Build status](https://img.shields.io/github/actions/workflow/status/haimgel/ezshare-sync/test.yaml?style=flat)](https://github.com/haimgel/mqtt2cmd/actions?workflow=release)
[![Hosted By: Cloudsmith](https://img.shields.io/badge/OSS%20hosting%20by-cloudsmith-blue?logo=cloudsmith&style=flat)](https://cloudsmith.com)

A tool for downloading files from EZ Share Wi-Fi enabled SD Cards. 

## What This Is

Many CPAP machines (ResMed, Philips Respironics, etc.) store detailed telemetry data on SD cards. This tool
automatically syncs that data from WiFi-enabled SD cards to your computer, making it easy to analyze your
sleep therapy data with tools like OSCAR.

**Works with**: Chinese EZ-Share WiFi SD card clones, as available on [AliExpress](https://www.aliexpress.com/item/1005005205172362.html).
**Tested Hardware**: LZ1801EDPG v1.0.0 firmware.

## Key Features

- Automatic sync of new files from SD card to local directory.
- Preserves directory structure and file timestamps.
- Skip already-downloaded files (by timestamp and size comparison).
- Optional SOCKS5 proxy support for remote access.
- Dry-run mode to preview what would be synced.
- Built-in retry logic for reliable transfers.

## Limitations

* The EZ Share Wi-Fi SD Card does not have a "real" API, so this tool is limited to a very specific set of
  operations, and it parses HTTP responses directly. This means we don't have access to detailed file metadata,
  the timestamps are imprecise, and the file sizes are rounded up to the nearest 1KB. This is "good enough" for
  CPAP data, but it might not be suitable for other use cases.
* Only downloading is supported. This SD-Card does not expose a write API.

## Quick Start

### Installation

**On macOS:**
```bash
# Add tap to your Homebrew
brew tap haimgel/tools

# Install it
brew install ezshare-sync
```

**On Windows:**

Download the `.zip` file from the [releases page](https://github.com/haimgel/ezshare-sync/releases), extract it, and add the directory to your PATH.

**On Linux:**

Option 1: Using package repositories
```bash
# Debian/Ubuntu - add Cloudsmith repository
curl -1sLf 'https://dl.cloudsmith.io/public/haimgel/public/setup.deb.sh' | sudo -E bash
sudo apt install ezshare-sync

# RedHat/Fedora/CentOS - add Cloudsmith repository
curl -1sLf 'https://dl.cloudsmith.io/public/haimgel/public/setup.rpm.sh' | sudo -E bash
sudo dnf install ezshare-sync

# Alpine - add Cloudsmith repository
curl -1sLf 'https://dl.cloudsmith.io/public/haimgel/public/setup.alpine.sh' | sudo -E sh
sudo apk add ezshare-sync
```

Option 2: Direct download
Download RPMs/DEBs/APKs from the [releases page](https://github.com/haimgel/ezshare-sync/releases).

### Basic Usage

```bash
# Sync all files from SD card to local directory
./ezshare-sync -target ~/cpap-data

# Preview what would be synced (dry run)
./ezshare-sync -target ~/cpap-data -dry-run
```

The tool will:
1. Connect to the SD card (default: http://192.168.4.1)
2. Recursively scan all directories
3. Download only new or modified files
4. Preserve directory structure and timestamps
5. Skip files that already exist with the same size

### Example Output

```
2026/01/06 10:15:23 Syncing from http://192.168.4.1 to /home/user/cpap-data
2026/01/06 10:15:24 Syncing /DATALOG/20260104/20260104_234139_CSL.edf
2026/01/06 10:15:25 Syncing /DATALOG/20260104/20260104_234156_BRP.edf
2026/01/06 10:15:26 Skipping /DATALOG/SETTINGS/STR.edf (already exists)
2026/01/06 10:15:27 Sync complete: 2 files synced, 1 skipped, 0 errors
```

## Go Library

This project also includes a Go library for programmatic access to EZ-Share WiFi SD cards. 
See [ezshare/README.md](ezshare/README.md) for the library documentation.

The EZ Share HTTP [API](API.md) is documented as well. Note that this is reverse-engineering
of the responses of the hardware that I have. Other cards might have a slightly different API.

## License

[APACHE 2.0](./LICENSE)
