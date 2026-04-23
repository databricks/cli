package direct

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// ImportResource fetches the named UC object via client.Get<Kind> and seeds
// the corresponding entry in state keyed by ucmKey. The config declaration
// under resources.<kind>.<ucmKey> supplies the fields the SDK does not echo
// (e.g. storage credential ClientSecret for Azure SP, skip_validation).
//
// kind is the CLI-facing singular name: "catalog", "schema",
// "storage_credential", "external_location", "volume", "connection".
func ImportResource(ctx context.Context, u *ucm.Ucm, client Client, state *State, kind, name, ucmKey string) error {
	switch kind {
	case "catalog":
		return importCatalog(ctx, u, client, state, name, ucmKey)
	case "schema":
		return importSchema(ctx, u, client, state, name, ucmKey)
	case "storage_credential":
		return importStorageCredential(ctx, u, client, state, name, ucmKey)
	case "external_location":
		return importExternalLocation(ctx, u, client, state, name, ucmKey)
	case "volume":
		return importVolume(ctx, u, client, state, name, ucmKey)
	case "connection":
		return importConnection(ctx, u, client, state, name, ucmKey)
	}
	return fmt.Errorf("unsupported import kind %q", kind)
}

func importCatalog(ctx context.Context, u *ucm.Ucm, client Client, state *State, name, key string) error {
	info, err := client.GetCatalog(ctx, name)
	if err != nil {
		return fmt.Errorf("get catalog %s: %w", name, err)
	}
	cfg := u.Config.Resources.Catalogs[key]
	rec := catalogStateFromConfig(cfg)
	mergeCatalogFromSDK(&rec, info)
	state.Catalogs[key] = ptrCatalog(rec)
	log.Infof(ctx, "direct: imported catalog %s as resources.catalogs.%s", name, key)
	return nil
}

func importSchema(ctx context.Context, u *ucm.Ucm, client Client, state *State, fullName, key string) error {
	info, err := client.GetSchema(ctx, fullName)
	if err != nil {
		return fmt.Errorf("get schema %s: %w", fullName, err)
	}
	cfg := u.Config.Resources.Schemas[key]
	rec := schemaStateFromConfig(cfg)
	mergeSchemaFromSDK(&rec, info)
	state.Schemas[key] = ptrSchema(rec)
	log.Infof(ctx, "direct: imported schema %s as resources.schemas.%s", fullName, key)
	return nil
}

func importStorageCredential(ctx context.Context, u *ucm.Ucm, client Client, state *State, name, key string) error {
	info, err := client.GetStorageCredential(ctx, name)
	if err != nil {
		return fmt.Errorf("get storage_credential %s: %w", name, err)
	}
	cfg := u.Config.Resources.StorageCredentials[key]
	rec := storageCredentialStateFromConfig(cfg)
	mergeStorageCredentialFromSDK(&rec, info)
	state.StorageCredentials[key] = ptrStorageCredential(rec)
	log.Infof(ctx, "direct: imported storage_credential %s as resources.storage_credentials.%s", name, key)
	return nil
}

func importExternalLocation(ctx context.Context, u *ucm.Ucm, client Client, state *State, name, key string) error {
	info, err := client.GetExternalLocation(ctx, name)
	if err != nil {
		return fmt.Errorf("get external_location %s: %w", name, err)
	}
	cfg := u.Config.Resources.ExternalLocations[key]
	rec := externalLocationStateFromConfig(cfg)
	mergeExternalLocationFromSDK(&rec, info)
	state.ExternalLocations[key] = ptrExternalLocation(rec)
	log.Infof(ctx, "direct: imported external_location %s as resources.external_locations.%s", name, key)
	return nil
}

func importVolume(ctx context.Context, u *ucm.Ucm, client Client, state *State, fullName, key string) error {
	info, err := client.GetVolume(ctx, fullName)
	if err != nil {
		return fmt.Errorf("get volume %s: %w", fullName, err)
	}
	cfg := u.Config.Resources.Volumes[key]
	rec := volumeStateFromConfig(cfg)
	mergeVolumeFromSDK(&rec, info)
	state.Volumes[key] = ptrVolume(rec)
	log.Infof(ctx, "direct: imported volume %s as resources.volumes.%s", fullName, key)
	return nil
}

func importConnection(ctx context.Context, u *ucm.Ucm, client Client, state *State, name, key string) error {
	info, err := client.GetConnection(ctx, name)
	if err != nil {
		return fmt.Errorf("get connection %s: %w", name, err)
	}
	cfg := u.Config.Resources.Connections[key]
	rec := connectionStateFromConfig(cfg)
	mergeConnectionFromSDK(&rec, info)
	state.Connections[key] = ptrConnection(rec)
	log.Infof(ctx, "direct: imported connection %s as resources.connections.%s", name, key)
	return nil
}

// ---- SDK → state merge helpers ----
//
// Each helper overlays fields that the SDK echoes back on top of the
// config-derived state. Server-authoritative strings (Name, URL, etc.) take
// the SDK value; user-only fields (e.g. Azure SP ClientSecret) stay as the
// config supplied them since the UC API never echoes secrets.

func mergeCatalogFromSDK(s *CatalogState, info *catalog.CatalogInfo) {
	if info == nil {
		return
	}
	if info.Name != "" {
		s.Name = info.Name
	}
	if info.Comment != "" {
		s.Comment = info.Comment
	}
	if info.StorageRoot != "" {
		s.StorageRoot = info.StorageRoot
	}
	if len(info.Properties) > 0 {
		s.Tags = copyTags(info.Properties)
	}
}

func mergeSchemaFromSDK(s *SchemaState, info *catalog.SchemaInfo) {
	if info == nil {
		return
	}
	if info.Name != "" {
		s.Name = info.Name
	}
	if info.CatalogName != "" {
		s.Catalog = info.CatalogName
	}
	if info.Comment != "" {
		s.Comment = info.Comment
	}
	if len(info.Properties) > 0 {
		s.Tags = copyTags(info.Properties)
	}
}

func mergeStorageCredentialFromSDK(s *StorageCredentialState, info *catalog.StorageCredentialInfo) {
	if info == nil {
		return
	}
	if info.Name != "" {
		s.Name = info.Name
	}
	if info.Comment != "" {
		s.Comment = info.Comment
	}
	s.ReadOnly = info.ReadOnly
	if info.AwsIamRole != nil {
		s.AwsIamRole = &AwsIamRoleState{RoleArn: info.AwsIamRole.RoleArn}
	}
	if info.AzureManagedIdentity != nil {
		s.AzureManagedIdentity = &AzureManagedIdentityState{
			AccessConnectorId: info.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: info.AzureManagedIdentity.ManagedIdentityId,
		}
	}
	// ClientSecret intentionally retained from config — the UC API never
	// returns Azure SP client secrets.
	if info.AzureServicePrincipal != nil {
		secret := ""
		if s.AzureServicePrincipal != nil {
			secret = s.AzureServicePrincipal.ClientSecret
		}
		s.AzureServicePrincipal = &AzureServicePrincipalState{
			DirectoryId:   info.AzureServicePrincipal.DirectoryId,
			ApplicationId: info.AzureServicePrincipal.ApplicationId,
			ClientSecret:  secret,
		}
	}
	if info.DatabricksGcpServiceAccount != nil {
		s.DatabricksGcpServiceAccount = &DatabricksGcpServiceAccountState{}
	}
}

func mergeExternalLocationFromSDK(s *ExternalLocationState, info *catalog.ExternalLocationInfo) {
	if info == nil {
		return
	}
	if info.Name != "" {
		s.Name = info.Name
	}
	if info.Url != "" {
		s.Url = info.Url
	}
	if info.CredentialName != "" {
		s.CredentialName = info.CredentialName
	}
	if info.Comment != "" {
		s.Comment = info.Comment
	}
	s.ReadOnly = info.ReadOnly
	s.Fallback = info.Fallback
}

func mergeVolumeFromSDK(s *VolumeState, info *catalog.VolumeInfo) {
	if info == nil {
		return
	}
	if info.Name != "" {
		s.Name = info.Name
	}
	if info.CatalogName != "" {
		s.CatalogName = info.CatalogName
	}
	if info.SchemaName != "" {
		s.SchemaName = info.SchemaName
	}
	if info.VolumeType != "" {
		s.VolumeType = string(info.VolumeType)
	}
	if info.StorageLocation != "" {
		s.StorageLocation = info.StorageLocation
	}
	if info.Comment != "" {
		s.Comment = info.Comment
	}
}

func mergeConnectionFromSDK(s *ConnectionState, info *catalog.ConnectionInfo) {
	if info == nil {
		return
	}
	if info.Name != "" {
		s.Name = info.Name
	}
	if info.ConnectionType != "" {
		s.ConnectionType = string(info.ConnectionType)
	}
	if len(info.Options) > 0 {
		s.Options = copyTags(info.Options)
	}
	if info.Comment != "" {
		s.Comment = info.Comment
	}
	if len(info.Properties) > 0 {
		s.Properties = copyTags(info.Properties)
	}
	s.ReadOnly = info.ReadOnly
}
