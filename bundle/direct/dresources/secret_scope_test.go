package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/require"
)

func TestSecretScopeDoReadMissingReturnsNotFound(t *testing.T) {
	_, client := setupTestServerClient(t)
	resource := (&ResourceSecretScope{}).New(client)

	remote, err := resource.DoRead(t.Context(), "missing-scope")

	require.Nil(t, remote)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}
