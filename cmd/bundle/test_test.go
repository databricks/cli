package bundle

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/databricks/cli/libs/execv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubBundleTestProcess(t *testing.T, lookPath func(string) (string, error), execute func(execv.Options) error) {
	originalLookPath := bundleTestLookPath
	originalExecv := bundleTestExecv
	bundleTestLookPath = lookPath
	bundleTestExecv = execute
	t.Cleanup(func() {
		bundleTestLookPath = originalLookPath
		bundleTestExecv = originalExecv
	})
}

func executeTestCommand(t *testing.T, args ...string) error {
	root := &cobra.Command{Use: "databricks", SilenceErrors: true, SilenceUsage: true}
	bundle := &cobra.Command{Use: "bundle"}
	root.AddCommand(bundle)
	bundle.AddCommand(newTestCommand())
	root.SetArgs(append([]string{"bundle", "test"}, args...))
	return root.ExecuteContext(t.Context())
}

func TestBundleTestForwardsArgumentsUnchanged(t *testing.T) {
	t.Setenv("BUNDLETEST_ADAPTER_TEST", "present")

	var options execv.Options
	stubBundleTestProcess(t,
		func(name string) (string, error) {
			assert.Equal(t, "bundletest", name)
			return "/resolved/bundletest", nil
		},
		func(got execv.Options) error {
			options = got
			return nil
		},
	)

	err := executeTestCommand(t,
		"--local",
		"--changed",
		"--base", "origin/main",
		"-k", "transform_orders",
		"--",
		"tests/test_job.py::test_transform",
	)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"/resolved/bundletest",
		"--local",
		"--changed",
		"--base", "origin/main",
		"-k", "transform_orders",
		"--",
		"tests/test_job.py::test_transform",
	}, options.Args)
	assert.Contains(t, options.Env, "BUNDLETEST_ADAPTER_TEST=present")
	assert.Empty(t, options.Dir)
}

func TestBundleTestRunnerNotFound(t *testing.T) {
	stubBundleTestProcess(t,
		func(string) (string, error) {
			return "", exec.ErrNotFound
		},
		func(execv.Options) error {
			return errors.New("execv should not be called")
		},
	)

	err := executeTestCommand(t, "--local")
	require.Error(t, err)
	assert.ErrorIs(t, err, exec.ErrNotFound)
	assert.ErrorContains(t, err, "cannot find bundletest on PATH")
	assert.ErrorContains(t, err, "experimental/bundletest")
}

func TestBundleTestRunnerStartError(t *testing.T) {
	startErr := errors.New("start failed")
	stubBundleTestProcess(t,
		func(string) (string, error) {
			return "/resolved/bundletest", nil
		},
		func(execv.Options) error {
			return startErr
		},
	)

	err := executeTestCommand(t, "--local")
	require.Error(t, err)
	assert.ErrorIs(t, err, startErr)
	assert.ErrorContains(t, err, "failed to run bundletest")
}
