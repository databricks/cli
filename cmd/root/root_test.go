package root_test

import (
	"bufio"
	"bytes"
	"context"
	"testing"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestExecute_version(t *testing.T) {
	stderr, logger := createFakeLogger()
	stdout := bytes.Buffer{}
	ctx := cmdio.NewContext(context.Background(), logger)

	cli := cmd.New(ctx)
	cli.SetArgs([]string{"version"})
	cli.SetOut(&stdout)

	code := root.Execute(ctx, cli)

	assert.Equal(t, "", stderr.String())
	assert.Contains(t, stdout.String(), "Databricks CLI v")
	assert.Equal(t, 0, code)
}

func TestExecute_unknownCommand(t *testing.T) {
	stderr, logger := createFakeLogger()
	stdout := bytes.Buffer{}
	ctx := cmdio.NewContext(context.Background(), logger)

	cli := cmd.New(ctx)
	cli.SetOut(&stdout)
	cli.SetArgs([]string{"abcabcabc"})

	code := root.Execute(ctx, cli)

	assert.Equal(t, `Error: unknown command "abcabcabc" for "databricks"`+"\n", stderr.String())
	assert.Equal(t, "", stdout.String())
	assert.Equal(t, 1, code)
}

func createFakeLogger() (*bytes.Buffer, *cmdio.Logger) {
	out := bytes.Buffer{}

	return &out, &cmdio.Logger{
		Mode:   flags.ModeAppend,
		Reader: bufio.Reader{},
		Writer: &out,
	}
}
