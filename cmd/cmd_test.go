package cmd_test

import (
	"testing"

	"github.com/databricks/cli/cmd"
	"github.com/stretchr/testify/assert"
)

func TestRootCommandNamesAreUnique(t *testing.T) {
	cli := cmd.New(t.Context())

	seen := map[string]bool{}
	for _, sub := range cli.Commands() {
		assert.False(t, seen[sub.Name()], "duplicate top-level command %q", sub.Name())
		seen[sub.Name()] = true
	}
}
