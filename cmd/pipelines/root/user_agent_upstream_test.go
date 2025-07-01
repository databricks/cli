// Copied from cmd/root/user_agent_upstream_test.go and adapted for pipelines use.
package root

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestUpstreamSet(t *testing.T) {
	t.Setenv(upstreamEnvVar, "foobar")
	ctx := withUpstreamInUserAgent(context.Background())
	assert.Contains(t, useragent.FromContext(ctx), "upstream/foobar")
}

func TestUpstreamSetEmpty(t *testing.T) {
	t.Setenv(upstreamEnvVar, "")
	ctx := withUpstreamInUserAgent(context.Background())
	assert.NotContains(t, useragent.FromContext(ctx), "upstream/")
}

func TestUpstreamVersionSet(t *testing.T) {
	t.Setenv(upstreamEnvVar, "foobar")
	t.Setenv(upstreamVersionEnvVar, "0.0.1")
	ctx := withUpstreamInUserAgent(context.Background())
	assert.Contains(t, useragent.FromContext(ctx), "upstream/foobar")
	assert.Contains(t, useragent.FromContext(ctx), "upstream-version/0.0.1")
}

func TestUpstreamVersionSetEmpty(t *testing.T) {
	t.Setenv(upstreamEnvVar, "foobar")
	t.Setenv(upstreamVersionEnvVar, "")
	ctx := withUpstreamInUserAgent(context.Background())
	assert.Contains(t, useragent.FromContext(ctx), "upstream/foobar")
	assert.NotContains(t, useragent.FromContext(ctx), "upstream-version/")
}

func TestUpstreamVersionSetUpstreamNotSet(t *testing.T) {
	t.Setenv(upstreamEnvVar, "")
	t.Setenv(upstreamVersionEnvVar, "0.0.1")
	ctx := withUpstreamInUserAgent(context.Background())
	assert.NotContains(t, useragent.FromContext(ctx), "upstream/")
	assert.NotContains(t, useragent.FromContext(ctx), "upstream-version/")
}
