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
	assert.Equal(t, "the Databricks Sandbox feature is not available in your region, or the service is temporarily unavailable", err.Error())
}

func TestAllow503RetryConsumesBudget(t *testing.T) {
	ctx := arm503Budget(t.Context())
	// max503Attempts-1 retries are allowed, then the budget is exhausted.
	for range max503Attempts - 1 {
		assert.True(t, allow503Retry(ctx))
	}
	assert.False(t, allow503Retry(ctx))
}

func TestAllow503RetryUnarmedContext(t *testing.T) {
	assert.False(t, allow503Retry(t.Context()))
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
