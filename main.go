package main

import (
	_ "github.com/databricks/bricks/cmd/api"
	_ "github.com/databricks/bricks/cmd/auth"
	_ "github.com/databricks/bricks/cmd/bundle"
	_ "github.com/databricks/bricks/cmd/bundle/debug"
	_ "github.com/databricks/bricks/cmd/configure"
	_ "github.com/databricks/bricks/cmd/fs"
	"github.com/databricks/bricks/cmd/root"
	_ "github.com/databricks/bricks/cmd/sync"
	_ "github.com/databricks/bricks/cmd/version"
)

func main() {
	root.Execute()
}
