package mutator_test

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckMatch(t *testing.T, force bool) {
	type test struct {
		Fetched    string
		Configured string
	}

	tests := []test{
		{
			Fetched:    "main",
			Configured: "main",
		},
		{
			Fetched:    "main",
			Configured: "",
		},
		{
			Fetched:    "",
			Configured: "main",
		},
		{
			Fetched:    "",
			Configured: "main",
		},
	}

	for test := range tests {
		name := "CheckMatch " + test.Fetched + " " + test.Configured
		t.Run(name, func(t *testing.T) {
			diags := checkMatch("", test.Fetched, testConfigured, false)
			assert.Nil(t, diags)
		})
		t.Run(name+" force", func(t *testing.T) {
			diags := checkMatch("", test.Fetched, testConfigured, true)
			assert.Nil(t, diags)
		})
	}
}

func TestCheckWarning(t *testing.T, force bool) {
	diags := checkMatch("Git branch", "feature", "main", true)
	require.Len(t, diags, 1)
	assert.Equal(t, diags[0].Severity, diag.Warning)
	assert.Equal(t, diags[0].Summary, "not on the right Git branch:\n  expected according to configuration: main\n  actual: feature")
}

func TestCheckError(t *testing.T, force bool) {
	diags := checkMatch("Git branch", "feature", "main", false)
	require.Len(t, diags, 1)
	assert.Equal(t, diags[0].Severity, diag.Error)
	assert.Equal(t, diags[0].Summary, "not on the right Git branch:\n  expected according to configuration: main\n  actual: feature\nuse --force to override")
}
