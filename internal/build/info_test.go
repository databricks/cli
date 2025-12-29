package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDetails(t *testing.T) {
	GetInfo()
}

func TestGetSanitizedVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "version with plus",
			version:  "1.0.0+abc123",
			expected: "1.0.0-abc123",
		},
		{
			name:     "version with colon (Windows problematic)",
			version:  "1.0.0:dev",
			expected: "1.0.0-dev",
		},
		{
			name:     "version with forward slash (Windows problematic)",
			version:  "1.0.0/beta",
			expected: "1.0.0-beta",
		},
		{
			name:     "version with backslash (Windows problematic)",
			version:  "1.0.0\\test",
			expected: "1.0.0-test",
		},
		{
			name:     "version with multiple problematic characters",
			version:  "1.0.0+abc:123/test\\dev",
			expected: "1.0.0-abc-123-test-dev",
		},
		{
			name:     "clean version",
			version:  "1.0.0-dev",
			expected: "1.0.0-dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := Info{Version: tt.version}
			result := info.GetSanitizedVersion()
			assert.Equal(t, tt.expected, result)
		})
	}
}
