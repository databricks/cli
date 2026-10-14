package dresources

import (
	"context"
	"fmt"
	"github.com/databricks/cli/libs/apierrs"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	secrets "github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/utils"
	sdksecrets "github.com/databricks/sdk-go/secrets/v1"
)

type ResourceSecretScopeAcls struct {
	client *sdkclients.Clients
}

type SecretScopeAclsState struct {
	ScopeName string            `json:"scope_name"`
	Acls      []secrets.AclItem `json:"acls,omitempty"`
}

func (*ResourceSecretScopeAcls) New(client *sdkclients.Clients) *ResourceSecretScopeAcls {
	return &ResourceSecretScopeAcls{client: client}
}

func (*ResourceSecretScopeAcls) PrepareState(s *SecretScopeAclsState) *SecretScopeAclsState {
	return s
}

func (*ResourceSecretScopeAcls) PrepareInputConfig(inputConfig *[]resources.SecretScopePermission, resourceKey string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(resourceKey, ".permissions")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .permissions", resourceKey)
	}

	acls := make([]secrets.AclItem, 0, len(*inputConfig))
	for _, elem := range *inputConfig {
		acl := secrets.AclItem{
			Permission: secrets.AclPermission(elem.Level),
			Principal:  nil,
		}
		if elem.UserName != "" {
			acl.Principal = new(elem.UserName)
		} else if elem.GroupName != "" {
			acl.Principal = new(elem.GroupName)
		} else if elem.ServicePrincipalName != "" {
			acl.Principal = new(elem.ServicePrincipalName)
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

func aclItemKey(x secrets.AclItem) (string, string) {
	return "principal", utils.Deref(x.Principal)
}

func (*ResourceSecretScopeAcls) KeyedSlices() map[string]any {
	return map[string]any{
		"acls": aclItemKey,
	}
}

func (r *ResourceSecretScopeAcls) DoRead(ctx context.Context, id string) (*SecretScopeAclsState, error) {
	// id is the scope name
	currentAcls, err := r.listACLs(ctx, id)
	if err != nil {
		return nil, err
	}

	return &SecretScopeAclsState{
		ScopeName: id,
		Acls:      currentAcls,
	}, nil
}

func (r *ResourceSecretScopeAcls) DoCreate(ctx context.Context, state *SecretScopeAclsState) (string, *SecretScopeAclsState, error) {
	err := r.setACLs(ctx, state.ScopeName, state.Acls)
	if err != nil {
		return "", nil, err
	}
	return state.ScopeName, nil, nil
}

// We implement DoUpdateWithId to ensure that the updated ID gets recorded in state.
func (r *ResourceSecretScopeAcls) DoUpdateWithID(ctx context.Context, id string, state *SecretScopeAclsState, _ *PlanEntry) (string, *SecretScopeAclsState, error) {
	err := r.setACLs(ctx, state.ScopeName, state.Acls)
	if err != nil {
		return "", nil, err
	}
	return state.ScopeName, nil, nil
}

func (r *ResourceSecretScopeAcls) DoUpdate(ctx context.Context, id string, state *SecretScopeAclsState, entry *PlanEntry) (*SecretScopeAclsState, error) {
	_, _, err := r.DoUpdateWithID(ctx, id, state, entry)
	return nil, err
}

// ResourceSecretScopeAcls intentionally implements no DoDelete: removing ACLs is a
// no-op, to match the behavior for permissions and grants. Deleting the resource is a
// state-only cleanup (see PlanEntry.StateOnly).

func (r *ResourceSecretScopeAcls) listACLs(ctx context.Context, scopeName string) ([]secrets.AclItem, error) {
	resp, err := r.client.Secrets.ListAcls(ctx, sdksecrets.ListAclsRequest{Scope: new(scopeName)})
	if err != nil {
		return nil, err
	}
	acls, err := secrets.ListAclsResponseFromSDK(resp)
	if err != nil {
		return nil, err
	}
	return acls.Items, nil
}

// setACLs reconciles the desired ACLs with the current state
func (r *ResourceSecretScopeAcls) setACLs(ctx context.Context, scopeName string, desiredAcls []secrets.AclItem) error {
	currentAcls, err := r.listACLs(ctx, scopeName)
	if err != nil {
		return fmt.Errorf("failed to list current ACLs: %w", err)
	}

	desired := make(map[string]secrets.AclPermission)
	for _, perm := range desiredAcls {
		desired[utils.Deref(perm.Principal)] = perm.Permission
	}

	current := make(map[string]secrets.AclPermission)
	for _, acl := range currentAcls {
		current[utils.Deref(acl.Principal)] = acl.Permission
	}

	type putACL struct {
		principal  string
		permission secrets.AclPermission
	}
	var toSet []putACL
	var toDelete []string

	for principal, permission := range desired {
		if currentPerm, exists := current[principal]; !exists || currentPerm != permission {
			toSet = append(toSet, putACL{principal: principal, permission: permission})
		}
	}

	for principal := range current {
		if _, exists := desired[principal]; !exists {
			toDelete = append(toDelete, principal)
		}
	}

	for _, acl := range toSet {
		_, err := r.client.Secrets.PutAcl(ctx, sdksecrets.PutAclRequest{
			Scope:      new(scopeName),
			Principal:  new(acl.principal),
			Permission: acl.permission,
		})
		if err != nil {
			return fmt.Errorf("failed to set ACL {%s %s %s} for principal %q: %w", acl.permission, acl.principal, scopeName, acl.principal, err)
		}
	}

	for _, principal := range toDelete {
		_, err := r.client.Secrets.DeleteAcl(ctx, sdksecrets.DeleteAclRequest{
			Scope:     new(scopeName),
			Principal: new(principal),
		})
		if apierrs.Is(err, apierrs.ErrNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to delete ACL {%s %s} for principal %q: %w", principal, scopeName, principal, err)
		}
	}

	return nil
}
