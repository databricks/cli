package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigFileNames_FindInPath(t *testing.T) {
	testCases := []struct {
		name     string
		files    []string
		expected string
		err      string
	}{
		{
			name:     "file found",
			files:    []string{"databricks.yml"},
			expected: "BASE/databricks.yml",
			err:      "",
		},
		{
			name:     "file found",
			files:    []string{"bundle.yml"},
			expected: "BASE/bundle.yml",
			err:      "",
		},
		{
			name:     "multiple files found",
			files:    []string{"databricks.yaml", "bundle.yml"},
			expected: "",
			err:      "multiple bundle root configuration files found",
		},
		{
			name:     "file not found",
			files:    []string{},
			expected: "",
			err:      "no such file or directory",
		},
	}

	if runtime.GOOS == "windows" {
		testCases[3].err = "The system cannot find the file specified."
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			for _, file := range tc.files {
				f1, _ := os.Create(filepath.Join(projectDir, file))
				f1.Close()
			}

			result, err := FileNames.FindInPath(projectDir)

			expected := strings.Replace(tc.expected, "BASE/", projectDir+string(os.PathSeparator), 1)
			assert.Equal(t, expected, result)

			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
