// Package mcp holds the logic behind `databricks aitools mcp`: discovery of Databricks managed
// MCP services, server naming, and service and sign-in URLs.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// mcpServicePrefix is the resource-name prefix of an MCP service.
const mcpServicePrefix = "mcp-services/"

// oauthProviderOption is the source connection option that marks a service as needing sign-in.
const oauthProviderOption = "oauth_provider"

// Service is an MCP service in a schema, as the API reports it.
type Service struct {
	// FullName is catalog.schema.leaf.
	FullName string
	Leaf     string
	// Comment is the user-provided description from the API.
	Comment string
	// SignInRequired is true when the service's source connection has an oauth_provider, so
	// each user signs in to the provider once.
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
		services = append(services, newService(&list[i]))
	}
	return services, nil
}

// GetService returns one MCP service by full name, with its sign-in requirement. The API
// answers NOT_FOUND (apierr.ErrNotFound) for a service the profile cannot see.
func GetService(ctx context.Context, w *databricks.WorkspaceClient, fullName string) (Service, error) {
	svc, err := w.AiGateway.GetMcpService(ctx, catalog.GetMcpServiceRequest{Name: mcpServicePrefix + fullName})
	if err != nil {
		return Service{}, err
	}
	return newService(svc), nil
}

// ListErrorKind names a listing failure.
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

// Error names the likely cause and keeps the server's message, because the SDK maps
// different failures to the same sentinel (any 404, or a private link 403).
func (e *ListError) Error() string {
	switch e.Kind {
	case ListErrLocationNotFound:
		return fmt.Sprintf("%s is not a schema visible to this profile: %v", e.Location, e.Err)
	case ListErrPermissionDenied:
		// Listing needs these two grants. Services the caller has no privilege on are
		// left out of the result instead of failing it.
		return fmt.Sprintf("cannot list MCP services in %s (listing needs USE_CATALOG on the catalog and USE_SCHEMA on the schema): %v", e.Location, e.Err)
	case ListErrRateLimited:
		return fmt.Sprintf("MCP service listing in %s was rate limited; retry in a minute: %v", e.Location, e.Err)
	}
	return fmt.Sprintf("cannot list MCP services in %s: %v", e.Location, e.Err)
}

// Unwrap returns the SDK error.
func (e *ListError) Unwrap() error {
	return e.Err
}

// listServices reads every page of the services in location, with source connections.
func listServices(ctx context.Context, w *databricks.WorkspaceClient, location string) ([]catalog.McpService, error) {
	return w.AiGateway.ListMcpServicesAll(ctx, catalog.ListMcpServicesRequest{
		Parent: "schemas/" + location,
		View:   catalog.ListMcpServicesRequestViewFull,
	})
}

// unrelatedErrorCodes are failures that look like a missing schema (404) or missing grants
// (403) but are about the workspace or network: the API is not available, or private link
// blocked the request. They keep the server's message and get no location or grants hint.
var unrelatedErrorCodes = []string{"ENDPOINT_NOT_FOUND", "FEATURE_DISABLED", "PRIVATE_LINK_VALIDATION_ERROR"}

// mapListError classifies a listing failure. Other errors are returned with context only.
func mapListError(location string, err error) error {
	apiErr, isAPIErr := errors.AsType[*apierr.APIError](err)

	var kind ListErrorKind
	switch {
	case isAPIErr && slices.Contains(unrelatedErrorCodes, apiErr.ErrorCode):
		// Not about the schema or grants: leave it unclassified.
	case errors.Is(err, apierr.ErrNotFound):
		kind = ListErrLocationNotFound
	case errors.Is(err, apierr.ErrPermissionDenied):
		kind = ListErrPermissionDenied
	case errors.Is(err, apierr.ErrTooManyRequests):
		kind = ListErrRateLimited
	}
	if kind == "" {
		return fmt.Errorf("cannot list MCP services in %s: %w", location, err)
	}
	return &ListError{Kind: kind, Location: location, Err: err}
}

// isDeletedSource reports whether the service's source connection was deleted.
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

// newService builds a Service from an API service. The API returns lowercase names whatever
// casing the caller used for the location, so the leaf and the system.ai check use the
// returned name.
func newService(svc *catalog.McpService) Service {
	fullName := strings.TrimPrefix(svc.Name, mcpServicePrefix)
	leaf := LeafOf(fullName)
	provider := oauthProvider(svc)

	s := Service{
		FullName:       fullName,
		Leaf:           leaf,
		Comment:        svc.Comment,
		SignInRequired: provider != "",
		OAuthProvider:  provider,
	}
	if known, ok := knownSystemAI[leaf]; ok && inSystemAI(fullName) {
		s.Known = &known
	}
	return s
}
