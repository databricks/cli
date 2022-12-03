package permissions

import (
	"github.com/databricks/bricks/cmd/permissions/permissions"
	workspace_assignment "github.com/databricks/bricks/cmd/permissions/workspace-assignment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "permissions",
}

func init() {

	Cmd.AddCommand(permissions.Cmd)
	Cmd.AddCommand(workspace_assignment.Cmd)
}
