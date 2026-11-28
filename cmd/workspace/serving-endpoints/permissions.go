package serving_endpoints

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

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
// shaped like an ID are passed through unchanged. Any other value can't be an
// ID the API accepts, so if the name lookup fails we return the lookup error,
// which says what is actually wrong (a typo, a missing permission), instead of
// sending the value on and getting the API's ID error.
func resolveEndpointID(ctx context.Context, arg string) (string, error) {
	if isEndpointID(arg) {
		return arg, nil
	}
	w := cmdctx.WorkspaceClient(ctx)
	endpoint, err := w.ServingEndpoints.GetByName(ctx, arg)
	if err != nil {
		return "", fmt.Errorf("%q is not a serving endpoint ID, and looking it up as an endpoint name failed: %w", arg, err)
	}
	if endpoint.Id == "" {
		// Endpoints without an ID, such as Foundation Model API pay-per-token
		// endpoints, have no permissions object of their own.
		return "", fmt.Errorf("serving endpoint %q has no endpoint ID, so its permissions can't be read or changed with this command", arg)
	}
	return endpoint.Id, nil
}

// permissionsOverride lets a permissions command take an endpoint name as well
// as an ID. It's generic over the request type of each permissions command.
func permissionsOverride[T any](cmd *cobra.Command, _ T) {
	cmd.Use = strings.Replace(cmd.Use, "SERVING_ENDPOINT_ID", "NAME_OR_ID", 1)
	cmd.Long = strings.Replace(cmd.Long,
		"SERVING_ENDPOINT_ID: The serving endpoint for which to get or manage permissions.",
		"NAME_OR_ID: The name or ID of the serving endpoint for which to get or manage\n      permissions. A name is resolved to the ID with \"serving-endpoints get\".", 1)
	generatedRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			id, err := resolveEndpointID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			args = append([]string{id}, args[1:]...)
		}
		return generatedRunE(cmd, args)
	}
}

func init() {
	getPermissionsOverrides = append(getPermissionsOverrides, permissionsOverride[*serving.GetServingEndpointPermissionsRequest])
	setPermissionsOverrides = append(setPermissionsOverrides, permissionsOverride[*serving.ServingEndpointPermissionsRequest])
	updatePermissionsOverrides = append(updatePermissionsOverrides, permissionsOverride[*serving.ServingEndpointPermissionsRequest])
}
