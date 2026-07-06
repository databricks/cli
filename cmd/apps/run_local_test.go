package apps

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/databricks/cli/libs/apps/runlocal"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSetupProxyPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{UserName: "test-user"}, nil)
	// setupProxy reads a token source off the config; the real command
	// always has a resolved config here.
	m.WorkspaceClient.Config = &config.Config{Host: "https://workspace.databricks.test", Token: "token"}

	cfg := runlocal.NewConfig("https://workspace.databricks.test", "123", t.TempDir(), runlocal.DEFAULT_HOST, runlocal.DEFAULT_PORT)
	err = setupProxy(t.Context(), &cobra.Command{}, cfg, m.WorkspaceClient, port, false)
	require.ErrorContains(t, err, "failed to start app proxy")
}

func TestSetupProxyPATOmitsTokenHeader(t *testing.T) {
	// A PAT config has no OAuth token source, so setupProxy must forward
	// requests without the OBO token header rather than 502 on every one.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, r.Header.Get(runlocal.HeaderForwardedAccessToken))
	}))
	defer backend.Close()

	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{UserName: "test-user"}, nil)
	m.WorkspaceClient.Config = &config.Config{Host: "https://workspace.databricks.test", Token: "dapi-token"}

	cfg := runlocal.NewConfig("https://workspace.databricks.test", "123", t.TempDir(), runlocal.DEFAULT_HOST, runlocal.DEFAULT_PORT)
	cfg.AppURL = backend.URL

	require.NoError(t, setupProxy(cmdio.MockDiscard(t.Context()), &cobra.Command{}, cfg, m.WorkspaceClient, port, false))

	resp, err := http.Get("http://" + net.JoinHostPort("localhost", strconv.Itoa(port)))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, string(body))
}

// TestAppHelperProcess is not a real test: TestKillAppProcess re-invokes the
// test binary with -test.run targeting it to get a long-running child process.
func TestAppHelperProcess(t *testing.T) {
	if os.Getenv("APPS_TEST_HELPER_PROCESS") != "1" {
		t.Skip("helper process for TestKillAppProcess")
	}
	time.Sleep(time.Minute)
}

func TestKillAppProcess(t *testing.T) {
	appCmd := exec.Command(os.Args[0], "-test.run=^TestAppHelperProcess$")
	appCmd.Env = append(os.Environ(), "APPS_TEST_HELPER_PROCESS=1")
	require.NoError(t, appCmd.Start())

	killAppProcess(appCmd)

	// A non-nil ProcessState proves the process was reaped; a non-success exit proves it was killed.
	require.NotNil(t, appCmd.ProcessState)
	require.False(t, appCmd.ProcessState.Success())
}
