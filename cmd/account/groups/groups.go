// These SCIM commands were previously auto-generated from OpenAPI specs
// but are now manually maintained because the v2 API versions have taken
// over as the generated commands. See https://github.com/databricks/cli/pull/4722.

package groups

import (
	groups_v2 "github.com/databricks/cli/cmd/account/groups-v2"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Slices with functions to override the behavior of individual commands.
var listOverrides []func(*cobra.Command)

// The commands are the groups-v2 commands (same API) presented under their
// original names and help text.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: `Groups simplify identity management, making it easier to assign access to Databricks account, data, and other securable objects.`,
		Long: `Groups simplify identity management, making it easier to assign access to
  Databricks account, data, and other securable objects.

  It is best practice to assign access to workspaces and access-control policies
  in Unity Catalog to groups, instead of to users individually. All Databricks
  account identities can be assigned as members of groups, and members inherit
  permissions that are assigned to their group.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	v2 := groups_v2.New()
	for _, sub := range v2.Commands() {
		doc, ok := commandDocs[sub.Name()]
		if !ok {
			continue
		}
		v2.RemoveCommand(sub)
		sub.Use = doc.use
		if doc.short != "" {
			sub.Short = doc.short
			sub.Long = doc.long
		}
		sub.Hidden = doc.hidden
		if sub.Name() == "list" {
			for _, fn := range listOverrides {
				fn(sub)
			}
		}
		cmd.AddCommand(sub)
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

type commandDoc struct {
	use, short, long string
	hidden           bool
}

var commandDocs = map[string]commandDoc{
	"create": {
		use:   "create",
		short: `Create a new group.`,
		long: `Create a new group.

  Creates a group in the Databricks account with a unique name, using the
  supplied group details.`,
	},
	"delete": {
		use:   "delete ID",
		short: `Delete a group.`,
		long: `Delete a group.

  Deletes a group from the Databricks account.

  Arguments:
    ID: Unique ID for a group in the Databricks account.`,
	},
	"get": {
		use:   "get ID",
		short: `Get group details.`,
		long: `Get group details.

  Gets the information for a specific group in the Databricks account.

  Arguments:
    ID: Unique ID for a group in the Databricks account.`,
	},
	"list": {
		use:   "list",
		short: `List group details.`,
		long: `List group details.

  Gets all details of the groups associated with the Databricks account. As of
  08/22/2025, this endpoint will not return members. Instead, members should be
  retrieved by iterating through Get group details.`,
	},
	"patch": {
		use:   "patch ID",
		short: `Update group details.`,
		long: `Update group details.

  Partially updates the details of a group.

  Arguments:
    ID: Unique ID in the Databricks workspace.`,
	},
	"update": {
		use:   "update ID",
		short: `Replace a group.`,
		long: `Replace a group.

  Updates the details of a group by replacing the entire group entity.

  Arguments:
    ID: Databricks group ID`,
	},
}
