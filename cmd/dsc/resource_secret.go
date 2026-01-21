package dsc

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	RegisterResourceWithMetadata("Databricks/Secret", &SecretHandler{}, secretMetadata())
	RegisterResourceWithMetadata("Databricks/SecretScope", &SecretScopeHandler{}, secretScopeMetadata())
	RegisterResourceWithMetadata("Databricks/SecretAcl", &SecretAclHandler{}, secretAclMetadata())
}

// ============================================================================
// Metadata Definitions - Using SDK types for schema generation
// ============================================================================

// Descriptions from SDK documentation
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

func secretMetadata() ResourceMetadata {
	schema, _ := GenerateSchemaWithOptions(reflect.TypeOf(workspace.PutSecret{}), SchemaOptions{
		Descriptions:      secretPropertyDescriptions,
		SchemaDescription: "Schema for managing Databricks secrets.",
		ResourceName:      "secret",
	})
	return ResourceMetadata{
		Type:        "Databricks.DSC/Secret",
		Version:     "0.1.0",
		Description: "Manage Databricks secrets",
		Tags:        []string{"databricks", "secret", "workspace"},
		ExitCodes: map[string]string{
			"0": "Success",
			"1": "Error",
		},
		Schema: ResourceSchema{
			Embedded: schema,
		},
	}
}

func secretScopeMetadata() ResourceMetadata {
	schema, _ := GenerateSchemaWithOptions(reflect.TypeOf(workspace.CreateScope{}), SchemaOptions{
		Descriptions:      secretScopePropertyDescriptions,
		SchemaDescription: "Schema for managing Databricks secret scopes.",
		ResourceName:      "secret scope",
	})
	return ResourceMetadata{
		Type:        "Databricks.DSC/SecretScope",
		Version:     "0.1.0",
		Description: "Manage Databricks secret scopes",
		Tags:        []string{"databricks", "secret", "scope", "workspace"},
		ExitCodes: map[string]string{
			"0": "Success",
			"1": "Error",
		},
		Schema: ResourceSchema{
			Embedded: schema,
		},
	}
}

func secretAclMetadata() ResourceMetadata {
	schema, _ := GenerateSchemaWithOptions(reflect.TypeOf(workspace.PutAcl{}), SchemaOptions{
		Descriptions:      secretAclPropertyDescriptions,
		SchemaDescription: "Schema for managing Databricks secret ACLs.",
		ResourceName:      "secret ACL",
	})
	return ResourceMetadata{
		Type:        "Databricks.DSC/SecretAcl",
		Version:     "0.1.0",
		Description: "Manage Databricks secret ACLs",
		Tags:        []string{"databricks", "secret", "acl", "permissions", "workspace"},
		ExitCodes: map[string]string{
			"0": "Success",
			"1": "Error",
		},
		Schema: ResourceSchema{
			Embedded: schema,
		},
	}
}

// ============================================================================
// Helper function to unmarshal input to SDK types
// ============================================================================

func unmarshalInput[T any](input json.RawMessage) (T, error) {
	var req T
	if err := json.Unmarshal(input, &req); err != nil {
		return req, fmt.Errorf("failed to parse input: %w", err)
	}
	return req, nil
}

// ============================================================================
// Secret Resource - Uses workspace.PutSecret from SDK
// ============================================================================

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
	if req.Scope == "" {
		return nil, fmt.Errorf("scope is required")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("key is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

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
	return nil, fmt.Errorf("secret not found: scope=%s, key=%s", req.Scope, req.Key)
}

func (h *SecretHandler) Set(ctx ResourceContext, input json.RawMessage) (any, error) {
	req, err := unmarshalInput[workspace.PutSecret](input)
	if err != nil {
		return nil, err
	}
	if req.Scope == "" {
		return nil, fmt.Errorf("scope is required")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if req.StringValue == "" && req.BytesValue == "" {
		return nil, fmt.Errorf("either string_value or bytes_value is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

	if err := w.Secrets.PutSecret(cmdCtx, req); err != nil {
		return nil, err
	}

	return DSCResult{
		ActualState:    SecretState{Scope: req.Scope, Key: req.Key},
		InDesiredState: true,
	}, nil
}

func (h *SecretHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.DeleteSecret](input)
	if err != nil {
		return err
	}
	if req.Scope == "" {
		return fmt.Errorf("scope is required")
	}
	if req.Key == "" {
		return fmt.Errorf("key is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

	return w.Secrets.DeleteSecret(cmdCtx, req)
}

func (h *SecretHandler) Export(ctx ResourceContext) (any, error) {
	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

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
	if req.Scope == "" {
		return nil, fmt.Errorf("scope is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

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
	return nil, fmt.Errorf("scope not found: %s", req.Scope)
}

func (h *SecretScopeHandler) Set(ctx ResourceContext, input json.RawMessage) (any, error) {
	req, err := unmarshalInput[workspace.CreateScope](input)
	if err != nil {
		return nil, err
	}
	if req.Scope == "" {
		return nil, fmt.Errorf("scope is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

	// Check if scope exists
	scopes := w.Secrets.ListScopes(cmdCtx)
	for {
		scope, err := scopes.Next(cmdCtx)
		if err != nil {
			break
		}
		if scope.Name == req.Scope {
			// Scope already exists
			return DSCResult{
				ActualState: SecretScopeState{
					Scope:       scope.Name,
					BackendType: scope.BackendType.String(),
				},
				InDesiredState: true,
			}, nil
		}
	}

	// Create new scope
	if err := w.Secrets.CreateScope(cmdCtx, req); err != nil {
		return nil, err
	}

	backendType := "DATABRICKS"
	if req.ScopeBackendType != "" {
		backendType = req.ScopeBackendType.String()
	}

	return DSCResult{
		ActualState: SecretScopeState{
			Scope:       req.Scope,
			BackendType: backendType,
		},
		InDesiredState: true,
	}, nil
}

func (h *SecretScopeHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.DeleteScope](input)
	if err != nil {
		return err
	}
	if req.Scope == "" {
		return fmt.Errorf("scope is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

	return w.Secrets.DeleteScope(cmdCtx, req)
}

func (h *SecretScopeHandler) Export(ctx ResourceContext) (any, error) {
	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

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
	if req.Scope == "" {
		return nil, fmt.Errorf("scope is required")
	}
	if req.Principal == "" {
		return nil, fmt.Errorf("principal is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

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

func (h *SecretAclHandler) Set(ctx ResourceContext, input json.RawMessage) (any, error) {
	req, err := unmarshalInput[workspace.PutAcl](input)
	if err != nil {
		return nil, err
	}
	if req.Scope == "" {
		return nil, fmt.Errorf("scope is required")
	}
	if req.Principal == "" {
		return nil, fmt.Errorf("principal is required")
	}
	if req.Permission == "" {
		return nil, fmt.Errorf("permission is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

	if err := w.Secrets.PutAcl(cmdCtx, req); err != nil {
		return nil, err
	}

	return DSCResult{
		ActualState: SecretAclState{
			Scope:      req.Scope,
			Principal:  req.Principal,
			Permission: req.Permission.String(),
		},
		InDesiredState: true,
	}, nil
}

func (h *SecretAclHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[workspace.DeleteAcl](input)
	if err != nil {
		return err
	}
	if req.Scope == "" {
		return fmt.Errorf("scope is required")
	}
	if req.Principal == "" {
		return fmt.Errorf("principal is required")
	}

	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

	return w.Secrets.DeleteAcl(cmdCtx, req)
}

func (h *SecretAclHandler) Export(ctx ResourceContext) (any, error) {
	cmdCtx := ctx.Cmd.Context()
	w := cmdctx.WorkspaceClient(cmdCtx)

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
