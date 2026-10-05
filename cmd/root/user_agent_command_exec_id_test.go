package root

import (
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestWithCommandExecIdInUserAgent(t *testing.T) {
	ctx := cmdctx.GenerateExecId(t.Context())
	ctx = withCommandExecIdInUserAgent(ctx)

	// user agent should contain cmd-exec-id/<shortid>
	ua := useragent.FromContext(ctx)
	assert.Regexp(t, `cmd-exec-id/[0-9A-Za-z]{7}( |$)`, ua)
}
