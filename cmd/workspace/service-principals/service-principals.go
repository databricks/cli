// These SCIM commands were previously auto-generated from OpenAPI specs
// but are now manually maintained because the v2 API versions have taken
// over as the generated commands. See https://github.com/databricks/cli/pull/4722.

package service_principals

import (
	"github.com/databricks/cli/cmd/root"
	service_principals_v2 "github.com/databricks/cli/cmd/workspace/service-principals-v2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Slices with functions to override the behavior of individual commands.
var listOverrides []func(*cobra.Command)

// The commands are the service-principals-v2 commands (same API) presented under their
// original names and help text.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-principals",
		Short: `Identities for use with jobs, automated tools, and systems such as scripts, apps, and CI/CD platforms.`,
		Long: `Identities for use with jobs, automated tools, and systems such as scripts,
  apps, and CI/CD platforms. Databricks recommends creating service principals
  to run production jobs or modify production data. If all processes that act on
  production data run with service principals, interactive users do not need any
  write, delete, or modify privileges in production. This eliminates the risk of
  a user overwriting production data by accident.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	v2 := service_principals_v2.New()
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
			// These were regular flags of the original commands.
			for _, name := range []string{"count", "start-index"} {
				if f := sub.Flags().Lookup(name); f != nil {
					f.Hidden = false
				}
			}
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
		short: `Create a service principal.`,
		long: `Create a service principal.

  Creates a new service principal in the Databricks workspace.`,
	},
	"delete": {
		use:   "delete ID",
		short: `Delete a service principal.`,
		long: `Delete a service principal.

  Delete a single service principal in the Databricks workspace.

  Arguments:
    ID: Unique ID for a service principal in the Databricks workspace.`,
	},
	"get": {
		use:   "get ID",
		short: `Get service principal details.`,
		long: `Get service principal details.

  Gets the details for a single service principal define in the Databricks
  workspace.

  Arguments:
    ID: Unique ID for a service principal in the Databricks workspace.`,
	},
	"list": {
		use:   "list",
		short: `List service principals.`,
		long: `List service principals.

  Gets the set of service principals associated with a Databricks workspace.`,
	},
	"patch": {
		use:   "patch ID",
		short: `Update service principal details.`,
		long: `Update service principal details.

  Partially updates the details of a single service principal in the Databricks
  workspace.

  Arguments:
    ID: Unique ID in the Databricks workspace.`,
	},
	"update": {
		use:   "update ID",
		short: `Replace service principal.`,
		long: `Replace service principal.

  Updates the details of a single service principal.

  This action replaces the existing service principal with the same name.

  Arguments:
    ID: Databricks service principal ID.`,
	},
}
