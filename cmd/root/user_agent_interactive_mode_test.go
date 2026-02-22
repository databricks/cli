package root

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestInteractiveModeWithCmdIO(t *testing.T) {
	ctx := context.Background()
	// Initialize cmdio with mock TTY capabilities
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText,
		io.NopCloser(strings.NewReader("")),
		cmdio.FakeTTY(io.Discard),
		cmdio.FakeTTY(io.Discard),
		"", ""))

	ctx = withInteractiveModeInUserAgent(ctx)
	ua := useragent.FromContext(ctx)

	assert.Contains(t, ua, "interactive/")
}

func TestInteractiveModeNone(t *testing.T) {
	ctx := context.Background()
	// MockDiscard sets all TTY flags to false, so InteractiveMode returns "none".
	ctx = cmdio.MockDiscard(ctx)

	ctx = withInteractiveModeInUserAgent(ctx)
	ua := useragent.FromContext(ctx)

	assert.Contains(t, ua, "interactive/none")
}
