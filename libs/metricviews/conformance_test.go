package metricviews

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func fixturePaths(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob("testdata/fixtures/*.yml")
	require.NoError(t, err)
	require.Len(t, paths, 47, "expected 47 fixtures")
	return paths
}

func TestFixturesCanonicalizeIdempotent(t *testing.T) {
	for _, p := range fixturePaths(t) {
		t.Run(filepath.Base(p), func(t *testing.T) {
			data, err := os.ReadFile(p)
			require.NoError(t, err, "fixture must parse")
			require.NoError(t, Validate(data))

			// Exercise the full canonicalization cycle the resource relies on for
			// drift detection (Parse -> Normalize -> Marshal). Re-parsing the result
			// and canonicalizing again must yield byte-identical output, otherwise a
			// no-op deploy would report persistent false drift.
			b1, err := canonicalize(data)
			require.NoError(t, err)
			b2, err := canonicalize(b1)
			require.NoError(t, err)
			assert.Equal(t, string(b1), string(b2), "canonicalization must be idempotent")
		})
	}
}

func canonicalize(data []byte) ([]byte, error) {
	m, err := Parse(data)
	if err != nil {
		return nil, err
	}
	m.Normalize()
	return yaml.Marshal(m)
}

func TestCanonicalizePreservesWindowOffset(t *testing.T) {
	spec := []byte(`version: "1.1"
source: samples.tpch.orders
measures:
  - name: previous_year_orders
    expr: COUNT(1)
    window:
      - order: order_date
        semiadditive: last
        range: current
        offset: -12 months
`)

	out, err := canonicalize(spec)
	require.NoError(t, err)
	assert.Contains(t, string(out), "offset: -12 months\n")
}

func TestFixturesStructdiffEqualToSelf(t *testing.T) {
	for _, p := range fixturePaths(t) {
		t.Run(filepath.Base(p), func(t *testing.T) {
			data, err := os.ReadFile(p)
			require.NoError(t, err)
			a, err := Parse(data)
			require.NoError(t, err)
			b, err := Parse(data)
			require.NoError(t, err)
			assert.True(t, structdiff.IsEqual(a, b), "a parsed view must equal itself under structdiff")
		})
	}
}
