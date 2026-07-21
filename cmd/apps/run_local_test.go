package apps

import (
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/databricks/cli/libs/apps/runlocal"
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

	config := runlocal.NewConfig("https://workspace.databricks.test", "123", t.TempDir(), runlocal.DEFAULT_HOST, runlocal.DEFAULT_PORT)
	err = setupProxy(t.Context(), &cobra.Command{}, config, m.WorkspaceClient, port, false)
	require.ErrorContains(t, err, "failed to start app proxy")
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
