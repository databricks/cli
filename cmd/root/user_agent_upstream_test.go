package root

import (
	"context"
	"regexp"
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWithCommandInUserAgent(t *testing.T) {
	ctx := withCommandInUserAgent(context.Background(), &cobra.Command{Use: "foo"})

	// Check that the command name is in the user agent string.
	ua := useragent.FromContext(ctx)
	assert.Contains(t, ua, "cmd/foo")

	// Check that the command trace ID is in the user agent string.
	re := regexp.MustCompile(`command-trace-id/([a-f0-9-]+) `)
	matches := re.FindAllStringSubmatch(ua, -1)

	// Assert that we have exactly one match and that it's a valid UUID.
	require.Len(t, matches, 1)
	_, err := uuid.Parse(matches[0][1])
	assert.NoError(t, err)
}
