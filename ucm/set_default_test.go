package ucm_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultSetsValueWhenAbsent(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  engine: direct
`))
	require.Empty(t, diags)
	u := &ucm.Ucm{Config: *cfg}

	ucm.SetDefault(t.Context(), u, "ucm.name", "example")
	assert.Equal(t, "example", u.Config.Ucm.Name)
}

func TestSetDefaultPreservesExistingValue(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: keep
`))
	require.Empty(t, diags)
	u := &ucm.Ucm{Config: *cfg}

	ucm.SetDefault(t.Context(), u, "ucm.name", "override")
	assert.Equal(t, "keep", u.Config.Ucm.Name)
}
