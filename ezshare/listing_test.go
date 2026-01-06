package ezshare

import (
	"strings"
	"testing"
	"time"
)

func TestParseDirectoryListing(t *testing.T) {
	// Real HTML response from the device
	html := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
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
   2026- 1- 4   10:56:12           1KB  <a href="http://192.168.4.1/download?file=IDNK8C~1.TGT"> Identification.tgt</a>
   2026- 1- 4   10:56:12         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG"> DATALOG</a>
   2026- 1- 5   12:10: 0          22KB  <a href="http://192.168.4.1/download?file=STR.EDF"> STR.edf</a>

Total Entries: 4
Total Size: 87KB
</pre>
</body>
</html>`

	entries, err := parseDirectoryListing(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseDirectoryListing failed: %v", err)
	}

	// Should have 4 entries (excluding . and ..)
	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	// Test first file entry
	if entries[0].Name != "Journal.dat" {
		t.Errorf("expected name 'Journal.dat', got '%s'", entries[0].Name)
	}
	if entries[0].IsDir {
		t.Error("expected IsDir to be false")
	}
	if entries[0].Size != 64*1024 {
		t.Errorf("expected size 65536 bytes, got %d", entries[0].Size)
	}

	// Test timestamp parsing
	expectedTime := time.Date(2026, 1, 4, 10, 55, 58, 0, time.UTC)
	if !entries[0].Timestamp.Equal(expectedTime) {
		t.Errorf("expected timestamp %v, got %v", expectedTime, entries[0].Timestamp)
	}

	// Test directory entry
	if entries[2].Name != "DATALOG" {
		t.Errorf("expected name 'DATALOG', got '%s'", entries[2].Name)
	}
	if !entries[2].IsDir {
		t.Error("expected IsDir to be true")
	}
	if entries[2].Size != 0 {
		t.Errorf("expected size 0 for directory, got %d", entries[2].Size)
	}

	// Test URL field contains the href from the HTML
	if entries[0].URL != "http://192.168.4.1/download?file=JOURNAL.DAT" {
		t.Errorf("expected URL 'http://192.168.4.1/download?file=JOURNAL.DAT', got '%s'", entries[0].URL)
	}
	if entries[2].URL != "dir?dir=A:%5CDATALOG" {
		t.Errorf("expected URL 'dir?dir=A:%%5CDATALOG', got '%s'", entries[2].URL)
	}
}

func TestParseDirectoryListing_Subdirectory(t *testing.T) {
	html := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=gb2312">
<title>Index of 20260104</title>
</head>
<body>
<h1><a href="photo">back to photo</a></h1>
<h1>Directory Index of 20260104</h1>
<pre>
   2026- 1- 4   12: 0: 2         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG%5C20260104"> .</a>
   2026- 1- 4   12: 0: 2         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG"> ..</a>
   2026- 1- 4   23:41:40           1KB  <a href="http://192.168.4.1/download?file=DATALOG%5C20260104%5C20CITZ~1.EDF"> 20260104_234139_CSL.edf</a>
   2026- 1- 5    5: 8:56        1918KB  <a href="http://192.168.4.1/download?file=DATALOG%5C20260104%5C20FL2G~1.EDF"> 20260104_234156_BRP.edf</a>

Total Entries: 4
Total Size: 1919KB
</pre>
</body>
</html>`

	entries, err := parseDirectoryListing(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseDirectoryListing failed: %v", err)
	}

	// Should exclude . and .. entries
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (excluding . and ..), got %d", len(entries))
	}

	// Test file in subdirectory
	if entries[0].Name != "20260104_234139_CSL.edf" {
		t.Errorf("expected name '20260104_234139_CSL.edf', got '%s'", entries[0].Name)
	}

	// Test URL field
	if entries[0].URL != "http://192.168.4.1/download?file=DATALOG%5C20260104%5C20CITZ~1.EDF" {
		t.Errorf("expected URL 'http://192.168.4.1/download?file=DATALOG%%5C20260104%%5C20CITZ~1.EDF', got '%s'", entries[0].URL)
	}

	// Test large file size
	expectedSize := int64(1918 * 1024)
	if entries[1].Size != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, entries[1].Size)
	}
}

func TestParseDirectoryListing_Empty(t *testing.T) {
	html := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=gb2312">
<title>Index of 20260105</title>
</head>
<body>
<h1><a href="photo">back to photo</a></h1>
<h1>Directory Index of 20260105</h1>
<pre>
   2026- 1- 5   12: 0: 0         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG%5C20260105"> .</a>
   2026- 1- 5   12: 0: 0         &lt;DIR&gt;   <a href="dir?dir=A:%5CDATALOG"> ..</a>

Total Entries: 2
Total Size: 0KB
</pre>
</body>
</html>`

	entries, err := parseDirectoryListing(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseDirectoryListing failed: %v", err)
	}

	// Should be empty (. and .. are excluded)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}
