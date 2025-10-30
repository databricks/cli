package deployment

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from Terraform to Direct deployment",
		Long: `Migrate bundle resources from Terraform deployment to Direct deployment.

This command converts your bundle from using Terraform for deployment to using
the Direct deployment engine. It reads resource IDs from the existing Terraform
state and creates a Direct deployment state file (resources.json) with the same
lineage and incremented serial number.

The command will:
1. Read the current Terraform state file (terraform.tfstate)
2. Extract resource IDs and lineage information
3. Create a Direct deployment state file (resources.json)
4. Preserve the same lineage with serial number incremented by 1

After migration:
- The CLI will automatically detect and use Direct deployment based on the
  presence of resources.json
- Use 'bundle deploy' as usual - it will use Direct deployment engine
- Your existing resources will be managed through Direct deployment

WARNING: After migration, don't switch back to Terraform deployment without
proper state management as it may cause resource conflicts.`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts := utils.ProcessOptions{
			AlwaysPull:   true,
			FastValidate: true,
			Build:        true,
		}

		b, err := utils.ProcessBundleWithOut(cmd, &opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		if opts.Winner == nil {
			return errors.New("no existing state found")
		}

		if *b.DirectDeployment {
			return errors.New("already using direct engine")
		}

		terraformResources, err := terraform.ParseResourcesState(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to parse terraform state: %w", err)
		}

		deploymentBundle := &direct.DeploymentBundle{}

		// Initialize adapters (required for MakePlan to work)
		w := b.WorkspaceClient()
		err = deploymentBundle.Init(w)
		if err != nil {
			return fmt.Errorf("failed to initialize deployment bundle: %w", err)
		}

		emptyDB := &dstate.Database{
			State: make(map[string]dstate.ResourceEntry),
		}

		// Create plan from current bundle configuration
		plan, err := deploymentBundle.MakePlan(ctx, &b.Config, emptyDB)
		if err != nil {
			return fmt.Errorf("failed to create deployment plan: %w", err)
		}

		// Create direct state with resource IDs from terraform
		directDB := dstate.Database{
			Serial:  int(opts.Winner.Serial + 1), // Increment serial (convert int64 to int)
			Lineage: opts.Winner.Lineage,         // Same lineage
			State:   make(map[string]dstate.ResourceEntry),
		}

		// Map terraform resources to direct state entries
		for resourceKey, planEntry := range plan.Plan {
			if planEntry.NewState == nil {
				// Skip delete entries or entries without config
				continue
			}

			// Parse resource key using structpath
			keyPath, err := structpath.Parse(resourceKey)
			if err != nil {
				log.Warnf(ctx, "Could not parse resource key: %s, error: %v", resourceKey, err)
				continue
			}

			// Extract group and resource name from path
			// Expected format: resources.{group}.{name} or resources.{group}.{name}.{subresource}
			// Use Prefix(3) to get "resources.group.name" part like the bundle code does
			resourcePrefix := keyPath.Prefix(3)
			if resourcePrefix == nil {
				log.Warnf(ctx, "Resource key too short: %s", resourceKey)
				continue
			}

			// Parse the prefix to get the individual components
			prefixPath, err := structpath.Parse(resourcePrefix.String())
			if err != nil {
				log.Warnf(ctx, "Could not parse resource prefix: %s, error: %v", resourcePrefix.String(), err)
				continue
			}

			// Navigate the path structure to extract components
			// Start from the end and work backwards: resources.group.name
			current := prefixPath
			var components []string
			for current != nil {
				if key, ok := current.StringKey(); ok {
					components = append([]string{key}, components...) // Prepend to reverse order
				}
				current = current.Parent()
			}

			if len(components) != 3 || components[0] != "resources" {
				log.Warnf(ctx, "Invalid resource key format: %s", resourceKey)
				continue
			}

			group := components[1]
			resourceName := components[2]

			// Look up resource ID from terraform state
			if groupResources, hasGroup := terraformResources[group]; hasGroup {
				if resourceState, hasResource := groupResources[resourceName]; hasResource {
					// Create state entry with ID from terraform and current config as state
					directDB.State[resourceKey] = dstate.ResourceEntry{
						ID:    resourceState.ID,
						State: planEntry.NewState.Config, // Use current config as the state
					}
					log.Infof(ctx, "Migrated %s with ID: %s", resourceKey, resourceState.ID)
				} else {
					log.Warnf(ctx, "Resource %s not found in terraform state group %s", resourceName, group)
				}
			} else {
				log.Warnf(ctx, "Resource group %s not found in terraform state", group)
			}
		}

		_, localDirectPath := b.StateFilenameDirect(ctx)
		deploymentState := dstate.DeploymentState{
			Path: localDirectPath,
			Data: directDB,
		}
		err = deploymentState.Finalize()
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, fmt.Sprintf("Migrated %d resources to direct engine state file: %s", len(directDB.State), localDirectPath))
		return nil
	}

	return cmd
}
