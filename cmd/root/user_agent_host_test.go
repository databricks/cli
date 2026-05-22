package root

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

// hostEnvKeys mirrors the env vars read by cmdio.DetectHost. Tests clear them
// so the developer's shell environment cannot bleed into assertions.
var hostEnvKeys = []string{
	"TERM_PROGRAM",
	"TERMINAL_EMULATOR",
}

func clearHostEnv(t *testing.T) {
	for _, k := range hostEnvKeys {
		t.Setenv(k, "")
	}
}

func TestHostInUserAgent_Unknown(t *testing.T) {
	clearHostEnv(t)
	ctx := withHostInUserAgent(t.Context())
	assert.Contains(t, useragent.FromContext(ctx), "host/unknown")
}

func TestHostInUserAgent_VSCode(t *testing.T) {
	clearHostEnv(t)
	t.Setenv("TERM_PROGRAM", "vscode")
	ctx := withHostInUserAgent(t.Context())
	assert.Contains(t, useragent.FromContext(ctx), "host/vscode")
}
