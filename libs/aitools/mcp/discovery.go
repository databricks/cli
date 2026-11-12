// Package mcp holds the logic behind `databricks aitools mcp`: discovery of Databricks managed
// MCP services, server naming, and service and sign-in URLs.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// mcpServicePrefix is the resource-name prefix of an MCP service.
const mcpServicePrefix = "mcp-services/"

// oauthProviderOption is the source connection option that marks a service as needing sign-in.
const oauthProviderOption = "oauth_provider"

// listPageSize is the largest page the API allows for view=FULL.
const listPageSize = 200

// Service is an MCP service in a schema, as the API reports it.
type Service struct {
	// FullName is catalog.schema.leaf.
	FullName string
	Leaf     string
	// Comment is the user-provided description from the API.
	Comment string
	// SignInRequired is true for services backed by a per-user connection.
	SignInRequired bool
	// OAuthProvider is the connection's oauth_provider option, empty when SignInRequired is false.
	OAuthProvider string
	// Known is the CLI's description of the service. It is nil outside system.ai and for
	// system.ai services missing from the table.
	Known *KnownService
}

// Discover lists the MCP services in location (catalog.schema). Services whose source
// connection was deleted are dropped. Listing failures the CLI names are returned as *ListError.
func Discover(ctx context.Context, w *databricks.WorkspaceClient, location string) ([]Service, error) {
	list, err := listServices(ctx, w, location)
	if err != nil {
		return nil, mapListError(location, err)
	}

	services := make([]Service, 0, len(list))
	for i := range list {
		if isDeletedSource(&list[i]) {
			continue
		}
		services = append(services, newService(location, &list[i]))
	}
	return services, nil
}

// ListErrorKind names a listing failure. Each value is the telemetry category name, so a
// caller can map it to telemetry without another table.
type ListErrorKind string

const (
	ListErrLocationNotFound ListErrorKind = "MCP_LOCATION_NOT_FOUND"
	ListErrPermissionDenied ListErrorKind = "MCP_PERMISSION_DENIED"
	ListErrRateLimited      ListErrorKind = "RATE_LIMITED"
)

// ListError is a classified failure to list MCP services. It wraps the SDK error, so
// errors.Is(err, apierr.ErrNotFound) and similar still work.
type ListError struct {
	Kind     ListErrorKind
	Location string
	Err      error
}

func (e *ListError) Error() string {
	switch e.Kind {
	case ListErrLocationNotFound:
		return e.Location + " is not a schema visible to this profile"
	case ListErrPermissionDenied:
		return fmt.Sprintf("cannot list MCP services in %s; needs USE on the schema and EXECUTE on the MCP service", e.Location)
	default:
		return "MCP service listing was rate limited; retry in a minute"
	}
}

func (e *ListError) Unwrap() error {
	return e.Err
}

func listServices(ctx context.Context, w *databricks.WorkspaceClient, location string) ([]catalog.McpService, error) {
	return listing.ToSlice(ctx, w.AiGateway.ListMcpServices(ctx, catalog.ListMcpServicesRequest{
		Parent:   "schemas/" + location,
		View:     catalog.ListMcpServicesRequestViewFull,
		PageSize: listPageSize,
	}))
}

// mapListError classifies a listing failure. Other errors are returned with context only.
func mapListError(location string, err error) error {
	var kind ListErrorKind
	switch {
	case errors.Is(err, apierr.ErrNotFound):
		kind = ListErrLocationNotFound
	case errors.Is(err, apierr.ErrPermissionDenied):
		kind = ListErrPermissionDenied
	case errors.Is(err, apierr.ErrTooManyRequests):
		kind = ListErrRateLimited
	default:
		return fmt.Errorf("cannot list MCP services in %s: %w", location, err)
	}
	return &ListError{Kind: kind, Location: location, Err: err}
}

func isDeletedSource(svc *catalog.McpService) bool {
	return svc.Config != nil && svc.Config.SourceConnection != nil && svc.Config.SourceConnection.IsDeleted
}

// oauthProvider returns the OAuth provider of a service's source connection, or "" when it
// has none. A non-empty provider means the user must sign in before the service is usable.
func oauthProvider(svc *catalog.McpService) string {
	if svc.Config == nil || svc.Config.SourceConnection == nil {
		return ""
	}
	return svc.Config.SourceConnection.Options[oauthProviderOption]
}

func newService(location string, svc *catalog.McpService) Service {
	fullName := strings.TrimPrefix(svc.Name, mcpServicePrefix)
	leaf := strings.TrimPrefix(fullName, location+".")
	provider := oauthProvider(svc)

	s := Service{
		FullName:       fullName,
		Leaf:           leaf,
		Comment:        svc.Comment,
		SignInRequired: provider != "",
		OAuthProvider:  provider,
	}
	if known, ok := knownSystemAI[leaf]; ok && location == systemAILocation {
		s.Known = &known
	}
	return s
}
