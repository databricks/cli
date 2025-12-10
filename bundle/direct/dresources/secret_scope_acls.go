package dresources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type ResourceSecretScopeAcls struct {
	client *databricks.WorkspaceClient
}

type SecretScopeAclsState struct {
	ScopeName string              `json:"scope_name"`
	Acls      []workspace.AclItem `json:"acls,omitempty"`
}

func PrepareSecretScopeAclsInputConfig(inputConfig []resources.SecretScopePermission, node string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(node, ".permissions")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .permissions", node)
	}

	acls := make([]workspace.AclItem, 0, len(inputConfig))
	for _, elem := range inputConfig {
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

func aclItemKey(x workspace.AclItem) (string, string) {
	return "principal", x.Principal
}

func (*ResourceSecretScopeAcls) KeyedSlices() map[string]any {
	return map[string]any{
		"acls": aclItemKey,
	}
}

func (r *ResourceSecretScopeAcls) DoRead(ctx context.Context, id string) (*SecretScopeAclsState, error) {
	// id is the scope name
	currentAcls, err := r.client.Secrets.ListAclsAll(ctx, workspace.ListAclsRequest{
		Scope: id,
	})
	if err != nil {
		return nil, err
	}

	return &SecretScopeAclsState{
		ScopeName: id,
		Acls:      currentAcls,
	}, nil
}

func (r *ResourceSecretScopeAcls) RemapState(remote *SecretScopeAclsState) *SecretScopeAclsState {
	return remote
}

func (r *ResourceSecretScopeAcls) DoCreate(ctx context.Context, state *SecretScopeAclsState) (string, *SecretScopeAclsState, error) {
	err := r.setACLs(ctx, state.ScopeName, state.Acls)
	if err != nil {
		return "", nil, err
	}
	return state.ScopeName, nil, nil
}

// We implement DoUpdateWithId to ensure that the updated ID gets recorded in state.
func (r *ResourceSecretScopeAcls) DoUpdateWithID(ctx context.Context, id string, state *SecretScopeAclsState) (string, *SecretScopeAclsState, error) {
	err := r.setACLs(ctx, state.ScopeName, state.Acls)
	if err != nil {
		return "", nil, err
	}
	return state.ScopeName, nil, nil
}

func (r *ResourceSecretScopeAcls) DoUpdate(ctx context.Context, id string, state *SecretScopeAclsState, changes *deployplan.Changes) (*SecretScopeAclsState, error) {
	_, _, err := r.DoUpdateWithID(ctx, id, state)
	return nil, err
}

func (r *ResourceSecretScopeAcls) FieldTriggers(isLocal bool) map[string]deployplan.ActionType {
	// When scope name changes, we need  a DoUpdateWithID trigger. This is necessary so that subsequent
	// DoRead operations use the correct ID and we do not end up with a persistent drift.
	return map[string]deployplan.ActionType{
		"scope_name": deployplan.ActionTypeUpdateWithID,
	}
}

// Removing ACLs is a no-op, to match the behavior for permissions and grants.
func (r *ResourceSecretScopeAcls) DoDelete(ctx context.Context, id string) error {
	return nil
}

// setACLs reconciles the desired ACLs with the current state
func (r *ResourceSecretScopeAcls) setACLs(ctx context.Context, scopeName string, desiredAcls []workspace.AclItem) error {
	// Get current ACLs
	currentAcls, err := r.client.Secrets.ListAclsAll(ctx, workspace.ListAclsRequest{
		Scope: scopeName,
	})
	if err != nil {
		return fmt.Errorf("failed to list current ACLs: %w", err)
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

	// Set ACLs. The service returns inconsistent results for parallel API calls. That's why we do them sequentially
	// here to maintain correctness.
	for _, acl := range toSet {
		err := r.client.Secrets.PutAcl(ctx, acl)
		if err != nil {
			return fmt.Errorf("failed to set ACL %v for principal %q: %w", acl, acl.Principal, err)
		}
	}

	// Delete ACLs
	for _, acl := range toDelete {
		err := r.client.Secrets.DeleteAcl(ctx, acl)
		// Ignore not found errors for ACLs.
		if errors.Is(err, apierr.ErrNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to delete ACL %v for principal %q: %w", acl, acl.Principal, err)
		}
	}

	return nil
}
