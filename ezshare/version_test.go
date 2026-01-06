package ezshare

import (
	"testing"
)

func TestParseVersionString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "Valid version string from real hardware",
			input: "LZ1801EDPG:1.0.0:2016-03-19:72 LZ1801EDRS:1.0.0:2016-03-19:72 SPEED:-H:SPEED",
			want: &Version{
				ChipModel:       "LZ1801EDPG",
				FirmwareVersion: "1.0.0",
				Date:            "2016-03-19",
				BuildNumber:     "72",
				Raw:             "LZ1801EDPG:1.0.0:2016-03-19:72 LZ1801EDRS:1.0.0:2016-03-19:72 SPEED:-H:SPEED",
			},
			wantErr: false,
		},
		{
			name:  "Single version component",
			input: "LZ1001:2.0.1:2020-01-15:100",
			want: &Version{
				ChipModel:       "LZ1001",
				FirmwareVersion: "2.0.1",
				Date:            "2020-01-15",
				BuildNumber:     "100",
				Raw:             "LZ1001:2.0.1:2020-01-15:100",
			},
			wantErr: false,
		},
		{
			name:    "Empty string",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid format - too few components",
			input:   "LZ1801:1.0.0:2016-03-19",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid format - too many components",
			input:   "LZ1801:1.0.0:2016-03-19:72:extra",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersionString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersionString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.ChipModel != tt.want.ChipModel {
				t.Errorf("ChipModel = %v, want %v", got.ChipModel, tt.want.ChipModel)
			}
			if got.FirmwareVersion != tt.want.FirmwareVersion {
				t.Errorf("FirmwareVersion = %v, want %v", got.FirmwareVersion, tt.want.FirmwareVersion)
			}
			if got.Date != tt.want.Date {
				t.Errorf("Date = %v, want %v", got.Date, tt.want.Date)
			}
			if got.BuildNumber != tt.want.BuildNumber {
				t.Errorf("BuildNumber = %v, want %v", got.BuildNumber, tt.want.BuildNumber)
			}
			if got.Raw != tt.want.Raw {
				t.Errorf("Raw = %v, want %v", got.Raw, tt.want.Raw)
			}
		})
	}
}
