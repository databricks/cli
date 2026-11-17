package tui

import (
	"fmt"
	"os"
	"strconv"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newHuhCmd() *cobra.Command {
	var field string
	cmd := &cobra.Command{
		Use:   "huh",
		Short: "Run one huh form field with the appkit theme",
		Long: `Run one huh form field with the appkit theme and print the answer.

The field is built here, not by a libs/apps/prompt helper: those helpers each
set their own title, options and validation. Only prompt.AppkitTheme is shared,
so this checks the theme, not the production prompt flows.`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch field {
			case "input", "select", "confirm", "multiselect":
				return nil
			default:
				return fmt.Errorf("--field must be one of input, select, confirm, multiselect, got %q", field)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			theme := prompt.AppkitTheme()
			features := databricksFeatures[:5]
			options := make([]huh.Option[string], 0, len(features))
			for _, name := range features {
				options = append(options, huh.NewOption(name, name))
			}

			var (
				text   string
				choice string
				yes    bool
				picked []string
				group  huh.Field
			)
			switch field {
			case "input":
				group = huh.NewInput().Title("Enter a value").Value(&text)
			case "select":
				group = huh.NewSelect[string]().Title("Pick an item").Options(options...).Value(&choice)
			case "confirm":
				group = huh.NewConfirm().Title("Continue?").Value(&yes)
			case "multiselect":
				group = huh.NewMultiSelect[string]().Title("Pick items").Options(options...).Value(&picked)
			}

			// Run the field the way the prompts in libs/apps/prompt do, so the
			// form around it (no help footer, the theme on the field only) is
			// the one production draws.
			if err := cmdio.RunHuhField(ctx, group.WithTheme(theme)); err != nil {
				return err
			}

			var answer string
			switch field {
			case "input":
				answer = text
			case "select":
				answer = choice
			case "confirm":
				answer = strconv.FormatBool(yes)
			case "multiselect":
				answer = fmt.Sprint(picked)
			}
			cmdio.LogString(ctx, "answer: "+answer)

			// The appkit theme is a huh.ThemeFunc, so its colours depend on the
			// background, queried once here for the probe. The blurred button is
			// the colour the theme picks for that background.
			dark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
			cmdio.LogString(ctx, fmt.Sprintf("dark_background=%t", dark))
			fg := theme.Theme(dark).Focused.BlurredButton.GetForeground()
			r, g, b, _ := fg.RGBA()
			cmdio.LogString(ctx, fmt.Sprintf("blurred_button=#%02x%02x%02x (resolved, dark_background=%t)", r>>8, g>>8, b>>8, dark))
			return nil
		},
	}
	cmd.Flags().StringVar(&field, "field", "input", "field type: input, select, confirm or multiselect")
	return cmd
}
