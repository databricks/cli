package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/direct"
	"github.com/databricks/cli/ucm/direct/dstate"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

const backupSuffix = ".backup"

// terraformStateHeader pulls the lineage + serial from a terraform.tfstate
// blob without reparsing the full resources tree (handled separately by
// terraform.ParseResourcesState). The fields are top-level in tfstate JSON
// and are required to seed direct-engine state with continuity — see
// dstate.NewDatabase / DeploymentState.Finalize for how lineage/serial are
// used by the direct engine.
type terraformStateHeader struct {
	Lineage string `json:"lineage"`
	Serial  int    `json:"serial"`
}

// readTerraformStateHeader returns lineage + serial from a local terraform
// state file. Returns an error if the file is missing or unparseable.
func readTerraformStateHeader(path string) (terraformStateHeader, error) {
	var hdr terraformStateHeader
	raw, err := os.ReadFile(path)
	if err != nil {
		return hdr, err
	}
	if err := json.Unmarshal(raw, &hdr); err != nil {
		return hdr, fmt.Errorf("parse %s: %w", path, err)
	}
	return hdr, nil
}

// runPlanCheck runs `ucm plan` and checks if there are any actions planned.
// Returns error if plan fails or if there are actions planned.
func runPlanCheck(cmd *cobra.Command, extraArgs []string, extraArgsStr string) error {
	ctx := cmd.Context()

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	args := []string{"ucm", "plan"}
	args = append(args, extraArgs...)

	planCmd := exec.CommandContext(ctx, executable, args...)
	var stdout bytes.Buffer
	planCmd.Stdout = &stdout
	planCmd.Stderr = cmd.ErrOrStderr()

	// Force the spawned plan to use the terraform engine — we need to verify
	// the existing terraform-managed deployment is in sync before we migrate.
	planCmd.Env = append(os.Environ(), engine.EnvVar+"=terraform")

	err = planCmd.Run()

	// Output the plan stdout as is.
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
		return fmt.Errorf("ucm plan failed with %s, aborting migration. To proceed with migration anyway, re-run the command with --noplancheck option", msg)
	}

	if !strings.Contains(output, "Plan:") {
		return fmt.Errorf("cannot parse 'databricks ucm plan%s' output, aborting migration. Skip plan check with --noplancheck option", extraArgsStr)
	}

	if !strings.Contains(output, "Plan: 0 to add, 0 to change, 0 to delete") {
		return fmt.Errorf("'databricks ucm plan%s' shows actions planned, aborting migration. Please run 'databricks ucm deploy%s' first to ensure your project is up to date, If actions persist after deploy, skip plan check with --noplancheck option", extraArgsStr, extraArgsStr)
	}

	return nil
}

// getCommonArgs propagates --target / --profile / --var to the spawned
// `ucm plan` invocation so it sees the same target+context as the migrate
// run. Mirrors the bundle helper of the same name.
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

// newMigrateCommand returns `databricks ucm deployment migrate`.
//
// Migrates a ucm project from the terraform engine to the direct engine by
// reading the local terraform.tfstate, building a dstate.Database with the
// same lineage + serial+1, and persisting it via the direct-engine state
// machinery. Mirrors `bundle deployment migrate` (cmd/bundle/deployment/
// migrate.go); the few UCM-specific divergences are called out inline.
func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from Terraform to Direct deployment engine",
		Long: `This command converts your ucm project from using Terraform for deployment
to using the Direct deployment engine. It reads resource IDs from the existing
Terraform state and creates a Direct deployment state file (resources.json)
with the same lineage and incremented serial number.

Note, the migration is performed locally only. To finalize it, run 'ucm deploy'. This will synchronize the state file
to the workspace so that subsequent deploys of this project use direct deployment engine as well.
`,
		Args: root.NoArgs,
	}

	var noPlanCheck bool
	cmd.Flags().BoolVar(&noPlanCheck, "noplancheck", false, "Skip running ucm plan before migration.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		extraArgs, extraArgsStr := getCommonArgs(cmd)

		// Clear the engine env var so migrate always uses terraform engine to
		// read existing state, regardless of what the user may have set in
		// their environment.
		cmd.SetContext(env.Set(cmd.Context(), engine.EnvVar, ""))

		opts := utils.ProcessOptions{
			AlwaysPull: true,
			// Same options as regular deploy, to ensure config is in the same state.
			FastValidate: true,
			Build:        true,
			PostInitFunc: func(_ context.Context, u *ucm.Ucm) error {
				if u.Config.Ucm.Engine == engine.EngineTerraform {
					return fmt.Errorf("ucm.engine is set to %q. Migration requires \"engine: direct\" or no engine setting. Change the setting to \"engine: direct\" and retry", engine.EngineTerraform)
				}
				return nil
			},
		}

		u, err := utils.ProcessUcm(cmd, opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		// UCM has no equivalent of bundle's StateDesc / PullResourcesState
		// today (the remote-state lineage/serial machinery is bundle-only).
		// Instead, derive lineage + serial directly from the local
		// terraform.tfstate header, which is the same source of truth bundle
		// reads via state pull. The direct state file is fresh (we error if
		// it already exists), so there is no second source to reconcile.
		localTerraformPath := deploy.LocalTfStatePath(u)
		if _, err = os.Stat(localTerraformPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				cmdio.LogString(ctx, `Error: This command migrates the existing Terraform state file (terraform.tfstate) to a direct deployment state file (resources.json). However, no existing local state was found.

To start using direct engine, set "engine: direct" under ucm in your ucm.yml or deploy with DATABRICKS_UCM_ENGINE=direct env var set.`)
				return root.ErrAlreadyPrinted
			}
			return fmt.Errorf("reading %s: %w", localTerraformPath, err)
		}

		header, err := readTerraformStateHeader(localTerraformPath)
		if err != nil {
			return fmt.Errorf("failed to read terraform state header: %w", err)
		}
		if header.Lineage == "" {
			cmdio.LogString(ctx, `Error: This command migrates the existing Terraform state file (terraform.tfstate) to a direct deployment state file (resources.json). However, the existing state has no lineage recorded.

To start using direct engine, set "engine: direct" under ucm in your ucm.yml or deploy with DATABRICKS_UCM_ENGINE=direct env var set.`)
			return root.ErrAlreadyPrinted
		}

		terraformResources, err := terraform.ParseResourcesState(ctx, u)
		if err != nil {
			return fmt.Errorf("failed to parse terraform state: %w", err)
		}

		for key, resourceEntry := range terraformResources {
			if resourceEntry.ID == "" {
				return fmt.Errorf("failed to interpret terraform state for %s: missing ID", key)
			}
		}

		localPath := phases.DirectStatePath(u)
		tempStatePath := localPath + ".temp-migration"
		if _, err = os.Stat(tempStatePath); err == nil {
			return fmt.Errorf("temporary state file %s already exists, another migration is in progress or was interrupted. In the latter case, delete the file", tempStatePath)
		}
		// "Already using direct" is detected by the local resources.json
		// file, not via StateDesc (which UCM does not have today; see #145).
		// A user who deletes resources.json and re-runs migrate against a
		// still-present terraform.tfstate.backup would re-create the direct
		// state cleanly.
		if _, err = os.Stat(localPath); err == nil {
			return fmt.Errorf("state file %s already exists", localPath)
		}

		// Run plan check unless --noplancheck is set.
		if !noPlanCheck {
			cmdio.LogString(ctx, "Note: Migration should be done after a full deploy. Running plan now to verify that deployment was done:")
			if err = runPlanCheck(cmd, extraArgs, extraArgsStr); err != nil {
				return err
			}
		}

		// UCM has no Dashboard (or any other ETag-bearing) resource type
		// today. Bundle copies Dashboard ETags into the migrated state so
		// post-migrate drift detection works; ucm's resource set has no
		// analog, so the per-key etag propagation block from bundle's
		// migrate is intentionally omitted. Tracked in #145 — revisit when
		// ucm grows an ETag-bearing resource.
		state := make(map[string]dstate.ResourceEntry)
		for key, resourceEntry := range terraformResources {
			state[key] = dstate.ResourceEntry{
				ID:    resourceEntry.ID,
				State: json.RawMessage("{}"),
			}
		}

		migratedDB := dstate.NewDatabase(header.Lineage, header.Serial+1)
		migratedDB.State = state

		deploymentUcm := &direct.DeploymentUcm{
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

		// Bundle's migrate calls resourcemutator.SecretScopeFixups here so
		// the config matches what the direct engine expects (MANAGE ACL on
		// every secret scope). UCM has no secret-scope resource type, so
		// the call is intentionally absent — see issue #145 for the full
		// rationale. If a UC-specific direct-engine fixup ever becomes
		// necessary, wire it here.

		plan, err := deploymentUcm.CalculatePlan(ctx, u.WorkspaceClient(), &u.Config)
		if err != nil {
			return err
		}

		for _, entry := range plan.Plan {
			// Force every action to "update" so the apply pass below walks
			// every resource in the migrated state and writes the
			// fully-resolved local config snapshot into it.
			entry.Action = deployplan.Update
		}

		deploymentUcm.Apply(ctx, u.WorkspaceClient(), plan, direct.MigrateMode(true))
		if err := deploymentUcm.StateDB.Finalize(); err != nil {
			logdiag.LogError(ctx, err)
		}
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
			// Not fatal — serial has already been incremented in the new
			// state file, so the migration succeeded even if the backup
			// rename did not.
			logdiag.LogError(ctx, err)
		}

		cmdio.LogString(ctx, fmt.Sprintf(`Success! Migrated %d resources to direct engine state file: %s

Validate the migration by running "databricks ucm plan%s", there should be no actions planned.

The state file is not synchronized to the workspace yet. To do that and finalize the migration, run "ucm deploy%s".

To undo the migration, remove %s and rename %s to %s
`, len(deploymentUcm.StateDB.Data.State), localPath, extraArgsStr, extraArgsStr, localPath, localTerraformBackupPath, localTerraformPath))
		return nil
	}

	return cmd
}
