package labs

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newInstalledCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "installed",
		Short: "List all installed labs",
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			Name	Description	Version
			{{range .Projects}}{{.Name}}	{{.Description}}	{{.Version}}
			{{end}}
			`),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			type installedProject struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Version     string `json:"version"`
			}
			projects, err := project.Installed(ctx)
			if err != nil {
				return err
			}
			var info struct {
				Projects []installedProject `json:"projects"`
			}
			for _, v := range projects {
				description := v.Description
				if len(description) > 50 {
					description = description[:50] + "..."
				}
				version, err := v.InstalledVersion(ctx)
				if err != nil {
					return fmt.Errorf("%s: %w", v.Name, err)
				}
				info.Projects = append(info.Projects, installedProject{
					Name:        v.Name,
					Description: description,
					Version:     version.Version,
				})
			}
			if len(info.Projects) == 0 {
				return errors.New("no projects installed")
			}
			return cmdio.Render(ctx, info)
		},
	}
}
