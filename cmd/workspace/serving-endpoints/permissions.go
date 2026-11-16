package serving_endpoints

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

const permissionsNameHelp = `
      You can also pass the endpoint NAME; it is resolved to the ID with
      "serving-endpoints get".`

const (
	// endpointIDLength is the length of a serving endpoint ID (32 hex characters).
	endpointIDLength = 32
	// rootEndpointID is the ID of the root object that endpoints inherit permissions from.
	rootEndpointID = "root"
)

// isEndpointID reports whether s is a value the permissions API accepts as an
// endpoint ID: "root" or 32 hex characters, matching the server's validation.
func isEndpointID(s string) bool {
	if s == rootEndpointID {
		return true
	}
	if len(s) != endpointIDLength {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

// resolveEndpointID returns the ID of the endpoint named arg. The permissions
// API only accepts endpoint IDs, and anything else fails with "is not a valid
// Inference Endpoint ID", but users and agents know endpoints by name. Values
// shaped like an ID are passed through unchanged, as is any value that doesn't
// resolve, so the API's own error is still what the user sees.
func resolveEndpointID(ctx context.Context, arg string) string {
	if isEndpointID(arg) {
		return arg
	}
	w := cmdctx.WorkspaceClient(ctx)
	endpoint, err := w.ServingEndpoints.GetByName(ctx, arg)
	if err != nil || endpoint.Id == "" {
		log.Debugf(ctx, "could not resolve serving endpoint name %q to an ID: %v", arg, err)
		return arg
	}
	return endpoint.Id
}

// permissionsOverride lets a permissions command take an endpoint name as well
// as an ID. It's generic over the request type of each permissions command.
func permissionsOverride[T any](cmd *cobra.Command, _ T) {
	cmd.Long += permissionsNameHelp
	generatedRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			args = append([]string{resolveEndpointID(cmd.Context(), args[0])}, args[1:]...)
		}
		return generatedRunE(cmd, args)
	}
}

func init() {
	getPermissionsOverrides = append(getPermissionsOverrides, permissionsOverride[*serving.GetServingEndpointPermissionsRequest])
	setPermissionsOverrides = append(setPermissionsOverrides, permissionsOverride[*serving.ServingEndpointPermissionsRequest])
	updatePermissionsOverrides = append(updatePermissionsOverrides, permissionsOverride[*serving.ServingEndpointPermissionsRequest])
}
