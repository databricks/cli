package main

import (
	"context"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
)

func main() {
	root.Execute(cmd.New(context.Background()))
}
