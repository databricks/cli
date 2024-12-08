package root

import (
	"context"
	"regexp"
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithCommandExecIdInUserAgent(t *testing.T) {
	ctx := withCommandExecIdInUserAgent(context.Background())

	// Check that the command exec ID is in the user agent string.
	ua := useragent.FromContext(ctx)
	re := regexp.MustCompile(`cmd-exec-id/([a-f0-9-]+)`)
	matches := re.FindAllStringSubmatch(ua, -1)

	// Assert that we have exactly one match and that it's a valid UUID.
	require.Len(t, matches, 1)
	_, err := uuid.Parse(matches[0][1])
	assert.NoError(t, err)
}
