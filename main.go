package main

import (
	"github.com/databricks/cli/cmd"
	_ "github.com/databricks/cli/cmd/account"
	_ "github.com/databricks/cli/cmd/api"
	_ "github.com/databricks/cli/cmd/auth"
	_ "github.com/databricks/cli/cmd/bundle"
	_ "github.com/databricks/cli/cmd/bundle/debug"
	_ "github.com/databricks/cli/cmd/configure"
	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/cmd/root"
	_ "github.com/databricks/cli/cmd/sync"
	_ "github.com/databricks/cli/cmd/version"
	_ "github.com/databricks/cli/cmd/workspace"
)

func main() {
	root.Execute(cmd.New())
}
