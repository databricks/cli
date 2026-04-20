package browser

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDisabled(t *testing.T) {
	tests := []struct {
		browser  string
		expected bool
	}{
		{"", false},
		{"none", true},
		{"firefox", false},
	}

	for _, tt := range tests {
		t.Run(tt.browser, func(t *testing.T) {
			ctx := env.Set(t.Context(), "BROWSER", tt.browser)
			assert.Equal(t, tt.expected, IsDisabled(ctx))
		})
	}
}

func TestOpenWithBrowserNonePrintsURL(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, "BROWSER", "none")

	require.NoError(t, Open(ctx, "https://example.com/resource"))

	assert.Contains(t, stderr.String(), "Open this URL in your browser:")
	assert.Contains(t, stderr.String(), "https://example.com/resource")
}
