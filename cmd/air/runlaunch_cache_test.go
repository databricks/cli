package aircmd

import (
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMeCountingServer returns a workspace whose SCIM Me calls are counted, and
// fails with status while status is non-zero.
func newMeCountingServer(t *testing.T) (*testserver.Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	// A fresh cache dir per test, so an entry from an earlier run or test can't serve Me.
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	server := testserver.New(t)
	t.Cleanup(server.Close)
	var calls, failStatus atomic.Int32
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(testserver.Request) any {
		calls.Add(1)
		if status := failStatus.Load(); status != 0 {
			return testserver.Response{StatusCode: int(status), Body: `{"error_code":"PERMISSION_DENIED","message":"denied"}`}
		}
		return iam.User{UserName: "me@example.com"}
	})
	testserver.AddDefaultHandlers(server)
	return server, &calls, &failStatus
}

func TestCachedCurrentUserEmailServesRepeatLaunchesFromCache(t *testing.T) {
	server, calls, _ := newMeCountingServer(t)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token-a"})
	require.NoError(t, err)

	for range 3 {
		email, err := cachedCurrentUserEmail(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, "me@example.com", email)
	}
	assert.Equal(t, int32(1), calls.Load(), "repeat launches with the same credential must not call Me")
}

// The cache key includes the Authorization header, so a different credential on the
// same host asks Me again rather than inheriting another identity's name.
func TestCachedCurrentUserEmailMissesForAnotherCredential(t *testing.T) {
	server, calls, _ := newMeCountingServer(t)
	wa, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token-a"})
	require.NoError(t, err)
	wb, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token-b"})
	require.NoError(t, err)

	_, err = cachedCurrentUserEmail(t.Context(), wa)
	require.NoError(t, err)
	_, err = cachedCurrentUserEmail(t.Context(), wb)
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

// A failed Me is an error, not an entry: the next launch asks again.
func TestCachedCurrentUserEmailDoesNotCacheFailures(t *testing.T) {
	server, calls, failStatus := newMeCountingServer(t)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token-c"})
	require.NoError(t, err)

	failStatus.Store(403)
	_, err = cachedCurrentUserEmail(t.Context(), w)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve current user")

	failStatus.Store(0)
	email, err := cachedCurrentUserEmail(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "me@example.com", email)
	assert.Equal(t, int32(2), calls.Load())
}

// DATABRICKS_CACHE_ENABLED=false is the opt-out and must restore one Me per launch.
func TestCachedCurrentUserEmailHonorsCacheDisabled(t *testing.T) {
	server, calls, _ := newMeCountingServer(t)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token-d"})
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	for range 2 {
		_, err := cachedCurrentUserEmail(t.Context(), w)
		require.NoError(t, err)
	}
	assert.Equal(t, int32(2), calls.Load())
}
