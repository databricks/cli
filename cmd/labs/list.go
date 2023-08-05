package labs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

type labsMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	License     string `json:"license"`
}

func httpCall(ctx context.Context, url string, response any) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("github request failed: %s", res.Status)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(raw, response)
	if err != nil {
		return err
	}
	return nil
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
			var repositories []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Fork        bool   `json:"fork"`
				Arcived     bool   `json:"archived"`
				License     struct {
					Name string `json:"name"`
				} `json:"license"`
			}
			err := httpCall(ctx,
				"https://api.github.com/users/databrickslabs/repos",
				&repositories)
			if err != nil {
				return err
			}
			info := []labsMeta{}
			for _, v := range repositories {
				if v.Arcived {
					continue
				}
				if v.Fork {
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
