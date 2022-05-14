package main

import (
	"github.com/databricks/bricks/cmd/root"
	_ "github.com/databricks/bricks/cmd/fs"
	_ "github.com/databricks/bricks/cmd/init"
	_ "github.com/databricks/bricks/cmd/launch"
	_ "github.com/databricks/bricks/cmd/test"
)

func main() {
	root.RootCmd.Execute()
}
