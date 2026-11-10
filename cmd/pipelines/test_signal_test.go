//go:build linux || darwin

package pipelines

import (
	"syscall"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A Ctrl+C while Start Update is in flight stops the wait, but not the request: the service may
// have accepted the update, so the command must still receive and print its ID.
func TestInterruptDuringStartUpdateKeepsTheUpdateID(t *testing.T) {
	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/pipelines/{pipeline_id}/updates", func(req testserver.Request) any {
		require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))
		time.Sleep(200 * time.Millisecond)
		return map[string]string{"update_id": "u1"}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	waitCtx, stop := notifyTestInterrupt(t.Context())
	defer stop()
	updateID, err := startTestUpdate(t.Context(), w, "p1", newStartTestUpdateRequest(nil, nil))
	require.NoError(t, err)
	assert.Equal(t, "u1", updateID)

	select {
	case <-waitCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the interrupt did not stop the wait")
	}
}
