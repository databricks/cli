package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Perform Databricks API call",
}

func makeCommand(method string) *cobra.Command {
	var payload flags.JsonFlag

	command := &cobra.Command{
		Use:   strings.ToLower(method),
		Args:  cobra.ExactArgs(1),
		Short: fmt.Sprintf("Perform %s request", method),
		RunE: func(cmd *cobra.Command, args []string) error {
			var path = args[0]

			var request any
			err := payload.Unmarshal(&request)
			if err != nil {
				return err
			}

			cfg := &config.Config{}

			// command-line flag can specify the profile in use
			profileFlag := cmd.Flag("profile")
			if profileFlag != nil {
				cfg.Profile = profileFlag.Value.String()
			}

			api, err := client.New(cfg)
			if err != nil {
				return err
			}

			var response any
			err = api.Do(cmd.Context(), method, path, request, &response)
			if err != nil {
				return err
			}
			return cmdio.Render(cmd.Context(), response)
		},
	}

	command.Flags().Var(&payload, "json", `either inline JSON string or @path/to/file.json with request body`)
	return command
}

func init() {
	apiCmd.AddCommand(
		makeCommand(http.MethodGet),
		makeCommand(http.MethodHead),
		makeCommand(http.MethodPost),
		makeCommand(http.MethodPut),
		makeCommand(http.MethodPatch),
		makeCommand(http.MethodDelete),
	)
	root.RootCmd.AddCommand(apiCmd)
}
