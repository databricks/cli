package metadata_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/databricks/cli/ucm/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	md := metadata.Metadata{
		Version:    metadata.Version,
		CliVersion: "0.123.0",
		Ucm: metadata.UcmMeta{
			Name:   "my-ucm",
			Target: "dev",
		},
		DeploymentID: "abc-123",
		Timestamp:    ts,
	}

	blob, err := json.Marshal(md)
	require.NoError(t, err)

	var got metadata.Metadata
	require.NoError(t, json.Unmarshal(blob, &got))
	assert.Equal(t, md, got)
}

func TestMetadataJSONFieldNames(t *testing.T) {
	md := metadata.Metadata{
		Version:    1,
		CliVersion: "v",
		Ucm: metadata.UcmMeta{
			Name:   "n",
			Target: "t",
		},
		DeploymentID: "d",
	}
	blob, err := json.Marshal(md)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(blob, &raw))
	for _, k := range []string{"version", "cli_version", "ucm", "deployment_id", "timestamp"} {
		_, ok := raw[k]
		assert.True(t, ok, "missing field %q in %v", k, raw)
	}

	ucm := raw["ucm"].(map[string]any)
	for _, k := range []string{"name", "target"} {
		_, ok := ucm[k]
		assert.True(t, ok, "missing field ucm.%q", k)
	}
}
