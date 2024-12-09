package mutator

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckMatch(t *testing.T) {
	type test struct {
		Fetched          string
		ConfiguredBefore string
		ConfiguredAfter  string
	}

	tests := []test{
		{
			Fetched:          "main",
			ConfiguredBefore: "main",
			ConfiguredAfter:  "main",
		},
		{
			Fetched:          "main",
			ConfiguredBefore: "",
			ConfiguredAfter:  "main",
		},
		{
			Fetched:          "",
			ConfiguredBefore: "main",
			ConfiguredAfter:  "main",
		},
		{
			Fetched:          "",
			ConfiguredBefore: "main",
			ConfiguredAfter:  "main",
		},
	}

	for _, test := range tests {
		name := "CheckMatch " + test.Fetched + " " + test.ConfiguredBefore + " " + test.ConfiguredAfter
		t.Run(name, func(t *testing.T) {
			configValue := test.ConfiguredBefore
			diags := checkMatch("", test.Fetched, &configValue, false)
			assert.Nil(t, diags)
			assert.Equal(t, test.ConfiguredAfter, configValue)
		})
		t.Run(name+" force", func(t *testing.T) {
			configValue := test.ConfiguredBefore
			diags := checkMatch("", test.Fetched, &configValue, true)
			assert.Nil(t, diags)
			assert.Equal(t, test.ConfiguredAfter, configValue)
		})
	}
}

func TestCheckWarning(t *testing.T) {
	configValue := "main"
	diags := checkMatch("Git branch", "feature", &configValue, true)
	require.Len(t, diags, 1)
	assert.Equal(t, diags[0].Severity, diag.Warning)
	assert.Equal(t, diags[0].Summary, "not on the right Git branch:\n  expected according to configuration: main\n  actual: feature")
	assert.Equal(t, "main", configValue)
}

func TestCheckError(t *testing.T) {
	configValue := "main"
	diags := checkMatch("Git branch", "feature", &configValue, false)
	require.Len(t, diags, 1)
	assert.Equal(t, diags[0].Severity, diag.Error)
	assert.Equal(t, diags[0].Summary, "not on the right Git branch:\n  expected according to configuration: main\n  actual: feature\nuse --force to override")
	assert.Equal(t, "main", configValue)
}
