package sandbox

import (
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
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

func TestTranslateErrorRewrites503(t *testing.T) {
	orig := &apierr.APIError{StatusCode: http.StatusServiceUnavailable, Message: "Service Unavailable"}
	err := translateError(orig)
	require.Error(t, err)
	assert.Equal(t, "the Databricks Sandboxes feature is not available in your region", err.Error())
}

func TestTranslateErrorPassesThroughOthers(t *testing.T) {
	require.NoError(t, translateError(nil))

	notFound := &apierr.APIError{StatusCode: http.StatusNotFound, Message: "Sandbox not found"}
	assert.Equal(t, error(notFound), translateError(notFound))
}

func TestValidateNameCountsBytesNotRunes(t *testing.T) {
	// 64 panda emoji = 64 × 4 bytes = 256 bytes — at the limit, OK.
	require.NoError(t, validateName(strings.Repeat("🐼", 64)))
	// 65 = 260 bytes, rejected.
	err := validateName(strings.Repeat("🐼", 65))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "260 bytes")
}
