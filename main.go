package main

import (
	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
)

func main() {
	root.Execute(cmd.New())
}
