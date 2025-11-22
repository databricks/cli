package dresources

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

type ResourceSecretScopeAcls struct {
	client *databricks.WorkspaceClient
}

type SecretScopeAclsState struct {
	ScopeName string              `json:"scope_name"`
	Acls      []workspace.AclItem `json:"acls,omitempty"`
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
	acls := make([]workspace.AclItem, 0, sliceValue.Len())
	for i := range sliceValue.Len() {
		elem := sliceValue.Index(i).Interface().(resources.SecretScopePermission)
		acl := workspace.AclItem{
			Permission: workspace.AclPermission(elem.Level),
			Principal:  "",
		}
		if elem.UserName != "" {
			acl.Principal = elem.UserName
		} else if elem.GroupName != "" {
			acl.Principal = elem.GroupName
		} else if elem.ServicePrincipalName != "" {
			acl.Principal = elem.ServicePrincipalName
		}
		acls = append(acls, acl)
	}

	// Sort ACLs by principal for deterministic ordering
	slices.SortFunc(acls, func(a, b workspace.AclItem) int {
		return strings.Compare(a.Principal, b.Principal)
	})

	return &structvar.StructVar{
		Value: &SecretScopeAclsState{
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

func (r *ResourceSecretScopeAcls) DoRead(ctx context.Context, id string) (*SecretScopeAclsState, error) {
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

	return &SecretScopeAclsState{
		ScopeName: id,
		Acls:      currentAcls,
	}, nil
}

func (r *ResourceSecretScopeAcls) RemapState(remote *SecretScopeAclsState) *SecretScopeAclsState {
	return remote
}

func (r *ResourceSecretScopeAcls) DoCreate(ctx context.Context, state *SecretScopeAclsState) (string, *SecretScopeAclsState, error) {
	scopeName, err := r.setACLs(ctx, state.ScopeName, state.Acls)
	if err != nil {
		return "", nil, err
	}
	return scopeName, nil, nil
}

func (r *ResourceSecretScopeAcls) DoUpdate(ctx context.Context, id string, state *SecretScopeAclsState) (*SecretScopeAclsState, error) {
	_, err := r.setACLs(ctx, state.ScopeName, state.Acls)
	return nil, err
}

func (r *ResourceSecretScopeAcls) DoUpdateWithID(ctx context.Context, oldID string, state *SecretScopeAclsState) (string, *SecretScopeAclsState, error) {
	// Use state.ScopeName instead of oldID because when the parent scope is recreated,
	// state.ScopeName will have the new (resolved) scope name, while oldID still has the old name
	scopeName, err := r.setACLs(ctx, state.ScopeName, state.Acls)
	if err != nil {
		return "", nil, err
	}
	return scopeName, nil, nil
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
func (r *ResourceSecretScopeAcls) setACLs(ctx context.Context, scopeName string, desiredAcls []workspace.AclItem) (string, error) {
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
		desired[perm.Principal] = perm.Permission
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

	// Execute all operations in parallel using errgroup
	g, ctx := errgroup.WithContext(ctx)

	// Set ACLs in parallel
	for _, acl := range toSet {
		g.Go(func() error {
			if err := r.client.Secrets.PutAcl(ctx, acl); err != nil {
				return fmt.Errorf("failed to set ACL %v for principal %q: %w", acl, acl.Principal, err)
			}
			return nil
		})
	}

	// Delete ACLs in parallel
	for _, acl := range toDelete {
		g.Go(func() error {
			err := r.client.Secrets.DeleteAcl(ctx, acl)
			// Ignore not found errors for ACLs.
			if errors.Is(err, apierr.ErrNotFound) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to delete ACL %v for principal %q: %w", acl, acl.Principal, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return "", err
	}

	return scopeName, nil
}
