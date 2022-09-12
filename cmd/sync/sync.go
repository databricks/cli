package sync

import (
	"fmt"
	"log"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/project"
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
		repositoryName, err := git.RepositoryName()
		if err != nil {
			return err
		}
		databricksRepoDir := fmt.Sprintf("/Repos/%s/%s", me.UserName, repositoryName)
		log.Printf("[INFO] Remote file sync location: %v", databricksRepoDir)

		fileSet, err := git.GetFileSet()
		if err != nil {
			return err
		}

		syncCallback := getRemoteSyncCallback(ctx, databricksRepoDir, wsc)
		err = spawnSyncRoutine(ctx, fileSet, *interval, syncCallback)
		return err
	},
}

// project files polling interval
var interval *time.Duration

func init() {
	root.RootCmd.AddCommand(syncCmd)
	interval = syncCmd.Flags().Duration("interval", 1*time.Second, "project files polling interval")
}
