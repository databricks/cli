package hostmetadata_test

import (
	"testing"

	"github.com/databricks/cli/libs/hostmetadata"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttach_SetsResolverOnConfig(t *testing.T) {
	ctx := t.Context()
	cfg := &config.Config{Host: "https://example.cloud.databricks.com"}
	require.Nil(t, cfg.HostMetadataResolver)

	hostmetadata.Attach(ctx, cfg)

	assert.NotNil(t, cfg.HostMetadataResolver)
}
