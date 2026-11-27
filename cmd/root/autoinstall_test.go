package root_test

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setAutoInstallRun replaces autoinstall.Run for the test. The variable is
// global, so tests that call it must not run in parallel.
func setAutoInstallRun(t *testing.T, run func(*cobra.Command) os.Signal) {
	prev := autoinstall.Run
	autoinstall.Run = run
	t.Cleanup(func() { autoinstall.Run = prev })
}

func newAutoInstallTestRoot(t *testing.T) *root.Tree {
	r := root.NewTree(t.Context())
	r.SetOut(io.Discard)
	r.SetErr(io.Discard)
	r.Add(root.CommandSpec{
		Use:   "jobs",
		Short: "jobs commands",
		Build: func(ctx context.Context, cmd *cobra.Command) {
			cmd.AddCommand(&cobra.Command{Use: "ok", RunE: func(*cobra.Command, []string) error { return nil }})
			cmd.AddCommand(&cobra.Command{Use: "fail", RunE: func(*cobra.Command, []string) error { return errors.New("failed") }})
		},
	})
	return r
}

func TestPostRunCallsAutoInstallWithTheCommand(t *testing.T) {
	var ran []string
	setAutoInstallRun(t, func(cmd *cobra.Command) os.Signal {
		ran = append(ran, cmd.CommandPath())
		return nil
	})

	err := newAutoInstallTestRoot(t).Execute(t.Context(), []string{"jobs", "ok"})
	require.NoError(t, err)
	assert.Equal(t, []string{"databricks jobs ok"}, ran)
}

func TestPostRunSkipsAutoInstallAfterAFailedCommand(t *testing.T) {
	var ran []string
	setAutoInstallRun(t, func(cmd *cobra.Command) os.Signal {
		ran = append(ran, cmd.CommandPath())
		return nil
	})

	err := newAutoInstallTestRoot(t).Execute(t.Context(), []string{"jobs", "fail"})
	require.Error(t, err)
	assert.Empty(t, ran)
}

func TestPostRunWithoutAutoInstall(t *testing.T) {
	setAutoInstallRun(t, nil)

	err := newAutoInstallTestRoot(t).Execute(t.Context(), []string{"jobs", "ok"})
	assert.NoError(t, err)
}
