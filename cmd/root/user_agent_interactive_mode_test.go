package root

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/useragent"
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

	if !strings.Contains(ua, "interactive/") {
		t.Errorf("expected user agent to contain 'interactive/', got %s", ua)
	}
}

func TestInteractiveModeNone(t *testing.T) {
	ctx := context.Background()
	// MockDiscard sets all TTY flags to false, so InteractiveMode returns "none".
	ctx = cmdio.MockDiscard(ctx)

	ctx = withInteractiveModeInUserAgent(ctx)
	ua := useragent.FromContext(ctx)

	if !strings.Contains(ua, "interactive/none") {
		t.Errorf("expected user agent to contain 'interactive/none', got %s", ua)
	}
}

func TestInteractiveModeNotSet(t *testing.T) {
	ctx := context.Background()
	// Don't initialize cmdio, so GetInteractiveMode returns ""

	ctx = withInteractiveModeInUserAgent(ctx)
	ua := useragent.FromContext(ctx)

	if strings.Contains(ua, "interactive/") {
		t.Errorf("expected user agent to not contain 'interactive/', got %s", ua)
	}
}
