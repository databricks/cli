package deploy

import (
	"log"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/bundle/deployer"
	"github.com/databricks/bricks/cmd/bundle/debug"
	"github.com/databricks/databricks-sdk-go"
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

		if *terraformRoot == "" {
			*terraformRoot = filepath.Join(d.DefaultTerraformRoot())
		}

		status, err := d.ApplyTerraformConfig(ctx, *terraformRoot, *terraformBinaryPath)
		switch status {
		case deployer.Failed:
			log.Printf("[ERROR] failed to initiate deployment")
		case deployer.NoChanges:
			log.Printf("[INFO] no changes detected")
		case deployer.Partial:
			log.Printf("[ERROR] started deployment, but failed to complete")
		case deployer.Success:
			log.Printf("[INFO] deployment complete")
		}
		return err
	},
}

var remoteRoot *string
var localRoot *string
var env *string

// todo test if this works with terraform json file
var terraformRoot *string

// TODO: remove this arguement once we package a terraform binary with the bricks cli
var terraformBinaryPath *string

func init() {
	// root.RootCmd.AddCommand(deployCmd)
	remoteRoot = deployTerraformCmd.Flags().String("remote-root", "", "workspace root of the project eg: /Repos/me@example.com/test-repo")
	localRoot = deployTerraformCmd.Flags().String("local-root", "", "path to the root directory of the DAB project. default: current working dir")
	terraformBinaryPath = deployTerraformCmd.Flags().String("terraform-cli-binary", "", "path to a terraform CLI executable binary")
	env = deployTerraformCmd.Flags().String("env", "development", "environment to deploy on. default: development")

	// TODO: test whether this command works with json terraform config?
	terraformRoot = deployTerraformCmd.Flags().String("terraform-root", "", "path to the terraform config file from project root")

	deployTerraformCmd.MarkFlagRequired("remote-root")
	deployTerraformCmd.MarkFlagRequired("terraform-cli-binary")
	debug.AddCommand(deployTerraformCmd)
}
