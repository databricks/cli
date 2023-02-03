package main

import (
	"testing"

	"github.com/databricks/bricks/cmd/root"
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
	queue := []*cobra.Command{root.RootCmd}
	for len(queue) > 0 {
		cmd := queue[0]
		assert.NotContains(t, cmd.Name(), "_")
		queue = append(queue[1:], cmd.Commands()...)
	}
}
