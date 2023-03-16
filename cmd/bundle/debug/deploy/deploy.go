package deploy

import (
	"os"
	"path/filepath"

	"github.com/databricks/bricks/bundle/deployer"
	"github.com/databricks/bricks/cmd/bundle/debug"
	"github.com/databricks/bricks/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/spf13/cobra"
)

// TODO: will add integration test once terraform binary is bundled with bricks
var deployTerraformCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploys resources defined in a terraform config to a Databricks workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// default to cwd
		if *localRoot == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			*localRoot = cwd
		}

		if *terraformBinaryPath == "" {
			installer := releases.ExactVersion{
				Product: product.Terraform,
				Version: version.Must(version.NewVersion("1.2.4")),
			}
			execPath, err := installer.Install(ctx)
			if err != nil {
				log.Errorf(ctx, "error installing Terraform: %s", err)
			}
			*terraformBinaryPath = execPath
			defer installer.Remove(ctx)
		}

		// TODO: load bundle and get the workspace client from there once bundles
		// are stable
		wsc, err := databricks.NewWorkspaceClient()
		if err != nil {
			return err
		}

		d, err := deployer.Create(ctx, *env, *localRoot, *remoteRoot, wsc)
		if err != nil {
			return err
		}

		if *terraformHcl == "" {
			*terraformHcl = filepath.Join(d.DefaultTerraformRoot())
		}

		status, err := d.ApplyTerraformConfig(ctx, *terraformHcl, *terraformBinaryPath, *isForced)
		switch status {
		case deployer.Failed:
			log.Errorf(ctx, "failed to initiate deployment")
		case deployer.NoChanges:
			log.Infof(ctx, "no changes detected")
		case deployer.Partial:
			log.Errorf(ctx, "started deployment, but failed to complete")
		case deployer.PartialButUntracked:
			log.Errorf(ctx, "started deployment, but failed to complete. Any partially deployed resources in this run are untracked in the databricks workspace and might not be cleaned up on future deployments")
		case deployer.CompleteButUntracked:
			log.Errorf(ctx, "deployment complete. Failed to track deployed resources. Any deployed resources in this run are untracked in the databricks workspace and might not be cleaned up on future deployments")
		case deployer.Complete:
			log.Infof(ctx, "deployment complete")
		}
		return err
	},
}

var remoteRoot *string
var localRoot *string
var env *string
var isForced *bool

var terraformHcl *string

// TODO: remove this arguement once we package a terraform binary with the bricks cli
var terraformBinaryPath *string

func init() {
	remoteRoot = deployTerraformCmd.Flags().String("remote-root", "", "workspace root of the project eg: /Repos/me@example.com/test-repo")
	localRoot = deployTerraformCmd.Flags().String("local-root", "", "path to the root directory of the DAB project. default: current working dir")
	terraformBinaryPath = deployTerraformCmd.Flags().String("terraform-cli-binary", "", "path to a terraform CLI executable binary")
	env = deployTerraformCmd.Flags().String("env", "development", "environment to deploy on. default: development")
	isForced = deployTerraformCmd.Flags().Bool("force", false, "force deploy your DAB to the workspace. default: false")
	terraformHcl = deployTerraformCmd.Flags().String("terraform-hcl", "", "path to the terraform config file from project root")

	deployTerraformCmd.MarkFlagRequired("remote-root")
	debug.AddCommand(deployTerraformCmd)
}
