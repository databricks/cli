package deployment

import (
	"context"
	"fmt"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// buildDirectClient is the indirection tests patch to inject a fake direct
// client. The production implementation defers to direct.NewClient against
// the memoized workspace client on u.
var buildDirectClient = func(_ context.Context, u *ucm.Ucm) (direct.Client, error) {
	w, err := u.WorkspaceClientE()
	if err != nil {
		return nil, fmt.Errorf("resolve workspace client: %w", err)
	}
	return direct.NewClient(w), nil
}

// bindableKind mirrors the plural resource names used in ucm.yml so error
// messages stay user-facing without translation.
type bindableKind string

const (
	kindCatalog           bindableKind = "catalogs"
	kindSchema            bindableKind = "schemas"
	kindStorageCredential bindableKind = "storage_credentials"
	kindExternalLocation  bindableKind = "external_locations"
	kindVolume            bindableKind = "volumes"
	kindConnection        bindableKind = "connections"
)

// resolveBindable returns the kind the given key maps to in the ucm config.
// Returns an error when the key matches zero or multiple kinds — bind is
// unambiguous by design.
func resolveBindable(u *ucm.Ucm, key string) (bindableKind, error) {
	var matches []bindableKind
	if _, ok := u.Config.Resources.Catalogs[key]; ok {
		matches = append(matches, kindCatalog)
	}
	if _, ok := u.Config.Resources.Schemas[key]; ok {
		matches = append(matches, kindSchema)
	}
	if _, ok := u.Config.Resources.StorageCredentials[key]; ok {
		matches = append(matches, kindStorageCredential)
	}
	if _, ok := u.Config.Resources.ExternalLocations[key]; ok {
		matches = append(matches, kindExternalLocation)
	}
	if _, ok := u.Config.Resources.Volumes[key]; ok {
		matches = append(matches, kindVolume)
	}
	if _, ok := u.Config.Resources.Connections[key]; ok {
		matches = append(matches, kindConnection)
	}
	if _, ok := u.Config.Resources.Grants[key]; ok {
		return "", fmt.Errorf("grants are not bindable (they reconcile per securable, not by name)")
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no bindable resource with key %q in ucm.yml", key)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous key %q: matches %v", key, matches)
	}
}

// bindResourceDirect fetches the live UC object by name and records the
// equivalent *State entry into direct-engine state. The ucm.yml config
// itself is not modified — bind only affects recorded state.
func bindResourceDirect(ctx context.Context, u *ucm.Ucm, client direct.Client, kind bindableKind, key, ucName string) error {
	state, err := direct.LoadState(direct.StatePath(u))
	if err != nil {
		return fmt.Errorf("load direct state: %w", err)
	}

	switch kind {
	case kindCatalog:
		info, err := client.GetCatalog(ctx, ucName)
		if err != nil {
			return fmt.Errorf("fetch catalog %q: %w", ucName, err)
		}
		state.Catalogs[key] = &direct.CatalogState{
			Name:        info.Name,
			Comment:     info.Comment,
			StorageRoot: info.StorageRoot,
			Tags:        copyStringMap(info.Properties),
		}
	case kindSchema:
		info, err := client.GetSchema(ctx, ucName)
		if err != nil {
			return fmt.Errorf("fetch schema %q: %w", ucName, err)
		}
		state.Schemas[key] = &direct.SchemaState{
			Name:    info.Name,
			Catalog: info.CatalogName,
			Comment: info.Comment,
			Tags:    copyStringMap(info.Properties),
		}
	case kindStorageCredential:
		info, err := client.GetStorageCredential(ctx, ucName)
		if err != nil {
			return fmt.Errorf("fetch storage_credential %q: %w", ucName, err)
		}
		state.StorageCredentials[key] = storageCredentialStateFromInfo(info)
	case kindExternalLocation:
		info, err := client.GetExternalLocation(ctx, ucName)
		if err != nil {
			return fmt.Errorf("fetch external_location %q: %w", ucName, err)
		}
		state.ExternalLocations[key] = &direct.ExternalLocationState{
			Name:           info.Name,
			Url:            info.Url,
			CredentialName: info.CredentialName,
			Comment:        info.Comment,
			ReadOnly:       info.ReadOnly,
			Fallback:       info.Fallback,
		}
	case kindVolume:
		info, err := client.GetVolume(ctx, ucName)
		if err != nil {
			return fmt.Errorf("fetch volume %q: %w", ucName, err)
		}
		state.Volumes[key] = &direct.VolumeState{
			Name:            info.Name,
			CatalogName:     info.CatalogName,
			SchemaName:      info.SchemaName,
			VolumeType:      string(info.VolumeType),
			StorageLocation: info.StorageLocation,
			Comment:         info.Comment,
		}
	case kindConnection:
		info, err := client.GetConnection(ctx, ucName)
		if err != nil {
			return fmt.Errorf("fetch connection %q: %w", ucName, err)
		}
		state.Connections[key] = &direct.ConnectionState{
			Name:           info.Name,
			ConnectionType: string(info.ConnectionType),
			Options:        copyStringMap(info.Options),
			Comment:        info.Comment,
			Properties:     copyStringMap(info.Properties),
			ReadOnly:       info.ReadOnly,
		}
	default:
		return fmt.Errorf("unsupported kind %q", kind)
	}

	if err := direct.SaveState(direct.StatePath(u), state); err != nil {
		return fmt.Errorf("save direct state: %w", err)
	}
	return nil
}

// unbindResourceDirect drops the recorded state entry for the given key.
// Returns a descriptive error if the key isn't currently bound.
func unbindResourceDirect(u *ucm.Ucm, kind bindableKind, key string) error {
	path := direct.StatePath(u)
	state, err := direct.LoadState(path)
	if err != nil {
		return fmt.Errorf("load direct state: %w", err)
	}

	present := false
	switch kind {
	case kindCatalog:
		_, present = state.Catalogs[key]
		delete(state.Catalogs, key)
	case kindSchema:
		_, present = state.Schemas[key]
		delete(state.Schemas, key)
	case kindStorageCredential:
		_, present = state.StorageCredentials[key]
		delete(state.StorageCredentials, key)
	case kindExternalLocation:
		_, present = state.ExternalLocations[key]
		delete(state.ExternalLocations, key)
	case kindVolume:
		_, present = state.Volumes[key]
		delete(state.Volumes, key)
	case kindConnection:
		_, present = state.Connections[key]
		delete(state.Connections, key)
	default:
		return fmt.Errorf("unsupported kind %q", kind)
	}
	if !present {
		return fmt.Errorf("no bound %s with key %q in direct state", kind, key)
	}
	return direct.SaveState(path, state)
}

// storageCredentialStateFromInfo projects the SDK StorageCredentialInfo into
// the recorded state shape. ClientSecret for Azure SP is never echoed by the
// UC API — a subsequent deploy re-records it once supplied via config.
func storageCredentialStateFromInfo(info *catalog.StorageCredentialInfo) *direct.StorageCredentialState {
	s := &direct.StorageCredentialState{
		Name:     info.Name,
		Comment:  info.Comment,
		ReadOnly: info.ReadOnly,
	}
	if info.AwsIamRole != nil {
		s.AwsIamRole = &direct.AwsIamRoleState{RoleArn: info.AwsIamRole.RoleArn}
	}
	if info.AzureManagedIdentity != nil {
		s.AzureManagedIdentity = &direct.AzureManagedIdentityState{
			AccessConnectorId: info.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: info.AzureManagedIdentity.ManagedIdentityId,
		}
	}
	if info.AzureServicePrincipal != nil {
		s.AzureServicePrincipal = &direct.AzureServicePrincipalState{
			DirectoryId:   info.AzureServicePrincipal.DirectoryId,
			ApplicationId: info.AzureServicePrincipal.ApplicationId,
			ClientSecret:  info.AzureServicePrincipal.ClientSecret,
		}
	}
	if info.DatabricksGcpServiceAccount != nil {
		s.DatabricksGcpServiceAccount = &direct.DatabricksGcpServiceAccountState{}
	}
	return s
}

// copyStringMap returns a copy of m or nil when m is empty.
func copyStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// notSupportedForEngine is the standard error for the terraform engine.
// Terraform-engine bind requires wiring `terraform import` through the ucm
// Terraform wrapper (new tfexec Import method + state pull/push integration)
// which is out of scope for this commit — tracked separately.
func notSupportedForEngine(e engine.EngineType) error {
	return fmt.Errorf("ucm deployment bind/unbind is not yet implemented for the %q engine; set engine: direct in ucm.yml or DATABRICKS_UCM_ENGINE=direct to proceed", e.ThisOrDefault())
}
