package ezshare

import (
	"testing"
)

func TestConvertUnixPathToAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/", "A:"},
		{"", "A:"},
		{"/DATALOG", "A:\\DATALOG"},
		{"/DATALOG/20260104", "A:\\DATALOG\\20260104"},
		{"/DATALOG/20260104/file.edf", "A:\\DATALOG\\20260104\\file.edf"},
		{"DATALOG/20260104", "A:\\DATALOG\\20260104"},
		{"STR.EDF", "A:\\STR.EDF"},
		{"/STR.EDF", "A:\\STR.EDF"},
	}

	for _, tt := range tests {
		result := convertUnixPathToAPI(tt.input)
		if result != tt.expected {
			t.Errorf("convertUnixPathToAPI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
