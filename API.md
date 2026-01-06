# EZ-Share WiFi SD Card HTTP API

**Firmware Tested**: LZ1801EDPG v1.0.0 (2016-03-19, Build 72)

This document describes the HTTP API as it actually exists on the hardware, including all quirks and peculiarities.

## Network Configuration

| Parameter | Default Value |
|-----------|---------------|
| SSID | `ez Share` |
| WiFi Password | `88888888` |
| Admin Password | `admin` |
| IP Address | `192.168.4.1` |
| DNS Name | `ezshare.card` |
| Character Encoding | `gb2312` (Chinese) |

## Endpoints

### GET /client?command=version

Returns firmware version information.

**Response Format**: XML with gb2312 encoding

**Response**:
```xml
<?xml version="1.0" encoding="gb2312"?>
<response>
<device>
<version>LZ1801EDPG:1.0.0:2016-03-19:72 LZ1801EDRS:1.0.0:2016-03-19:72 SPEED:-H:SPEED</version>
</device>
</response>
```

**Version String Format**: `{chip_model}:{firmware_version}:{date}:{build_number}`

**Other Commands**: All other `command` values (`getcfg`, `setcfg`, `status`, `info`, `help`, `list`, `mode`) return:
```xml
<?xml version="1.0" encoding="gb2312"?>
<response>
this is ezshare!</response>
```

---

### GET /dir?dir={path}

Lists files and directories in the specified path.

**Query Parameters**:
- `dir` - Directory path (URL-encoded, using backslashes)
  - Root: `A:`
  - Subdirectory: `A:%5CDATALOG` (backslash = `%5C`)
  - Nested: `A:%5CDATALOG%5C20260104`

**Response Format**: HTML with `<pre>` tag containing directory listing

**HTML Structure**:
```html
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=gb2312">
<title>Index of A:</title>
</head>
<body>
<h1><a href="photo">back to photo</a></h1>
<h1>Directory Index of A:</h1>
<pre>
   2026- 1- 4   10:55:58          64KB  <a href="http://192.168.4.1/download?file=JOURNAL.DAT"> Journal.dat</a>
   2026- 1- 4   10:56:12         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG"> DATALOG</a>
   2026- 1- 5   12:10: 0          22KB  <a href="http://192.168.4.1/download?file=STR.EDF"> STR.edf</a>

Total Entries: 7
Total Size: 88KB
</pre>
</body>
</html>
```

**Entry Format**: `{timestamp} {size} <a href="{url}"> {name}</a>`

**Timestamp Format**: `YYYY- M- D HH:MM:SS`
- Variable spacing (single-digit months/days have spaces instead of zero-padding)
- Space may appear after hour colon: `5: 8:56`
- Regex: `(\d{4})-\s*(\d{1,2})-\s*(\d{1,2})\s+(\d{1,2}):\s*(\d{1,2}):\s*(\d{1,2})`

**Size Format**:
- Files: `{number}KB` (e.g., `64KB`, `1918KB`)
- Directories: `&lt;DIR&gt;` (HTML entity)

**URL Format**:
- Files: Absolute URLs `http://192.168.4.1/download?file=FILENAME`
- Directories: Relative URLs `dir?dir=A:%5CSUBDIR`
- Current directory: `.` (filtered out by most clients)
- Parent directory: `..` (filtered out by most clients)

**Summary Footer**:
- `Total Entries: {count}`
- `Total Size: {size}KB` (files only, excludes directories)

---

### GET /download?file={path}

Downloads a file from the SD card.

**Query Parameters**:
- `file` - Full file path from root (URL-encoded, using backslashes)
  - Root file: `FILENAME.EXT`
  - Subdirectory: `DATALOG%5C20260104%5CFILE.EDF`

**Response**: Raw file content with headers

**Response Headers**:
```http
HTTP/1.1 200 OK
Last-Modified: Thu, 14 Oct 1990 05:46:10 GMT
Content-Type: text/plain
Content-Disposition: attachment; filename="str.edf"
Content-Length: 21586
Accept-Ranges: bytes
ETag: "5452"
```

**Header Details**:
- `Content-Type`: Inferred from file extension
- `Content-Disposition`: Filename in lowercase
- `Content-Length`: File size in bytes
- `Accept-Ranges: bytes`: **Supports HTTP range requests** (resume downloads)
- `Last-Modified`: File modification timestamp
- `ETag`: Entity tag for caching

**Error Responses**:
- `404 Not Found` - File doesn't exist
- `500+` - Server error (should retry)

---

### GET /photo

Web-based photo gallery interface.

**Query Parameters** (all optional):
- `vtype` - View type (0=English, 1=Chinese)
- `fdir` - Photo directory path
- `ftype` - File type (0=photos, 1=videos)
- `devw` - Device width for thumbnails
- `devh` - Device height for thumbnails
- `folderFlag` - Folder navigation flag

**Response**: Full HTML photo gallery with JavaScript

**Features**:
- Photo thumbnails and previews
- Video gallery: `/photo?vtype=0&fdir=&ftype=1`
- Configuration link: `/publicdir/index.htm`
- Disk list link: `/dir?dir=A:`
- Batch download (creates TAR archives)
- Search functionality
- Mobile/Classic view toggle

**Default Directory**: `A:\DCIM\`

---

### GET /publicdir/index.htm

Configuration interface (password protected).

**Authentication**: Requires admin password (default: `admin`)

**Response**: HTML login form with password prompt

**Configurable Settings**:
- WiFi SSID
- WiFi password
- Admin password
- Other device parameters

**Configuration Storage**:
- Settings stored in `ezshare.cfg` on SD card root
- Deleting `ezshare.cfg` resets to factory defaults
- Settings lost if SD card is formatted

---

### GET /publicdir/welcome.htm

Welcome/landing page with loading animation.

**Response**: HTML page with JavaScript progress bar

**Behavior**: Automatically redirects to photo gallery after animation

---

### GET /

Root endpoint.

**Response**: HTML redirect to welcome page
```html
<html>
<head><title>Success</title></head>
<body>
<form name="REDIRECTFORM" method="get" action="http://ezshare.card/publicdir/welcome.htm"></form>
<script language='javascript'>REDIRECTFORM.submit();</script>
</body>
</html>
```

---

## API Quirks and Peculiarities

### Path Format

- **DOS-style paths**: Uses `A:` drive letter (FAT32 filesystem)
- **Backslashes**: Must use `\` (not `/`) in paths
- **URL encoding**: Backslash encoded as `%5C`
- **Case insensitive**: FAT32 filesystem is case-insensitive

### Filename Handling

- **8.3 format**: Long filenames use DOS 8.3 format (e.g., `LONGFI~1.JPG`)
- **Display names**: HTML shows user-friendly names, hrefs use 8.3 format
- **Case variations**: Displayed filenames may have mixed case, actual names in uppercase

### Timestamp Peculiarities

- **Variable spacing**: Single-digit values use spaces instead of zeros
  - `2026- 1- 4` not `2026-01-04`
  - `5: 8:56` not `05:08:56`
- **Inconsistent padding**: Makes standard datetime parsing fail
- **Space after hour colon**: Sometimes appears: `5: 8:56`

### Character Encoding

- **gb2312**: All responses use Chinese GB2312/GBK encoding
- **HTML meta tag**: `<meta http-equiv="Content-Type" content="text/html; charset=gb2312">`
- **Chinese UI**: Interface elements contain Chinese characters
- **English option**: Available via `vtype=0` parameter

### HTML Response Format

- **Not JSON/XML**: Directory listings are HTML (except version endpoint)
- **HTML entities**: Directory marker is `&lt;DIR&gt;` not `<DIR>`
- **Absolute vs relative URLs**: Files use absolute URLs, directories use relative
- **No API consistency**: Different endpoints have different response formats

### Download Behavior

- **Range support**: Fully supports HTTP range requests (`Accept-Ranges: bytes`)
- **Resume capable**: Can resume interrupted downloads
- **No batch API**: Batch downloads only via web UI (creates TAR files)
- **No streaming API**: Cannot stream directory listings (must parse HTML)

### Error Handling

- **Minimal error info**: 404 for not found, 500+ for server errors
- **No error messages**: HTTP status codes only, no JSON error responses
- **Default responses**: Unknown commands return `this is ezshare!`

### Network Behavior

- **AP mode only**: Device only works as access point, not client mode
- **DHCP**: Provides DHCP in `/24` range
- **DNS name**: `ezshare.card` resolves to `192.168.4.1`
- **No HTTPS**: HTTP only, no encryption

### Limitations

- **Read-only**: No API for uploading or deleting files
- **No WebDAV**: Only HTTP GET endpoints
- **No file search**: Must traverse directory tree manually
- **No metadata API**: File attributes only via directory listing
- **Photo-centric**: Interface designed primarily for photos
- **DCIM restriction**: Web UI restricts access to photo directories (but API doesn't enforce)

## Response Samples

### Empty Directory
```html
<pre>
   2026- 1- 5   12: 0: 0         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG%5C20260105"> .</a>
   2026- 1- 5   12: 0: 0         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG"> ..</a>

Total Entries: 2
Total Size: 0KB
</pre>
```

### Subdirectory with Files
```html
<pre>
   2026- 1- 4   23:41:40           1KB  <a href="http://192.168.4.1/download?file=DATALOG%5C20260104%5C20CITZ~1.EDF"> 20260104_234139_CSL.edf</a>
   2026- 1- 5    5: 8:56        1918KB  <a href="http://192.168.4.1/download?file=DATALOG%5C20260104%5C20FL2G~1.EDF"> 20260104_234156_BRP.edf</a>

Total Entries: 2
Total Size: 1919KB
</pre>
```

## Firmware Variations

Different firmware versions may have different capabilities:

| Chip Model | Tested | Features | Notes |
|------------|--------|----------|-------|
| LZ1801EDPG | ✅ Yes | All above endpoints | This documentation |
| LZ1801EDRS | ❌ No | Unknown | Mentioned in version response |
| LZ1001EDPG | ❌ No | Different API | Community reports different format |

**Warning**: API format may differ significantly between firmware versions. Always check version first.

## Security Notes

- **Default credentials**: WiFi password `88888888` and admin password `admin` are factory defaults
- **No encryption**: All traffic is unencrypted HTTP
- **No authentication**: Most endpoints require no authentication
- **Change passwords**: Strongly recommended via configuration interface
- **Trusted networks only**: Use only in controlled environments
