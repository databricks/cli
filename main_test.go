package main

import (
	"bufio"
	"bytes"
	"context"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"

	"github.com/databricks/cli/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCommandsDontUseUnderscoreInName(t *testing.T) {
	// We use underscore as separator between commands in logs
	// so need to enforce that no command uses it in its name.
	//
	// This test lives in the main package because this is where
	// all commands are imported.
	//
	queue := []*cobra.Command{cmd.New(context.Background())}
	for len(queue) > 0 {
		cmd := queue[0]
		assert.NotContains(t, cmd.Name(), "_")
		queue = append(queue[1:], cmd.Commands()...)
	}
}

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
