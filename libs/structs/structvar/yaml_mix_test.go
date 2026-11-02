package structvar_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeFields decodes the file into a struct that has only the given fields and
// returns the names of the unknown fields reported. Anchors must not be reported.
func decodeFields[T any](t *testing.T, file string) []string {
	input, err := os.ReadFile(file)
	require.NoError(t, err)
	var dst T
	_, diags, err := structvar.DecodeYAML(file, bytes.NewBuffer(input), &dst)
	require.NoError(t, err)
	var out []string
	for _, d := range diags {
		out = append(out, d.Summary)
	}
	return out
}

// The old tests checked dyn.Value.IsAnchor. The decoder exposes anchors only through
// the "unknown field" warning, which is not reported for anchors.
func TestYAMLMix01(t *testing.T) {
	file := "testdata/yaml/mix_01.yml"
	loadYAML(t, file)

	type known struct{}
	warnings := decodeFields[known](t, file)
	assert.Contains(t, warnings, "unknown field: office_address")
	assert.NotContains(t, warnings, "unknown field: base_address")
}

func TestYAMLMix02(t *testing.T) {
	file := "testdata/yaml/mix_02.yml"
	loadYAML(t, file)

	type known struct{}
	warnings := decodeFields[known](t, file)
	assert.Contains(t, warnings, "unknown field: theme")
	assert.NotContains(t, warnings, "unknown field: base_colors")
}
