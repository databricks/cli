package root

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/command"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestWithCommandExecIdInUserAgent(t *testing.T) {
	ctx := command.MockExecId(context.Background(), "some-exec-id")
	ctx = withCommandExecIdInUserAgent(ctx)

	// Check that the command exec ID is set in the user agent string.
	ua := useragent.FromContext(ctx)
	assert.Contains(t, ua, "cmd-exec-id/some-exec-id")
}
