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
		repositoryName, err := git.RepositoryName()
		if err != nil {
			return err
		}
		repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, repositoryName)
		log.Printf("[INFO] Remote file sync location: %v", repoPath)

		repos, err := utilities.GetAllRepos(ctx, wsc, repoPath)
		if err != nil {
			return fmt.Errorf("could not get repos: %s", err)
		}
		if len(repos) == 0 {
			return fmt.Errorf("no matching repo found, please ensure %s exists", repoPath)
		}
		if len(repos) > 1 {
			return fmt.Errorf("multiple repos found matching prefix: %s", repoPath)
		}

		fileSet, err := git.GetFileSet()
		if err != nil {
			return err
		}

		syncCallback := getRemoteSyncCallback(ctx, repoPath, wsc)
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
