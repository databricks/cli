package internal

import (
	"fmt"
	"os"
	"path/filepath"
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
			name: "exclude with terraform and readplan example",
			matrix: map[string][]string{
				"DATABRICKS_BUNDLE_ENGINE": {"terraform", "direct"},
				"READPLAN":                 {"0", "1"},
			},
			exclude: map[string][]string{
				"noplantf": {"READPLAN=1", "DATABRICKS_BUNDLE_ENGINE=terraform"},
			},
			expected: [][]string{
				{"DATABRICKS_BUNDLE_ENGINE=terraform", "READPLAN=0"},
				{"DATABRICKS_BUNDLE_ENGINE=direct", "READPLAN=0"},
				{"DATABRICKS_BUNDLE_ENGINE=direct", "READPLAN=1"},
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

func TestSubsetEnvMatrix_SingleValues(t *testing.T) {
	// Single-value variables are kept as-is.
	matrix := map[string][]string{
		"KEY1": {"A"},
		"KEY2": {"B"},
	}
	result := SubsetEnvMatrix(matrix, "some/test", false)
	assert.Equal(t, map[string][]string{"KEY1": {"A"}, "KEY2": {"B"}}, result)
}

func TestSubsetEnvMatrix_EmptyMatrix(t *testing.T) {
	result := SubsetEnvMatrix(nil, "test", false)
	assert.Nil(t, result)
}

func TestSubsetEnvMatrix_NonEngineMultipleValues(t *testing.T) {
	// For non-engine variables with multiple values, exactly one is selected.
	matrix := map[string][]string{
		"FOO": {"a", "b", "c"},
	}
	result := SubsetEnvMatrix(matrix, "test/dir", false)
	require.Len(t, result["FOO"], 1)
	assert.Contains(t, []string{"a", "b", "c"}, result["FOO"][0])
}

func TestSubsetEnvMatrix_NonEngineDeterministic(t *testing.T) {
	// Same inputs produce same output.
	matrix := map[string][]string{
		"FOO": {"a", "b", "c"},
	}
	r1 := SubsetEnvMatrix(matrix, "test/dir", false)
	r2 := SubsetEnvMatrix(matrix, "test/dir", false)
	assert.Equal(t, r1, r2)
}

func TestSubsetEnvMatrix_NonEngineDifferentDirs(t *testing.T) {
	// Different test dirs may select different values (not guaranteed but likely with enough dirs).
	matrix := map[string][]string{
		"FOO": {"a", "b", "c", "d", "e"},
	}
	seen := map[string]bool{}
	for i := range 100 {
		dir := fmt.Sprintf("dir%d", i)
		r := SubsetEnvMatrix(matrix, dir, false)
		seen[r["FOO"][0]] = true
	}
	assert.Greater(t, len(seen), 1, "expected different dirs to select different values")
}

func TestSubsetEnvMatrix_EngineScriptUsesEngine(t *testing.T) {
	// When script uses $DATABRICKS_BUNDLE_ENGINE, both variants are kept.
	matrix := map[string][]string{
		"DATABRICKS_BUNDLE_ENGINE": {"terraform", "direct"},
	}
	result := SubsetEnvMatrix(matrix, "test/dir", true)
	assert.Equal(t, []string{"terraform", "direct"}, result["DATABRICKS_BUNDLE_ENGINE"])
}

func TestSubsetEnvMatrix_EngineScriptDoesNotUseEngine(t *testing.T) {
	// When script doesn't use $DATABRICKS_BUNDLE_ENGINE, exactly one variant is selected.
	matrix := map[string][]string{
		"DATABRICKS_BUNDLE_ENGINE": {"terraform", "direct"},
	}
	result := SubsetEnvMatrix(matrix, "test/dir", false)
	require.Len(t, result["DATABRICKS_BUNDLE_ENGINE"], 1)
}

func TestSubsetEnvMatrix_EngineDirectBias(t *testing.T) {
	// Across many test dirs, "direct" should be selected ~90% of the time.
	matrix := map[string][]string{
		"DATABRICKS_BUNDLE_ENGINE": {"terraform", "direct"},
	}
	directCount := 0
	total := 1000
	for i := range total {
		dir := fmt.Sprintf("test/dir%d", i)
		r := SubsetEnvMatrix(matrix, dir, false)
		if r["DATABRICKS_BUNDLE_ENGINE"][0] == "direct" {
			directCount++
		}
	}
	ratio := float64(directCount) / float64(total)
	assert.InDelta(t, 0.9, ratio, 0.05, "expected ~90%% direct, got %.1f%%", ratio*100)
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
				"suite/test/test.toml": "Local = true\n",
			},
			dir:        "suite/test",
			wantPhase:  0,
			wantConfig: "test.toml, suite/test/test.toml",
		},
		{
			name: "leaf phase wins",
			files: map[string]string{
				"test.toml":            "Phase = 3\n",
				"suite/test/test.toml": "Local = true\nPhase = 1\n",
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
