package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/databricks-sdk-go/databricks"
	"github.com/databricks/databricks-sdk-go/databricks/client"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Perform Databricks API call",
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

			api, err := client.New(&databricks.Config{})
			if err != nil {
				return err
			}
			err = api.Do(cmd.Context(), method, path, request, &response)
			if err != nil {
				return err
			}

			if response != nil {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				enc.Encode(response)
			}

			return nil
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
