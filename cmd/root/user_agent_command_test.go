package root

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCommandString(t *testing.T) {
	root := &cobra.Command{
		Use: "root",
	}

	hello := &cobra.Command{
		Use: "hello",
	}

	world := &cobra.Command{
		Use: "world",
	}

	root.AddCommand(hello)
	hello.AddCommand(world)

	assert.Equal(t, "root", commandString(root))
	assert.Equal(t, "hello", commandString(hello))
	assert.Equal(t, "hello_world", commandString(world))
}
