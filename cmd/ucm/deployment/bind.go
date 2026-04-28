package deployment

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

// errBindAborted is the sentinel returned when the user answers "no" to the
// bind confirmation prompt.
var errBindAborted = errors.New("bind aborted")

// errNeedsAutoApprove is returned when the terminal cannot prompt and
// --auto-approve was not supplied.
var errNeedsAutoApprove = errors.New("this operation requires user confirmation, but the current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")

// newBindCommand returns `databricks ucm deployment bind KEY UC_NAME`.
// Records a state entry so subsequent deploys update — rather than recreate —
// the existing UC object. The ucm.yml config is never modified.
func newBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind KEY UC_NAME",
		Short: "Bind a ucm-declared resource to an existing Unity Catalog object",
		Long: `Bind a ucm-declared resource to an existing Unity Catalog object.

After binding, subsequent deploys reconcile (update) the UC object rather
than attempt to create a new one.

Arguments:
  KEY     - The resource key declared in ucm.yml (e.g. team_alpha)
  UC_NAME - The name/full-name of the existing Unity Catalog object

Examples:
  # Bind a catalog declaration to an existing UC catalog
  databricks ucm deployment bind team_alpha team_alpha

  # Bind a schema (UC_NAME must be the schema's full name)
  databricks ucm deployment bind bronze team_alpha.bronze

  # Bind with automatic approval (CI/CD)
  databricks ucm deployment bind my_vol team_alpha.bronze.landing --auto-approve

Supported kinds: catalogs, schemas, storage_credentials, external_locations,
volumes, connections. Grants are not bindable (they reconcile per securable).

WARNING: After binding, the UC object will be managed by ucm. Manual changes
made outside ucm may be overwritten on the next deploy.`,
		Args: root.ExactArgs(2),
	}

	var autoApprove bool
	var forceLock bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the binding.")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		key, ucName := args[0], args[1]

		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{
			InitFunc: func(u *ucm.Ucm) {
				u.ForceLock = forceLock
			},
			InitIDs: true,
		})
		ctx := cmd.Context()
		if err != nil {
			return err
		}
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		kind, err := resolveBindable(u, key)
		if err != nil {
			return err
		}

		if err := validateBindName(kind, ucName); err != nil {
			return err
		}

		if !autoApprove {
			if !cmdio.IsPromptSupported(ctx) {
				return errNeedsAutoApprove
			}
			ok, err := cmdio.AskYesOrNo(ctx, "Bind "+string(kind)+"."+key+" -> "+ucName+"?")
			if err != nil {
				return err
			}
			if !ok {
				return errBindAborted
			}
		}

		// UCM's phases.Bind needs a Backend + TerraformFactory that ProcessUcm
		// does not yet plumb (tracked in #103).
		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve bind options: %w", err)
		}
		opts.ForceLock = u.ForceLock

		phases.Bind(ctx, u, opts, phases.BindRequest{Kind: kind, Name: ucName, Key: key})
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		cmdio.LogString(ctx, "Successfully bound "+string(kind)+"."+key+" to "+ucName)
		cmdio.LogString(ctx, "Run 'databricks ucm deploy' to reconcile the bound resource.")
		return nil
	}

	return cmd
}
