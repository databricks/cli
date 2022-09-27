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

func makeCommand(method string) *cobra.Command {
	var body string

	command := &cobra.Command{
		Use:   strings.ToLower(method),
		Args:  cobra.ExactArgs(1),
		Short: fmt.Sprintf("Perform %s request", method),
		RunE: func(cmd *cobra.Command, args []string) error {
			var request any
			var err error
			if body != "" {
				// Load request from file if it starts with '@' (like curl).
				if body[0] == '@' {
					path := body[1:]
					f, err := os.Open(path)
					if err != nil {
						return fmt.Errorf("error opening %s: %w", path, err)
					}
					defer f.Close()
					request = f
				} else {
					request = body
				}
			}

			var response any
			api := client.New(&databricks.Config{})
			err = api.Do(cmd.Context(), method, args[0], request, &response)
			if err != nil {
				return err
			}

			if response != nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(response)
			}

			return nil
		},
	}

	command.Flags().StringVar(&body, "body", "", "Request body")
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
