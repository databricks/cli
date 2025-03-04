package patchwheel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCalculateNewVersion tests the CalculateNewVersion function.
func TestCalculateNewVersion(t *testing.T) {
	tests := []struct {
		name     string
		info     *WheelInfo
		mtime    time.Time
		expectedVersion  string
		expectedFilename string
	}{
		{
			name: "basic version",
			info: &WheelInfo{
				Distribution: "mypkg",
				Version:      "1.2.3",
				Tags:         []string{"py3", "none", "any"},
			},
			mtime:    time.Date(2025, 3, 4, 12, 34, 56, 0, time.UTC),
			expectedVersion:  "1.2.3+20250304123456",
			expectedFilename: "mypkg-1.2.3+20250304123456-py3-none-any.whl",
		},
		{
			name: "existing plus version",
			info: &WheelInfo{
				Distribution: "mypkg",
				Version:      "1.2.3+local",
				Tags:         []string{"py3", "none", "any"},
			},
			mtime:    time.Date(2025, 3, 4, 12, 34, 56, 0, time.UTC),
			expectedVersion:  "1.2.3+20250304123456",
			expectedFilename: "mypkg-1.2.3+20250304123456-py3-none-any.whl",
		},
		{
			name: "complex distribution name",
			info: &WheelInfo{
				Distribution: "my-pkg-name",
				Version:      "1.2.3",
				Tags:         []string{"py3", "none", "any"},
			},
			mtime:    time.Date(2025, 3, 4, 12, 34, 56, 0, time.UTC),
			expectedVersion:  "1.2.3+20250304123456",
			expectedFilename: "my-pkg-name-1.2.3+20250304123456-py3-none-any.whl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newVersion, newFilename := CalculateNewVersion(tt.info, tt.mtime)
			if newVersion != tt.expectedVersion {
				t.Errorf("expected version %s, got %s", tt.expectedVersion, newVersion)
			}
			if newFilename != tt.expectedFilename {
				t.Errorf("expected filename %s, got %s", tt.expectedFilename, newFilename)
			}
		})
	}
}

// TestParseWheelFilename tests the ParseWheelFilename function.
func TestParseWheelFilename(t *testing.T) {
	tests := []struct {
		filename         string
		wantDistribution string
		wantVersion      string
		wantTags         []string
		wantErr          bool
	}{
		{
			filename:         "myproj-0.1.0-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "myproj-0.1.0+20240303123456-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0+20240303123456",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "my-proj-with-hyphens-0.1.0-py3-none-any.whl",
			wantDistribution: "my-proj-with-hyphens",
			wantVersion:      "0.1.0",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "invalid-filename.txt",
			wantDistribution: "",
			wantVersion:      "",
			wantTags:         nil,
			wantErr:          true,
		},
		{
			filename:         "not-enough-parts-py3.whl",
			wantDistribution: "",
			wantVersion:      "",
			wantTags:         nil,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info, err := ParseWheelFilename(tt.filename)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantDistribution, info.Distribution)
				require.Equal(t, tt.wantVersion, info.Version)
				require.Equal(t, tt.wantTags, info.Tags)
			}
		})
	}
}
