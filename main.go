package main

import (
	"github.com/databricks/cli/cmd"
	_ "github.com/databricks/cli/cmd/configure"
	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/cmd/root"
	_ "github.com/databricks/cli/cmd/sync"
	_ "github.com/databricks/cli/cmd/version"
)

func main() {
	root.Execute(cmd.New())
}
