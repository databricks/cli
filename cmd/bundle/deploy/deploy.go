package deploy

import (
	"log"
	"os"

	"github.com/databricks/bricks/bundle"
	parent "github.com/databricks/bricks/cmd/bundle"
	"github.com/spf13/cobra"
)

// TODO: move to bundle directory these codes
// TODO: smoke test all flag configurations

// Q: Why do we pass the context object everywhere?
// Add ability to read logs from deploy

// WIP: will add integration test and develop this command for terraform state sync
// Files in workspace is not available every workspace\

// TODO: place the command under bundle
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

		bundle := bundle.CreateBundle(*env, *localRoot, *remoteRoot)
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
				// TODO: confirm that this error is actually printed
				log.Printf("[ERROR] %s", err)
			}
		}()
		// TODO: terraform apply here
		err = bundle.ImportTerraformState(ctx)
		if err != nil {
			return err
		}

		return nil

		// if *remotePath != "" {
		// 	prj.OverrideRemoteRoot(*remotePath)
		// }

		// targetDir, err := prj.RemoteRoot()
		// if err != nil {
		// 	return err
		// }

		// locker, err := CreateLocker(ctx, false, targetDir)
		// if err != nil {
		// 	return err
		// }

		// err = locker.Lock(ctx)
		// if err != nil {
		// 	return err
		// }
		// defer func() {
		// 	err = locker.Unlock(ctx)
		// 	if err != nil {
		// 		log.Printf("[ERROR] %s", err)
		// 	}
		// }()
		// // time.Sleep(5 * time.Second)

		// remoteTfState, err := readRemoteTfStateFile(ctx)
		// if err != nil {
		// 	return err
		// }

		// localTfState, err := readLocalTfStateFile(ctx)
		// if err != nil {
		// 	return err
		// }

		// combinedTfState := tfStateSchema{
		// 	DeploymentNumber: remoteTfState.DeploymentNumber + 1,
		// 	Name:             localTfState.Name,
		// }

		// err = safeWriteRemoteTfStateFile(ctx, combinedTfState, locker)
		// if err != nil {
		// 	return err
		// }

		// err = writeLocalTfStateFile(ctx, combinedTfState)
		// if err != nil {
		// 	return err
		// }

		// log.Printf("[INFO] deploy completed. congrats!!")
		// // TODO: Unlock lock even in case of an error (put in defer block)
		// return nil
	},
}

var remoteRoot *string
var localRoot *string
var env *string

func init() {
	// root.RootCmd.AddCommand(deployCmd)
	remoteRoot = deployCmd.Flags().String("remote-root", "", "workspace root of the project eg: /Repos/me@example.com/test-repo")
	localRoot = deployCmd.Flags().String("local-root", "", "path to the root directory of the DAB project. default: current working dir")
	env = deployCmd.Flags().String("env", "development", "environment to deploy on. default: development")
	deployCmd.MarkFlagRequired("remote-root")
	parent.AddCommand(deployCmd)
}
