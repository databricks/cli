package tui

import "github.com/spf13/cobra"

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Test terminal UI components (spinners, prompts, etc.)",
	}

	cmd.AddCommand(
		newAskCmd(),
		newAskYesOrNoCmd(),
		newColorsCmd(),
		newHuhCmd(),
		newKeysCmd(),
		newPagerCmd(),
		newProfileCmd(),
		newPromptCmd(),
		newRunSelectCmd(),
		newSecretCmd(),
		newSelectCmd(),
		newSelectOrderedCmd(),
		newSpinnerCmd(),
		newTableCmd(),
	)

	return cmd
}
