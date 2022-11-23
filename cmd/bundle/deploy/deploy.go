package deploy

import (
	"fmt"
	"log"
	"os"

	"github.com/databricks/bricks/bundle"
	parent "github.com/databricks/bricks/cmd/bundle"
	"github.com/spf13/cobra"
)

// TODO: smoke test all flag configurations

// WIP: will add integration test and develop this command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploys a DAB",
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

		bundle := bundle.CreateBundle(*env, *localRoot, *remoteRoot, *terraformBinaryPath)
		err := bundle.ExportTerraformState(ctx)
		if err != nil {
			return err
		}

		err = bundle.Lock(ctx)
		if err != nil {
			return err
		}
		defer func() {
			err = bundle.Unlock(ctx)
			if err != nil {
				log.Printf("[ERROR] %s", err)
			}
		}()
		tf, err := bundle.GetTerraformHandle(ctx)
		nonEmptyPlan, err := tf.Plan(ctx)
		if err != nil {
			return err
		}
		if !nonEmptyPlan {
			log.Printf("[INFO] state diff is empty. No changes applied")
			return nil
		}
		err = tf.Apply(ctx)
		// upload state even if apply fails to handle partial deployments
		err2 := bundle.ImportTerraformState(ctx)
		if err != nil {
			return fmt.Errorf("deploymented failed: %s", err)
		}
		if err2 != nil {
			return fmt.Errorf("failed to upload updated tfstate file: %s", err2)
		}
		log.Printf("[INFO] deployment complete")
		return nil
	},
}

var remoteRoot *string
var localRoot *string
var env *string

// TODO: remove this arguement once we package a terraform binary with the bricks cli
var terraformBinaryPath *string

func init() {
	// root.RootCmd.AddCommand(deployCmd)
	remoteRoot = deployCmd.Flags().String("remote-root", "", "workspace root of the project eg: /Repos/me@example.com/test-repo")
	localRoot = deployCmd.Flags().String("local-root", "", "path to the root directory of the DAB project. default: current working dir")
	terraformBinaryPath = deployCmd.Flags().String("tf-path", "", "path to a terraform executable binary")
	env = deployCmd.Flags().String("env", "development", "environment to deploy on. default: development")
	deployCmd.MarkFlagRequired("remote-root")
	deployCmd.MarkFlagRequired("tf-path")
	parent.AddCommand(deployCmd)
}
