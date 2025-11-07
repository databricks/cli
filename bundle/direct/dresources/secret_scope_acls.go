package dresources

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

type ResourceSecretScopeAcls struct {
	client *databricks.WorkspaceClient
}

type SecretScopeAclsState struct {
	ScopeName string                            `json:"scope_name"`
	Acls      []resources.SecretScopePermission `json:"acls,omitempty"`
}

func PrepareSecretScopeAclsInputConfig(inputConfig any, node string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(node, ".permissions")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .permissions", node)
	}

	// Use reflection to get the slice from the pointer
	rv := reflect.ValueOf(inputConfig)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return nil, fmt.Errorf("inputConfig must be a pointer to a slice, got: %T", inputConfig)
	}

	sliceValue := rv.Elem()

	// Convert slice to []resources.SecretScopePermission
	acls := make([]resources.SecretScopePermission, 0, sliceValue.Len())
	for i := range sliceValue.Len() {
		elem := sliceValue.Index(i).Interface().(resources.SecretScopePermission)
		acls = append(acls, elem)
	}

	// Sort ACLs by principal for deterministic ordering
	slices.SortFunc(acls, func(a, b resources.SecretScopePermission) int {
		return strings.Compare(getPrincipal(a), getPrincipal(b))
	})

	return &structvar.StructVar{
		Config: &SecretScopeAclsState{
			ScopeName: "", // Always a reference, defined in Refs below
			Acls:      acls,
		},
		Refs: map[string]string{
			"scope_name": "${" + baseNode + ".name}",
		},
	}, nil
}

func (*ResourceSecretScopeAcls) New(client *databricks.WorkspaceClient) *ResourceSecretScopeAcls {
	return &ResourceSecretScopeAcls{client: client}
}

func (*ResourceSecretScopeAcls) PrepareState(s *SecretScopeAclsState) *SecretScopeAclsState {
	return s
}

func (r *ResourceSecretScopeAcls) DoRefresh(ctx context.Context, id string) (*SecretScopeAclsState, error) {
	// id is the scope name
	currentAcls, err := r.client.Secrets.ListAclsAll(ctx, workspace.ListAclsRequest{
		Scope: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ACLs: %w", err)
	}

	// Sort ACLs by principal for deterministic ordering
	slices.SortFunc(currentAcls, func(a, b workspace.AclItem) int {
		return strings.Compare(a.Principal, b.Principal)
	})

	acls := make([]resources.SecretScopePermission, 0, len(currentAcls))
	for _, acl := range currentAcls {
		perm := resources.SecretScopePermission{
			Level:                resources.SecretScopePermissionLevel(acl.Permission),
			UserName:             "",
			ServicePrincipalName: "",
			GroupName:            "",
		}

		// Set the appropriate principal field
		if strings.Contains(acl.Principal, "@") {
			perm.UserName = acl.Principal
		} else {
			// Assume it's a group if it doesn't look like an email
			perm.GroupName = acl.Principal
		}

		acls = append(acls, perm)
	}

	return &SecretScopeAclsState{
		ScopeName: id,
		Acls:      acls,
	}, nil
}

func (r *ResourceSecretScopeAcls) RemapState(remote *SecretScopeAclsState) *SecretScopeAclsState {
	return remote
}

func (r *ResourceSecretScopeAcls) DoCreate(ctx context.Context, state *SecretScopeAclsState) (string, error) {
	return r.setACLs(ctx, state.ScopeName, state.Acls)
}

func (r *ResourceSecretScopeAcls) DoUpdate(ctx context.Context, id string, state *SecretScopeAclsState) error {
	// This method is required by the adapter interface, but we always use DoUpdateWithID
	// because the ID can change when the parent scope is recreated
	_, err := r.setACLs(ctx, state.ScopeName, state.Acls)
	return err
}

// TODO: We need a more general solution for this.  This is a problem for all types of resources.
func (r *ResourceSecretScopeAcls) DoUpdateWithID(ctx context.Context, oldID string, state *SecretScopeAclsState) (string, error) {
	// Use state.ScopeName instead of oldID because when the parent scope is recreated,
	// state.ScopeName will have the new (resolved) scope name, while oldID still has the old name
	return r.setACLs(ctx, state.ScopeName, state.Acls)
}

func (r *ResourceSecretScopeAcls) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"scope_name": deployplan.ActionTypeUpdateWithID, // When scope name changes, ID changes
		"acls":       deployplan.ActionTypeUpdate,       // When ACLs change, just update
	}
}

// Removing ACLs is a no-op, to match the behavior for permissions and grants.
func (r *ResourceSecretScopeAcls) DoDelete(ctx context.Context, id string) error {
	return nil
}

// setACLs reconciles the desired ACLs with the current state
func (r *ResourceSecretScopeAcls) setACLs(ctx context.Context, scopeName string, desiredAcls []resources.SecretScopePermission) (string, error) {
	// Get current ACLs
	currentAcls, err := r.client.Secrets.ListAclsAll(ctx, workspace.ListAclsRequest{
		Scope: scopeName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list current ACLs: %w", err)
	}

	// Build maps for reconciliation
	desired := make(map[string]workspace.AclPermission)
	for _, perm := range desiredAcls {
		principal := getPrincipal(perm)
		desired[principal] = workspace.AclPermission(perm.Level)
	}

	current := make(map[string]workspace.AclPermission)
	for _, acl := range currentAcls {
		current[acl.Principal] = acl.Permission
	}

	// Collect operations to perform in parallel
	var toSet []workspace.PutAcl
	var toDelete []workspace.DeleteAcl

	// Find ACLs to set (new or changed)
	for principal, permission := range desired {
		if currentPerm, exists := current[principal]; !exists || currentPerm != permission {
			toSet = append(toSet, workspace.PutAcl{
				Scope:      scopeName,
				Principal:  principal,
				Permission: permission,
			})
		}
	}

	// Find ACLs to delete
	for principal := range current {
		if _, exists := desired[principal]; !exists {
			toDelete = append(toDelete, workspace.DeleteAcl{
				Scope:     scopeName,
				Principal: principal,
			})
		}
	}

	// Sort operations for deterministic ordering
	slices.SortFunc(toSet, func(a, b workspace.PutAcl) int {
		return strings.Compare(a.Principal, b.Principal)
	})
	slices.SortFunc(toDelete, func(a, b workspace.DeleteAcl) int {
		return strings.Compare(a.Principal, b.Principal)
	})

	// Execute all operations in parallel using errgroup
	g, ctx := errgroup.WithContext(ctx)

	// Set ACLs in parallel
	for _, acl := range toSet {
		g.Go(func() error {
			if err := r.client.Secrets.PutAcl(ctx, acl); err != nil {
				return fmt.Errorf("failed to set ACL for principal %q: %w", acl.Principal, err)
			}
			return nil
		})
	}

	// Delete ACLs in parallel
	for _, acl := range toDelete {
		g.Go(func() error {
			if err := r.client.Secrets.DeleteAcl(ctx, acl); err != nil {
				return fmt.Errorf("failed to delete ACL for principal %q: %w", acl.Principal, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return "", err
	}

	return scopeName, nil
}

// getPrincipal extracts the principal from a SecretScopePermission
func getPrincipal(perm resources.SecretScopePermission) string {
	if perm.UserName != "" {
		return perm.UserName
	}
	if perm.ServicePrincipalName != "" {
		return perm.ServicePrincipalName
	}
	if perm.GroupName != "" {
		return perm.GroupName
	}
	return ""
}
