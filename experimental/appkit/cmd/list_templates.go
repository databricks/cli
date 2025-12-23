package appkit

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

type templateInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// builtinTemplates lists the available built-in templates.
var builtinTemplates = []templateInfo{
	{
		Name:        "appkit",
		Description: "Full-stack TypeScript template with React, and Tailwind CSS",
	},
}

func newListTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-templates",
		Short: "List available AppKit templates",
		Long: `List available AppKit templates.

Shows built-in templates that can be used with 'databricks experimental appkit create'.
You can also use custom templates by providing a local path with --template.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return cmdio.Render(ctx, builtinTemplates)
		},
	}

	return cmd
}
