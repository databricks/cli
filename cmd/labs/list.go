package labs

import (
	"context"
	"slices"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

const (
	labsOrg = "databrickslabs"

	// installableTopic is the GitHub repository topic that labs maintainers add to
	// projects installable via `databricks labs install`. The repositories API
	// returns topics inline, so filtering on it costs no extra requests.
	installableTopic = "databricks-cli-installable"
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
// install. Most repositories don't ship a labs.yml manifest (e.g. libraries
// published to package indexes); maintainers tag the installable ones with
// installableTopic so the listing doesn't advertise projects that fail to install.
func installableRepos(ctx context.Context) (github.Repositories, error) {
	repos, err := allRepos(ctx)
	if err != nil {
		return nil, err
	}
	var out github.Repositories
	for _, repo := range repos {
		if repo.IsArchived || repo.IsFork {
			continue
		}
		if slices.Contains(repo.Topics, installableTopic) {
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
