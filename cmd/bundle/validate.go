package bundle

import (
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/spf13/cobra"
)

// "red":     color.RedString,
// "green":   color.GreenString,
// "blue":    color.BlueString,
// "yellow":  color.YellowString,
// "magenta": color.MagentaString,
// "cyan":    color.CyanString,

const tmpl = `
  {{ "Warning" | yellow }}: {{ .Summary }}
  {{ "At " }}{{ .Path.String | green }}
  {{ "In " }}{{ .Location.String | cyan }}
`

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Args:  root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = diags.Extend(bundle.Apply(ctx, b, phases.Initialize()))
		if err := diags.Error(); err != nil {
			return err
		}

		// Until we change up the output of this command to be a text representation,
		// we'll just output all diagnostics as debug logs.
		for _, d := range diags {
			switch d.Severity {
			case diag.Warning:
				// Make file relative to bundle root
				out, _ := filepath.Rel(b.RootPath, d.Location.File)
				d.Location.File = out
				err := cmdio.RenderWithTemplate(ctx, d, "", tmpl)
				if err != nil {
					return err
				}
			}

			// log.Debugf(cmd.Context(), "[%s]: %s", diag.Location, diag.Summary)
		}

		// buf, err := json.MarshalIndent(b.Config, "", "  ")
		// if err != nil {
		// 	return err
		// }
		// cmd.OutOrStdout().Write(buf)
		return nil
	}

	return cmd
}
