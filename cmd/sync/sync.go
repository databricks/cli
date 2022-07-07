package sync

import (
	"context"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/project"
	"github.com/databrickslabs/terraform-provider-databricks/repos"
	"github.com/databrickslabs/terraform-provider-databricks/workspace"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "run syncs for the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		origin, err := git.HttpsOrigin()
		if err != nil {
			return err
		}
		log.Printf("[INFO] %s", origin)
		ctx := cmd.Context()
		client := project.Current.Client()
		reposAPI := repos.NewReposAPI(ctx, client)


		checkouts, err := reposAPI.List("/")
		if err != nil {
			return err
		}
		for _, v := range checkouts {
			log.Printf("[INFO] %s", v.Url)
		}
		me := project.Current.Me()
		repositoryName, err := git.RepositoryName()
		if err != nil {
			return err
		}
		base := fmt.Sprintf("/Repos/%s/%s", me.UserName, repositoryName)
		return watchForChanges(ctx, git.MustGetFileSet(), *interval, func(d diff) error {
			wsAPI := workspace.NewNotebooksAPI(ctx, client)
			for _, v := range d.delete {
				err := wsAPI.Delete(path.Join(base, v), true)
				if err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func ImportFile(ctx context.Context, path string, content io.Reader) error {
	client := project.Current.Client()
	apiPath := fmt.Sprintf(
		"/workspace-files/import-file/%s?overwrite=true",
		strings.TrimLeft(path, "/"))
	// TODO: change upstream client to support io.Reader as body
	return client.Post(ctx, apiPath, content, nil)
}

// project files polling interval
var interval *time.Duration

func init() {
	root.RootCmd.AddCommand(syncCmd)
	interval = syncCmd.Flags().Duration("interval", 1*time.Second, "project files polling interval")
}
