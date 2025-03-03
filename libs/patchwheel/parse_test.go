package patchwheel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
