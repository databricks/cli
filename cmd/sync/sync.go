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
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/databricks-sdk-go/workspaces"
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

		wsc := project.Current.WorkspacesClient()
		checkouts, err := GetAllRepos(ctx, wsc, "/")
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
			for _, v := range d.delete {
				err := wsc.Workspace.Delete(ctx,
					workspace.DeleteRequest{
						Path:      path.Join(base, v),
						Recursive: true,
					},
				)
				if err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func GetAllRepos(ctx context.Context, wsc *workspaces.WorkspacesClient, pathPrefix string) (resultRepos []repos.GetRepoResponse, err error) {
	nextPageToken := ""
	for {
		listReposResponse, err := wsc.Repos.ListRepos(ctx,
			repos.ListReposRequest{
				PathPrefix:    pathPrefix,
				NextPageToken: nextPageToken,
			},
		)
		if err != nil {
			break
		}
		resultRepos = append(resultRepos, listReposResponse.Repos...)
		if nextPageToken == "" {
			break
		}
	}
	return
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
