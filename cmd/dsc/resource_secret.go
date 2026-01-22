package dsc

import (
	"encoding/json"
	"reflect"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	RegisterResourceWithMetadata("Databricks.DSC/Secret", &SecretHandler{}, secretMetadata())
	RegisterResourceWithMetadata("Databricks.DSC/SecretScope", &SecretScopeHandler{}, secretScopeMetadata())
	RegisterResourceWithMetadata("Databricks.DSC/SecretAcl", &SecretAclHandler{}, secretAclMetadata())
}

// ============================================================================
// Property Descriptions (from SDK documentation)
// ============================================================================

var secretPropertyDescriptions = PropertyDescriptions{
	"key":          "A unique name to identify the secret.",
	"scope":        "The name of the scope to which the secret will be associated with.",
	"string_value": "If specified, note that the value will be stored in UTF-8 (MB4) form.",
	"bytes_value":  "If specified, value will be stored as bytes.",
}

var secretScopePropertyDescriptions = PropertyDescriptions{
	"scope":                    "A unique name to identify the scope.",
	"initial_manage_principal": "The principal that is initially granted MANAGE permission to the created scope.",
	"scope_backend_type":       "The backend type the scope will be created with (DATABRICKS or AZURE_KEYVAULT).",
	"backend_azure_keyvault":   "The metadata for the Azure KeyVault if using Azure-backed secret scope.",
}

var secretAclPropertyDescriptions = PropertyDescriptions{
	"scope":      "The name of the scope to apply permissions to.",
	"principal":  "The principal (user or group) to apply permissions to.",
	"permission": "The permission level applied to the principal (READ, WRITE, or MANAGE).",
}

// ============================================================================
// Metadata Definitions
// ============================================================================

func secretMetadata() ResourceMetadata {
	return BuildMetadata(MetadataConfig{
		ResourceType:      "Databricks.DSC/Secret",
		Description:       "Manage Databricks secrets",
		SchemaDescription: "Schema for managing Databricks secrets.",
		ResourceName:      "secret",
		Tags:              []string{"databricks", "secret", "workspace"},
		Descriptions:      secretPropertyDescriptions,
		SchemaType:        reflect.TypeOf(workspace.PutSecret{}),
	})
}

func secretScopeMetadata() ResourceMetadata {
	return BuildMetadata(MetadataConfig{
		ResourceType:      "Databricks.DSC/SecretScope",
		Description:       "Manage Databricks secret scopes",
		SchemaDescription: "Schema for managing Databricks secret scopes.",
		ResourceName:      "secret scope",
		Tags:              []string{"databricks", "secret", "scope", "workspace"},
		Descriptions:      secretScopePropertyDescriptions,
		SchemaType:        reflect.TypeOf(workspace.CreateScope{}),
	})
}

func secretAclMetadata() ResourceMetadata {
	return BuildMetadata(MetadataConfig{
		ResourceType:      "Databricks.DSC/SecretAcl",
		Description:       "Manage Databricks secret ACLs",
		SchemaDescription: "Schema for managing Databricks secret ACLs.",
		ResourceName:      "secret ACL",
		Tags:              []string{"databricks", "secret", "acl", "permissions", "workspace"},
		Descriptions:      secretAclPropertyDescriptions,
		SchemaType:        reflect.TypeOf(workspace.PutAcl{}),
	})
}

type SecretState struct {
	Scope string `json:"scope"`
	Key   string `json:"key"`
}

type SecretHandler struct{}

func (h *SecretHandler) Get(ctx ResourceContext, input json.RawMessage) (any, error) {
	req, err := unmarshalInput[workspace.PutSecret](input)
	if err != nil {
		return nil, err
	}
	if err := validateRequired(
		RequiredField{"scope", req.Scope},
		RequiredField{"key", req.Key},
	); err != nil {
		return nil, err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	secrets := w.Secrets.ListSecrets(cmdCtx, workspace.ListSecretsRequest{Scope: req.Scope})
	for {
		secret, err := secrets.Next(cmdCtx)
		if err != nil {
			break
		}
		if secret.Key == req.Key {
			return SecretState{Scope: req.Scope, Key: req.Key}, nil
		}
	}
	return nil, NotFoundError("secret", "scope="+req.Scope, "key="+req.Key)
}

func (h *SecretHandler) Set(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.PutSecret](input)
	if err != nil {
		return err
	}
	if err := validateRequired(
		RequiredField{"scope", req.Scope},
		RequiredField{"key", req.Key},
	); err != nil {
		return err
	}
	if err := validateAtLeastOne("string_value or bytes_value", req.StringValue, req.BytesValue); err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	return w.Secrets.PutSecret(cmdCtx, req)
}

func (h *SecretHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.DeleteSecret](input)
	if err != nil {
		return err
	}
	if err := validateRequired(
		RequiredField{"scope", req.Scope},
		RequiredField{"key", req.Key},
	); err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	return w.Secrets.DeleteSecret(cmdCtx, req)
}

func (h *SecretHandler) Export(ctx ResourceContext) (any, error) {
	cmdCtx, w := getWorkspaceClient(ctx)

	var allSecrets []SecretState

	scopes := w.Secrets.ListScopes(cmdCtx)
	for {
		scope, err := scopes.Next(cmdCtx)
		if err != nil {
			break
		}

		secrets := w.Secrets.ListSecrets(cmdCtx, workspace.ListSecretsRequest{Scope: scope.Name})
		for {
			secret, err := secrets.Next(cmdCtx)
			if err != nil {
				break
			}
			allSecrets = append(allSecrets, SecretState{
				Scope: scope.Name,
				Key:   secret.Key,
			})
		}
	}

	return allSecrets, nil
}

type SecretScopeState struct {
	Scope       string `json:"scope"`
	BackendType string `json:"backend_type"`
}

type SecretScopeHandler struct{}

func (h *SecretScopeHandler) Get(ctx ResourceContext, input json.RawMessage) (any, error) {
	req, err := unmarshalInput[workspace.CreateScope](input)
	if err != nil {
		return nil, err
	}
	if err := validateRequired(RequiredField{"scope", req.Scope}); err != nil {
		return nil, err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	scopes := w.Secrets.ListScopes(cmdCtx)
	for {
		scope, err := scopes.Next(cmdCtx)
		if err != nil {
			break
		}
		if scope.Name == req.Scope {
			return SecretScopeState{
				Scope:       scope.Name,
				BackendType: scope.BackendType.String(),
			}, nil
		}
	}
	return nil, NotFoundError("scope", req.Scope)
}

func (h *SecretScopeHandler) Set(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.CreateScope](input)
	if err != nil {
		return err
	}
	if err := validateRequired(RequiredField{"scope", req.Scope}); err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	// Check if scope exists
	scopes := w.Secrets.ListScopes(cmdCtx)
	for {
		scope, err := scopes.Next(cmdCtx)
		if err != nil {
			break
		}
		if scope.Name == req.Scope {
			// Scope already exists, nothing to do
			return nil
		}
	}

	// Create new scope
	return w.Secrets.CreateScope(cmdCtx, req)
}

func (h *SecretScopeHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.DeleteScope](input)
	if err != nil {
		return err
	}
	if err := validateRequired(RequiredField{"scope", req.Scope}); err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	return w.Secrets.DeleteScope(cmdCtx, req)
}

func (h *SecretScopeHandler) Export(ctx ResourceContext) (any, error) {
	cmdCtx, w := getWorkspaceClient(ctx)

	var allScopes []SecretScopeState

	scopes := w.Secrets.ListScopes(cmdCtx)
	for {
		scope, err := scopes.Next(cmdCtx)
		if err != nil {
			break
		}
		allScopes = append(allScopes, SecretScopeState{
			Scope:       scope.Name,
			BackendType: scope.BackendType.String(),
		})
	}

	return allScopes, nil
}

type SecretAclState struct {
	Scope      string `json:"scope"`
	Principal  string `json:"principal"`
	Permission string `json:"permission"`
}

type SecretAclHandler struct{}

func (h *SecretAclHandler) Get(ctx ResourceContext, input json.RawMessage) (any, error) {
	req, err := unmarshalInput[workspace.GetAclRequest](input)
	if err != nil {
		return nil, err
	}
	if err := validateRequired(
		RequiredField{"scope", req.Scope},
		RequiredField{"principal", req.Principal},
	); err != nil {
		return nil, err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	acl, err := w.Secrets.GetAcl(cmdCtx, req)
	if err != nil {
		return nil, err
	}

	return SecretAclState{
		Scope:      req.Scope,
		Principal:  acl.Principal,
		Permission: acl.Permission.String(),
	}, nil
}

func (h *SecretAclHandler) Set(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.PutAcl](input)
	if err != nil {
		return err
	}
	if err := validateRequired(
		RequiredField{"scope", req.Scope},
		RequiredField{"principal", req.Principal},
		RequiredField{"permission", string(req.Permission)},
	); err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	return w.Secrets.PutAcl(cmdCtx, req)
}

func (h *SecretAclHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.DeleteAcl](input)
	if err != nil {
		return err
	}
	if err := validateRequired(
		RequiredField{"scope", req.Scope},
		RequiredField{"principal", req.Principal},
	); err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	return w.Secrets.DeleteAcl(cmdCtx, req)
}

func (h *SecretAclHandler) Export(ctx ResourceContext) (any, error) {
	cmdCtx, w := getWorkspaceClient(ctx)

	var allAcls []SecretAclState

	scopes := w.Secrets.ListScopes(cmdCtx)
	for {
		scope, err := scopes.Next(cmdCtx)
		if err != nil {
			break
		}

		acls := w.Secrets.ListAcls(cmdCtx, workspace.ListAclsRequest{Scope: scope.Name})
		for {
			acl, err := acls.Next(cmdCtx)
			if err != nil {
				break
			}
			allAcls = append(allAcls, SecretAclState{
				Scope:      scope.Name,
				Principal:  acl.Principal,
				Permission: acl.Permission.String(),
			})
		}
	}

	return allAcls, nil
}
