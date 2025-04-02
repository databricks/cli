package labs

import (
	"context"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
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
	cache := github.NewRepositoryCache("databrickslabs", cacheDir)
	return cache.Load(ctx)
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all labs",
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			Name	Description
			{{range .}}{{.Name}}	{{.Description}}
			{{end}}
			`),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repositories, err := allRepos(ctx)
			if err != nil {
				return err
			}
			var info []labsMeta
			for _, v := range repositories {
				if v.IsArchived {
					continue
				}
				if v.IsFork {
					continue
				}
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
