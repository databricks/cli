package root

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestWithCommandInUserAgent(t *testing.T) {
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

	assert.Equal(t, "root", CommandString(root))
	assert.Equal(t, "hello", CommandString(hello))
	assert.Equal(t, "hello_world", CommandString(world))

	ctx := WithCommandInUserAgent(context.Background(), world)

	ua := useragent.FromContext(ctx)
	assert.Contains(t, ua, "cmd/hello_world")
}
