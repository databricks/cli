package deploy

import (
	"log"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

// WIP: will add integration test and develop this command for terraform state sync
var deployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "deploys a DAB",
	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prj := project.Get(ctx)

		if *remotePath != "" {
			prj.OverrideRemoteRoot(*remotePath)
		}

		targetDir, err := prj.RemoteRoot()
		if err != nil {
			return err
		}

		locker, err := CreateLocker(ctx, false, targetDir)
		if err != nil {
			return err
		}

		err = locker.Lock(ctx)
		if err != nil {
			return err
		}
		defer func() {
			err = locker.Unlock(ctx)
			if err != nil {
				log.Printf("[ERROR] %s", err)
			}
		}()
		// time.Sleep(5 * time.Second)

		remoteTfState, err := readRemoteTfStateFile(ctx)
		if err != nil {
			return err
		}

		localTfState, err := readLocalTfStateFile(ctx)
		if err != nil {
			return err
		}

		combinedTfState := tfStateSchema{
			DeploymentNumber: remoteTfState.DeploymentNumber + 1,
			Name:             localTfState.Name,
		}

		err = safeWriteRemoteTfStateFile(ctx, combinedTfState, locker)
		if err != nil {
			return err
		}

		err = writeLocalTfStateFile(ctx, combinedTfState)
		if err != nil {
			return err
		}

		log.Printf("[INFO] deploy completed. congrats!!")
		// TODO: Unlock lock even in case of an error (put in defer block)
		return nil
	},
}

var remotePath *string

func init() {
	root.RootCmd.AddCommand(deployCmd)
	remotePath = deployCmd.Flags().String("remote-path", "", "workspace root of the project eg: /Repos/me@example.com/test-repo")
}
