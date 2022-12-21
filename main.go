package main

import (
	_ "github.com/databricks/bricks/cmd/api"
	_ "github.com/databricks/bricks/cmd/bundle"
	_ "github.com/databricks/bricks/cmd/bundle/debug"
	_ "github.com/databricks/bricks/cmd/bundle/debug/deploy"
	_ "github.com/databricks/bricks/cmd/configure"
	_ "github.com/databricks/bricks/cmd/fs"
	_ "github.com/databricks/bricks/cmd/init"
	_ "github.com/databricks/bricks/cmd/launch"
	"github.com/databricks/bricks/cmd/root"
	_ "github.com/databricks/bricks/cmd/sync"
	_ "github.com/databricks/bricks/cmd/test"
)

func main() {
	root.Execute()
}
