// These SCIM commands were previously auto-generated from OpenAPI specs
// but are now manually maintained because the v2 API versions have taken
// over as the generated commands. See https://github.com/databricks/cli/pull/4722.

package users

import (
	users_v2 "github.com/databricks/cli/cmd/account/users-v2"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Slices with functions to override the behavior of individual commands.
var listOverrides []func(*cobra.Command)

// The commands are the users-v2 commands (same API) presented under their
// original names and help text.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: `User identities recognized by Databricks and represented by email addresses.`,
		Long: `User identities recognized by Databricks and represented by email addresses.

  Databricks recommends using SCIM provisioning to sync users and groups
  automatically from your identity provider to your Databricks account. SCIM
  streamlines onboarding a new employee or team by using your identity provider
  to create users and groups in Databricks account and give them the proper
  level of access. When a user leaves your organization or no longer needs
  access to Databricks account, admins can terminate the user in your identity
  provider and that user’s account will also be removed from Databricks
  account. This ensures a consistent offboarding process and prevents
  unauthorized users from accessing sensitive data.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	v2 := users_v2.New()
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
		short: `Create a new user.`,
		long: `Create a new user.

  Creates a new user in the Databricks account. This new user will also be added
  to the Databricks account.`,
	},
	"delete": {
		use:   "delete ID",
		short: `Delete a user.`,
		long: `Delete a user.

  Deletes a user. Deleting a user from a Databricks account also removes objects
  associated with the user.

  Arguments:
    ID: Unique ID for a user in the Databricks account.`,
	},
	"get": {
		use:   "get ID",
		short: `Get user details.`,
		long: `Get user details.

  Gets information for a specific user in Databricks account.

  Arguments:
    ID: Unique ID for a user in the Databricks account.`,
	},
	"list": {
		use:   "list",
		short: `List users.`,
		long: `List users.

  Gets details for all the users associated with a Databricks account.`,
	},
	"patch": {
		use:   "patch ID",
		short: `Update user details.`,
		long: `Update user details.

  Partially updates a user resource by applying the supplied operations on
  specific user attributes.

  Arguments:
    ID: Unique ID in the Databricks workspace.`,
	},
	"update": {
		use:   "update ID",
		short: `Replace a user.`,
		long: `Replace a user.

  Replaces a user's information with the data supplied in request.

  Arguments:
    ID: Databricks user ID.`,
	},
}
