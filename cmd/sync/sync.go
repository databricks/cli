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

	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prj := project.Get(ctx)
		wsc := prj.WorkspacesClient()

		if *remotePath == "" {
			me, err := prj.Me()
			if err != nil {
				return err
			}
			repositoryName, err := git.RepositoryName()
			if err != nil {
				return err
			}
			*remotePath = fmt.Sprintf("/Repos/%s/%s", me.UserName, repositoryName)
		}

		log.Printf("[INFO] Remote file sync location: %v", *remotePath)
		repoExists, err := git.RepoExists(*remotePath, ctx, wsc)
		if err != nil {
			return err
		}
		if !repoExists {
			return fmt.Errorf("repo not found, please ensure %s exists", *remotePath)
		}

		root := prj.Root()
		fileSet := git.NewFileSet(root)
		if err != nil {
			return err
		}
		syncCallback := getRemoteSyncCallback(ctx, root, *remotePath, wsc)
		err = spawnSyncRoutine(ctx, fileSet, *interval, syncCallback)
		return err
	},
}

// project files polling interval
var interval *time.Duration

var remotePath *string

var disableSnapshot *bool

func init() {
	root.RootCmd.AddCommand(syncCmd)
	interval = syncCmd.Flags().Duration("interval", 1*time.Second, "project files polling interval")
	remotePath = syncCmd.Flags().String("remote-path", "", "remote path to store repo in. eg: /Repos/me@example.com/test-repo")
	disableSnapshot = syncCmd.Flags().Bool("disable-snapshot", false, "Disable local snapshots of sync state")
}
