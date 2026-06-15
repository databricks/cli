package labs

import (
	"context"
	"errors"
	"time"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/cmd/labs/localcache"
	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

const (
	labsOrg             = "databrickslabs"
	installableCacheTTL = 24 * time.Hour
)

type labsMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	License     string `json:"license"`
}

func allRepos(ctx context.Context) (github.Repositories, error) {
	cacheDir, err := project.PathInLabs(ctx)
	if err != nil {
		return nil, err
	}
	cache := github.NewRepositoryCache(labsOrg, cacheDir)
	return cache.Load(ctx)
}

// installableRepos returns the org repositories that `databricks labs install` can
// actually install. Most repositories in the org don't ship a labs.yml manifest
// (e.g. libraries published to package indexes), so listing them would only
// advertise projects that fail to install.
func installableRepos(ctx context.Context) (github.Repositories, error) {
	cacheDir, err := project.PathInLabs(ctx)
	if err != nil {
		return nil, err
	}
	cache := localcache.NewLocalCache[github.Repositories](cacheDir, labsOrg+"-installable-repositories", installableCacheTTL)
	return cache.Load(ctx, func() (github.Repositories, error) {
		repos, err := allRepos(ctx)
		if err != nil {
			return nil, err
		}
		return filterInstallable(ctx, repos)
	})
}

// filterInstallable keeps repositories that have a root labs.yml manifest on their
// default branch. The manifest is fetched from raw.githubusercontent.com, which is
// not subject to the low unauthenticated GitHub API rate limit.
func filterInstallable(ctx context.Context, repos github.Repositories) (github.Repositories, error) {
	installable := make([]bool, len(repos))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	for i, repo := range repos {
		if repo.IsArchived || repo.IsFork {
			continue
		}
		g.Go(func() error {
			_, err := github.ReadFileFromRef(gctx, labsOrg, repo.Name, repo.DefaultBranch, "labs.yml")
			if errors.Is(err, github.ErrNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			installable[i] = true
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	var out github.Repositories
	for i, repo := range repos {
		if installable[i] {
			out = append(out, repo)
		}
	}
	return out, nil
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List labs that can be installed",
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			Name	Description
			{{range .}}{{.Name}}	{{.Description}}
			{{end}}
			`),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repositories, err := installableRepos(ctx)
			if err != nil {
				return err
			}
			var info []labsMeta
			for _, v := range repositories {
				description := v.Description
				if len(description) > 50 {
					description = description[:50] + "..."
				}
				info = append(info, labsMeta{
					Name:        v.Name,
					Description: description,
					License:     v.License.Name,
				})
			}
			return cmdio.Render(ctx, info)
		},
	}
}
