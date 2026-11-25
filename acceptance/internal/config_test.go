package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnvMatrix(t *testing.T) {
	tests := []struct {
		name      string
		matrix    map[string][]string
		exclude   map[string][]string
		extraVars []string
		expected  [][]string
	}{
		{
			name:     "empty matrix",
			matrix:   map[string][]string{},
			expected: [][]string{{}},
		},
		{
			name: "single key with single value",
			matrix: map[string][]string{
				"KEY": {"VALUE"},
			},
			expected: [][]string{
				{"KEY=VALUE"},
			},
		},
		{
			name: "single key with multiple values",
			matrix: map[string][]string{
				"KEY": {"A", "B"},
			},
			expected: [][]string{
				{"KEY=A"},
				{"KEY=B"},
			},
		},
		{
			name: "multiple keys with single values",
			matrix: map[string][]string{
				"KEY1": {"VALUE1"},
				"KEY2": {"VALUE2"},
			},
			expected: [][]string{
				{"KEY1=VALUE1", "KEY2=VALUE2"},
			},
		},
		{
			name: "multiple keys with multiple values",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {"C", "D"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY2=C"},
				{"KEY1=A", "KEY2=D"},
				{"KEY1=B", "KEY2=C"},
				{"KEY1=B", "KEY2=D"},
			},
		},
		{
			name: "keys with empty values are filtered out",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {},
				"KEY3": {"C"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY3=C"},
				{"KEY1=B", "KEY3=C"},
			},
		},
		{
			name: "all keys with empty values",
			matrix: map[string][]string{
				"KEY1": {},
				"KEY2": {},
			},
			expected: [][]string{{}},
		},
		{
			name: "example from documentation",
			matrix: map[string][]string{
				"KEY":   {"A", "B"},
				"OTHER": {"VALUE"},
			},
			expected: [][]string{
				{"KEY=A", "OTHER=VALUE"},
				{"KEY=B", "OTHER=VALUE"},
			},
		},
		{
			name: "exclude single combination",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {"C", "D"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=C"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY2=D"},
				{"KEY1=B", "KEY2=C"},
				{"KEY1=B", "KEY2=D"},
			},
		},
		{
			name: "exclude multiple combinations",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {"C", "D"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=C"},
				"rule2": {"KEY1=B", "KEY2=D"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY2=D"},
				{"KEY1=B", "KEY2=C"},
			},
		},
		{
			name: "exclude with dms and readplan example",
			matrix: map[string][]string{
				"DMS":      {"", "true"},
				"READPLAN": {"0", "1"},
			},
			exclude: map[string][]string{
				"nodmsplan": {"READPLAN=1", "DMS=true"},
			},
			expected: [][]string{
				{"DMS=", "READPLAN=0"},
				{"DMS=", "READPLAN=1"},
				{"DMS=true", "READPLAN=0"},
			},
		},
		{
			name: "exclude rule with subset of keys matches",
			matrix: map[string][]string{
				"KEY1": {"A"},
				"KEY2": {"B"},
				"KEY3": {"C"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=B"},
			},
			expected: nil,
		},
		{
			name: "exclude rule with more keys than envset does not match",
			matrix: map[string][]string{
				"KEY1": {"A"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=B"},
			},
			expected: [][]string{
				{"KEY1=A"},
			},
		},
		{
			name: "extraVars used for exclusion matching but stripped from result",
			matrix: map[string][]string{
				"KEY": {"A", "B"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY=A", "CONFIG_Cloud=true"},
			},
			extraVars: []string{"CONFIG_Cloud=true"},
			expected: [][]string{
				{"KEY=B"},
			},
		},
		{
			name: "extraVars not matching exclusion rule",
			matrix: map[string][]string{
				"KEY": {"A", "B"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY=A", "CONFIG_Cloud=true"},
			},
			extraVars: nil,
			expected: [][]string{
				{"KEY=A"},
				{"KEY=B"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvMatrix(tt.matrix, tt.exclude, tt.extraVars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubsetExpanded(t *testing.T) {
	expanded := [][]string{
		{"DMS=", "READPLAN="},
		{"DMS=true", "READPLAN="},
		{"DMS=true", "READPLAN=1"},
	}
	result := SubsetExpanded(expanded, "test/dir")
	require.Len(t, result, 1)
	assert.Contains(t, expanded, result[0])
	// Consistent for the same dir.
	assert.Equal(t, result, SubsetExpanded(expanded, "test/dir"))

	// Every variant gets picked for some dir.
	seen := make(map[string]bool)
	for i := range 100 {
		r := SubsetExpanded(expanded, fmt.Sprintf("test/dir%d", i))
		seen[strings.Join(r[0], ",")] = true
	}
	assert.Len(t, seen, len(expanded))
}

func TestLoadConfigPhaseIsNotInherited(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		dir        string
		wantPhase  int
		wantConfig string
	}{
		{
			name: "missing leaf config defaults to zero",
			files: map[string]string{
				"test.toml": "Phase = 3\n",
			},
			dir:        "suite/test",
			wantPhase:  0,
			wantConfig: "test.toml",
		},
		{
			name: "leaf config without phase resets inherited value",
			files: map[string]string{
				"test.toml":            "Phase = 3\n",
				"suite/test/test.toml": "Cloud = false\n",
			},
			dir:        "suite/test",
			wantPhase:  0,
			wantConfig: "test.toml, suite/test/test.toml",
		},
		{
			name: "leaf phase wins",
			files: map[string]string{
				"test.toml":            "Phase = 3\n",
				"suite/test/test.toml": "Cloud = false\nPhase = 1\n",
			},
			dir:        "suite/test",
			wantPhase:  1,
			wantConfig: "test.toml, suite/test/test.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)

			for path, contents := range tt.files {
				absPath := filepath.Join(root, filepath.FromSlash(path))
				require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
				require.NoError(t, os.WriteFile(absPath, []byte(contents), 0o644))
			}

			config, configPath := LoadConfig(t, tt.dir)

			assert.Equal(t, tt.wantPhase, config.Phase)
			assert.Equal(t, tt.wantConfig, configPath)
		})
	}
}

func TestLoadConfigEnvMatrixFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	files := map[string]string{
		"suite/test.toml":       "EnvMatrixFiles.VALUE = \"values/*.txt\"\n",
		"suite/values/b.txt":    "",
		"suite/values/a.txt":    "",
		"suite/values/c.txt":    "",
		"suite/values/d.md":     "",
		"suite/one/script":      "",
		"suite/two/script":      "",
		"suite/three/script":    "",
		"suite/three/test.toml": "EnvMatrix.VALUE = [\"x\"]\n",
		"suite/values/b.txt-exclude.toml": `
[[Exclude]]
Tests = ["one"]
Reason = "b does not apply to one"

[[Exclude]]
When = ["CONFIG_Cloud=true"]
Reason = "b is local-only"
`,
	}
	for path, contents := range files {
		absPath := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
		require.NoError(t, os.WriteFile(absPath, []byte(contents), 0o644))
	}

	assert.Equal(t, []string{"VALUE=a.txt", "VALUE=c.txt"}, loadVariants(t, "suite/one"))
	assert.Equal(t, []string{"VALUE=a.txt", "VALUE=b.txt", "VALUE=c.txt"}, loadVariants(t, "suite/two"))
	assert.Equal(t, []string{"VALUE=a.txt", "VALUE=c.txt"}, loadVariants(t, "suite/two", "CONFIG_Cloud=true"))
	assert.Equal(t, []string{"VALUE=x"}, loadVariants(t, "suite/three"))

	// Exclusions without conditions apply to the materialized config, as EnvMatrixExclude does.
	config, _ := LoadConfig(t, "suite/one")
	assert.Equal(t, "EnvMatrix.VALUE = [\"a.txt\", \"c.txt\"]\n", GenerateMaterializedConfig(&config))
}

func TestExpandEnvMatrixFilesSortsValuesByName(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"b/a.txt": "",
		"a/c.txt": "",
	})

	config := TestConfig{EnvMatrixFiles: map[string]string{"VALUE": "*/*.txt"}}
	require.NoError(t, expandEnvMatrixFiles(&config, filepath.Join(root, ConfigFilename), root))
	assert.Equal(t, []string{"a.txt", "c.txt"}, config.EnvMatrix["VALUE"])
}

func TestExpandEnvMatrixFilesErrors(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		envMatrix map[string][]string
		pattern   string
		wantErr   string
	}{
		{
			name:    "glob matches nothing",
			pattern: "values/*.txt",
			wantErr: `EnvMatrixFiles.VALUE = "values/*.txt" matches no files`,
		},
		{
			name:      "key set in both EnvMatrix and EnvMatrixFiles",
			files:     map[string]string{"values/a.txt": ""},
			envMatrix: map[string][]string{"VALUE": {"x"}},
			wantErr:   "VALUE is set in both EnvMatrix and EnvMatrixFiles",
		},
		{
			name: "duplicate base name",
			files: map[string]string{
				"one/a.txt": "",
				"two/a.txt": "",
			},
			pattern: "*/a.txt",
			wantErr: `matches several files named "a.txt"`,
		},
		{
			name: "sidecar matched as a value",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]]\nReason = \"r\"\n",
			},
			pattern: "values/*",
			wantErr: "matches sidecar",
		},
		{
			name: "orphan sidecar",
			files: map[string]string{
				"values/a.txt":              "",
				"values/b.txt-exclude.toml": "[[Exclude]]\nReason = \"r\"\n",
			},
			wantErr: "values/b.txt-exclude.toml has no matching file",
		},
		{
			name: "malformed sidecar",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]\n",
			},
			wantErr: "failed to parse",
		},
		{
			name: "unknown key in sidecar",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]]\nReason = \"r\"\nTest = [\"one\"]\n",
			},
			wantErr: "undecoded keys",
		},
		{
			name: "empty sidecar",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "# nothing\n",
			},
			wantErr: "has no [[Exclude]] entries",
		},
		{
			name: "missing Reason",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]]\nWhen = [\"DMS=true\"]\n",
			},
			wantErr: "Exclude[0] has no Reason",
		},
		{
			name: "blank Reason",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]]\nReason = \"r\"\n\n[[Exclude]]\nReason = \"  \"\n",
			},
			wantErr: "Exclude[1] has no Reason",
		},
		{
			name: "Tests entry that does not exist",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]]\nTests = [\"missing\"]\nReason = \"r\"\n",
			},
			wantErr: `names test "missing", which is not a directory`,
		},
		{
			name: "Tests entry that is a file",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]]\nTests = [\"values/a.txt\"]\nReason = \"r\"\n",
			},
			wantErr: `names test "values/a.txt", which is not a directory`,
		},
		{
			name: "malformed When",
			files: map[string]string{
				"values/a.txt":              "",
				"values/a.txt-exclude.toml": "[[Exclude]]\nWhen = [\"DMS\"]\nReason = \"r\"\n",
			},
			wantErr: `has condition "DMS", expected KEY=value`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, tt.files)

			pattern := tt.pattern
			if pattern == "" {
				pattern = "values/*.txt"
			}
			config := TestConfig{
				EnvMatrix:      tt.envMatrix,
				EnvMatrixFiles: map[string]string{"VALUE": pattern},
			}
			err := expandEnvMatrixFiles(&config, filepath.Join(root, ConfigFilename), root)
			require.Error(t, err)
			assert.Contains(t, filepath.ToSlash(err.Error()), tt.wantErr)
		})
	}
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	for path, contents := range files {
		absPath := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
		require.NoError(t, os.WriteFile(absPath, []byte(contents), 0o644))
	}
}

// loadVariants returns the variants the harness runs for dir, as their subtest names.
func loadVariants(t *testing.T, dir string, extraVars ...string) []string {
	config, _ := LoadConfig(t, dir)
	var result []string
	for _, envset := range ExpandEnvMatrix(config.EnvMatrix, config.EnvMatrixExclude, extraVars) {
		result = append(result, strings.Join(envset, "/"))
	}
	return result
}
