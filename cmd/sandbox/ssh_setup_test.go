package sandbox

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

func TestSSHSetupRequiresLiteralAlias(t *testing.T) {
	cmd := newSSHSetupCommand()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"id", "--name", "../escape"})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "SSH host alias")
	cmd = newSSHSetupCommand()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"id"})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "required flag")
}

func TestSSHSetupRequiresPersistentWorkspaceSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	cmd := newSSHSetupCommand()
	cmd.PreRunE = nil
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"id", "--name", "test"})
	require.ErrorContains(t, cmd.ExecuteContext(cmdctx.SetWorkspaceClient(t.Context(), w)), "requires a named Databricks profile")
}
