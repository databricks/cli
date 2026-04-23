package mutator_test

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/require"
)

// loadUcm parses raw YAML and returns a fresh Ucm backed by it.
func loadUcm(t *testing.T, yaml string) *ucm.Ucm {
	t.Helper()
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(yaml))
	require.NoError(t, diags.Error())
	return &ucm.Ucm{Config: *cfg}
}

func summaries(ds diag.Diagnostics) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Summary)
	}
	return out
}
