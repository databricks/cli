package sync

import (
	"fmt"
	"log"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/git"
	"github.com/databricks/bricks/libs/sync"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize a local directory to a workspace directory",

	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prj := project.Get(ctx)
		wsc := prj.WorkspacesClient()

		me, err := prj.Me()
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

		cacheDir, err := prj.CacheDir()
		if err != nil {
			return err
		}

		opts := sync.SyncOptions{
			LocalPath:        prj.Root(),
			RemotePath:       *remotePath,
			Full:             *full,
			SnapshotBasePath: cacheDir,
			PollInterval:     *interval,
			WorkspaceClient:  wsc,
		}

		s, err := sync.New(ctx, opts)
		if err != nil {
			return err
		}

		if *watch {
			return s.RunContinuous(ctx)
		}

		return s.RunOnce(ctx)
	},
}

// project files polling interval
var interval *time.Duration

var remotePath *string

var full *bool
var watch *bool

func init() {
	root.RootCmd.AddCommand(syncCmd)
	interval = syncCmd.Flags().Duration("interval", 1*time.Second, "file system polling interval (for --watch)")
	remotePath = syncCmd.Flags().String("remote-path", "", "remote path to store repo in. eg: /Repos/me@example.com/test-repo")
	full = syncCmd.Flags().Bool("full", false, "perform full synchronization (default is incremental)")
	watch = syncCmd.Flags().Bool("watch", false, "watch local file system for changes")
}
