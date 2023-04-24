package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:    "api",
	Short:  "Perform Databricks API call",
	Hidden: true,
}

func requestBody(arg string) (any, error) {
	if arg == "" {
		return nil, nil
	}

	// Load request from file if it starts with '@' (like curl).
	if arg[0] == '@' {
		path := arg[1:]
		buf, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", path, err)
		}
		return buf, nil
	}

	return arg, nil
}

func makeCommand(method string) *cobra.Command {
	var bodyArgument string

	command := &cobra.Command{
		Use:   strings.ToLower(method),
		Args:  cobra.ExactArgs(1),
		Short: fmt.Sprintf("Perform %s request", method),
		RunE: func(cmd *cobra.Command, args []string) error {
			var path = args[0]
			var response any

			request, err := requestBody(bodyArgument)
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
			err = api.Do(cmd.Context(), method, path, request, &response)
			if err != nil {
				return err
			}
			return cmdio.Render(cmd.Context(), response)
		},
	}

	command.Flags().StringVar(&bodyArgument, "body", "", "Request body")
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
