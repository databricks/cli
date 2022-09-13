package sync

import (
	"fmt"
	"log"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/project"
	"github.com/databricks/bricks/utilities"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "run syncs for the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		wsc := project.Current.WorkspacesClient()

		me, err := project.Current.Me()
		if err != nil {
			return err
		}
		if *remotePath == "" {
			repositoryName, err := git.RepositoryName()
			if err != nil {
				return err
			}
			*remotePath = fmt.Sprintf("/Repos/%s/%s", me.UserName, repositoryName)
		}

		log.Printf("[INFO] Remote file sync location: %v", *remotePath)
		repos, err := utilities.GetAllRepos(ctx, wsc, *remotePath)
		if err != nil {
			return fmt.Errorf("could not get repos: %s", err)
		}
		if len(repos) == 0 {
			return fmt.Errorf("no matching repo found, please ensure %s exists", *remotePath)
		}
		// TODO: remove this error check by comparing the entire repo name instead
		// of just the prefix. https://github.com/databricks/bricks/issues/53
		if len(repos) > 1 {
			return fmt.Errorf("multiple repos found matching prefix: %s", *remotePath)
		}

		fileSet, err := git.GetFileSet()
		if err != nil {
			return err
		}

		syncCallback := getRemoteSyncCallback(ctx, *remotePath, wsc)
		err = spawnSyncRoutine(ctx, fileSet, *interval, syncCallback)
		return err
	},
}

// project files polling interval
var interval *time.Duration

var remotePath *string

func init() {
	root.RootCmd.AddCommand(syncCmd)
	interval = syncCmd.Flags().Duration("interval", 1*time.Second, "project files polling interval")
	remotePath = syncCmd.Flags().String("remote-path", "", "remote path to store repo in. eg: /Repos/me@example.com/test-repo")
}
