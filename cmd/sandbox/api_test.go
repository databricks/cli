package sandbox

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNameAcceptsAscii(t *testing.T) {
	require.NoError(t, validateName(""))
	require.NoError(t, validateName("my-project"))
	require.NoError(t, validateName(strings.Repeat("a", 256))) // boundary: exactly the limit
}

func TestValidateNameRejectsOversize(t *testing.T) {
	err := validateName(strings.Repeat("a", 257))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "257 bytes")
	assert.Contains(t, err.Error(), "256")
}

func TestValidateNameCountsBytesNotRunes(t *testing.T) {
	// 64 panda emoji = 64 × 4 bytes = 256 bytes — at the limit, OK.
	require.NoError(t, validateName(strings.Repeat("🐼", 64)))
	// 65 = 260 bytes, rejected.
	err := validateName(strings.Repeat("🐼", 65))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "260 bytes")
}

// In regions where the manager isn't deployed the gateway silently
// holds the connection open rather than returning a structured error,
// so the do wrapper must surface a timeout in user-language instead of
// letting the call hang.
func TestSandboxAPIDoTranslatesTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The SDK probes .well-known/databricks-config before the
		// first API call; let that 404 fast so the test exercises
		// the sandbox API path and not the SDK's host-metadata
		// fallback (which has its own 60s timeout).
		if strings.HasPrefix(r.URL.Path, "/.well-known/") {
			http.NotFound(w, r)
			return
		}
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: srv.URL, Token: "test-token"})
	require.NoError(t, err)
	c, err := client.New(w.Config)
	require.NoError(t, err)
	api := &sandboxAPI{c: c, timeout: 50 * time.Millisecond}

	_, err = api.get(t.Context(), "any-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox API timed out")
	assert.Contains(t, err.Error(), "this region may not have sandbox enabled")
}
