package main

import (
	_ "github.com/databricks/bricks/cmd/fs"
	_ "github.com/databricks/bricks/cmd/init"
	_ "github.com/databricks/bricks/cmd/launch"
	"github.com/databricks/bricks/cmd/root"
	_ "github.com/databricks/bricks/cmd/sync"
	_ "github.com/databricks/bricks/cmd/test"
)

func main() {
	root.RootCmd.Execute()
}
