package deployment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/spf13/cobra"
)

const backupSuffix = ".backup"

// runPlanCheck runs bundle plan and checks if there are any actions planned.
// Returns error if plan fails or if there are actions planned.
func runPlanCheck(cmd *cobra.Command, extraArgs []string, extraArgsStr string) error {
	ctx := cmd.Context()

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	args := []string{"bundle", "plan"}
	args = append(args, extraArgs...)

	planCmd := exec.CommandContext(ctx, executable, args...)
	var stdout bytes.Buffer
	planCmd.Stdout = &stdout
	planCmd.Stderr = cmd.ErrOrStderr()

	// Use the engine encoded in the state
	planCmd.Env = append(os.Environ(), "DATABRICKS_BUNDLE_ENGINE=terraform")

	err = planCmd.Run()

	// Output the plan stdout as is
	output := stdout.String()
	fmt.Fprint(cmd.OutOrStdout(), output)

	if err != nil {
		var exitErr *exec.ExitError
		msg := ""
		if errors.As(err, &exitErr) {
			msg = fmt.Sprintf("exit code %d", exitErr.ExitCode())
		} else {
			msg = err.Error()
		}
		return fmt.Errorf("bundle plan failed with %s, aborting migration. To proceed with migration anyway, re-run the command with --noplancheck option", msg)
	}

	if !strings.Contains(output, "Plan:") {
		return fmt.Errorf("cannot parse 'databricks bundle plan%s' output, aborting migration. Skip plan check with --noplancheck option", extraArgsStr)
	}

	if !strings.Contains(output, "Plan: 0 to add, 0 to change, 0 to delete") {
		return fmt.Errorf("'databricks bundle plan%s' shows actions planned, aborting migration. Please run 'databricks bundle deploy%s' first to ensure your bundle is up to date, If actions persist after deploy, skip plan check with --noplancheck option", extraArgsStr, extraArgsStr)
	}

	return nil
}

func getCommonArgs(cmd *cobra.Command) ([]string, string) {
	var args []string
	var quotedArgs []string

	if flag := cmd.Flag("target"); flag != nil && flag.Changed {
		target := flag.Value.String()
		if target != "" {
			args = append(args, "-t")
			args = append(args, target)
			quotedArgs = append(quotedArgs, "-t")
			quotedArgs = append(quotedArgs, shellquote.BashArg(target))
		}
	}
	if flag := cmd.Flag("profile"); flag != nil && flag.Changed {
		profile := flag.Value.String()
		if profile != "" {
			args = append(args, "-p")
			args = append(args, profile)
			quotedArgs = append(quotedArgs, "-p")
			quotedArgs = append(quotedArgs, shellquote.BashArg(profile))
		}
	}
	if flag := cmd.Flag("var"); flag != nil && flag.Changed {
		varValues, err := cmd.Flags().GetStringSlice("var")
		if err == nil {
			for _, v := range varValues {
				args = append(args, "--var")
				args = append(args, v)
				quotedArgs = append(quotedArgs, "--var")
				quotedArgs = append(quotedArgs, shellquote.BashArg(v))
			}
		}
	}

	argsStr := ""

	if len(quotedArgs) > 0 {
		argsStr = " " + strings.Join(quotedArgs, " ")
	}

	return args, argsStr
}

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from Terraform to Direct deployment engine",
		Long: `This command converts your bundle from using Terraform for deployment to using
the Direct deployment engine. It reads resource IDs from the existing Terraform
state and creates a Direct deployment state file (resources.json) with the same
lineage and incremented serial number.

Note, the migration is performed locally only. To finalize it, run 'bundle deploy'. This will synchronize the state file
to the workspace so that subsequent deploys of this bundle use direct deployment engine as well.

WARNING: Both direct deployment engine and this command are experimental and not recommended for production targets yet.
`,
		Args: root.NoArgs,
	}

	var noPlanCheck bool
	cmd.Flags().BoolVar(&noPlanCheck, "noplancheck", false, "Skip running bundle plan before migration.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		extraArgs, extraArgsStr := getCommonArgs(cmd)

		opts := utils.ProcessOptions{
			SkipEngineEnvVar: true,
			AlwaysPull:       true,
			// Same options as regular deploy, to ensure bundle config is in the same state
			FastValidate: true,
			Build:        true,
		}

		b, stateDesc, err := utils.ProcessBundleRet(cmd, opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		if stateDesc.Lineage == "" {
			// TODO: mention bundle.engine once it's there
			cmdio.LogString(ctx, `Error: This command migrates the existing Terraform state file (terraform.tfstate) to a direct deployment state file (resources.json). However, no existing local or remote state was found.

To start using direct engine, deploy with DATABRICKS_BUNDLE_ENGINE=direct env var set.`)
			return root.ErrAlreadyPrinted
		}

		if stateDesc.Engine.IsDirect() {
			return fmt.Errorf("already using direct engine\nDetails: %s", stateDesc.String())
		}

		_, localTerraformPath := b.StateFilenameTerraform(ctx)
		if _, err = os.Stat(localTerraformPath); err != nil {
			return fmt.Errorf("reading %s: %w", localTerraformPath, err)
		}

		terraformResources, err := terraform.ParseResourcesState(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to parse terraform state: %w", err)
		}

		_, localPath := b.StateFilenameDirect(ctx)
		tempStatePath := localPath + ".temp-migration"
		if _, err = os.Stat(tempStatePath); err == nil {
			return fmt.Errorf("temporary state file %s already exists, another migration is in progress or was interrupted. In the latter case, delete the file", tempStatePath)
		}
		if _, err = os.Stat(localPath); err == nil {
			return fmt.Errorf("state file %s already exists", localPath)
		}

		// Run plan check unless --noplancheck is set
		if !noPlanCheck {
			fmt.Fprintf(cmd.OutOrStdout(), "Note: Migration should be done after a full deploy. Running plan now to verify that deployment was done:\n")
			if err = runPlanCheck(cmd, extraArgs, extraArgsStr); err != nil {
				return err
			}
		}

		etags := map[string]string{}

		state := make(map[string]dstate.ResourceEntry)
		for key, resourceEntry := range terraformResources {
			state[key] = dstate.ResourceEntry{
				ID:    resourceEntry.ID,
				State: json.RawMessage("{}"),
			}
			if resourceEntry.ETag != "" {
				// dashboard:
				etags[key] = resourceEntry.ETag
			}
		}

		migratedDB := dstate.NewDatabase(stateDesc.Lineage, stateDesc.Serial+1)
		migratedDB.State = state

		deploymentBundle := &direct.DeploymentBundle{
			StateDB: dstate.DeploymentState{
				Path: tempStatePath,
				Data: migratedDB,
			},
		}

		tempStatePathAutoRemove := true

		defer func() {
			if tempStatePathAutoRemove {
				_ = os.Remove(tempStatePath)
			}
		}()

		plan, err := deploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(), &b.Config, tempStatePath)
		if err != nil {
			return err
		}

		// We need to copy ETag into new state.
		// For most resources state consists of fully resolved local config snapshot + id.
		// Dashboards are special in that they also store "etag" in state which is not provided by user but
		// comes from remote state. If we don't store "etag" in state, we won't detect remote drift, because
		// local=nil, remote="<some new etag>" which will be classified as "server_side_default".

		for key := range plan.Plan {
			etag := etags[key]
			if etag == "" {
				continue
			}
			// Get the cached StructVar created during planning
			sv, ok := deploymentBundle.StructVarCache.Load(key)
			if !ok {
				return fmt.Errorf("failed to get cached state for %s", key)
			}
			err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag)
			if err != nil {
				return fmt.Errorf("failed to set etag on %s: %w", key, err)
			}
		}

		deploymentBundle.Apply(ctx, b.WorkspaceClient(), &b.Config, plan, direct.MigrateMode(true))
		if logdiag.HasError(ctx) {
			logdiag.LogError(ctx, errors.New("migration failed; ensure you have done full deploy before the migration"))
			return root.ErrAlreadyPrinted
		}

		if err := os.Rename(tempStatePath, localPath); err != nil {
			return fmt.Errorf("renaming %s to %s: %w", tempStatePath, localPath, err)
		}
		tempStatePathAutoRemove = false

		localTerraformBackupPath := localTerraformPath + backupSuffix

		err = os.Rename(localTerraformPath, localTerraformBackupPath)
		if err != nil {
			// not fatal, since we've increased serial
			logdiag.LogError(ctx, err)
		}

		cmdio.LogString(ctx, fmt.Sprintf(`Success! Migrated %d resources to direct engine state file: %s

Validate the migration by running "databricks bundle plan%s", there should be no actions planned.

The state file is not synchronized to the workspace yet. To do that and finalize the migration, run "bundle deploy%s".

To undo the migration, remove %s and rename %s to %s
`, len(deploymentBundle.StateDB.Data.State), localPath, extraArgsStr, extraArgsStr, localPath, localTerraformBackupPath, localTerraformPath))
		return nil
	}

	return cmd
}
