package root

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestWithCommandExecIdInUserAgent(t *testing.T) {
	ctx := cmdctx.GenerateExecId(context.Background())
	ctx = withCommandExecIdInUserAgent(ctx)

	// user agent should contain cmd-exec-id/<UUID>
	ua := useragent.FromContext(ctx)
	assert.Regexp(t, `cmd-exec-id/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, ua)
}
