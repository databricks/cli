package resources

import (
	"github.com/databricks/cli/libs/sdktypes/uc/catalogs/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/externallocations/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/schemas/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/volumes/v1"
)

// The Unity Catalog create requests in sdk-go reuse the full securable
// message, including output-only fields such as created_at and metastore_id.
// Bundle configuration only accepts the fields a user can set on create, so
// these types carry that subset and convert to the full request.

type CreateCatalog struct {
	// User-provided free-form text description.
	Comment *string `json:"comment,omitempty"`
	// The name of the connection to an external data source.
	ConnectionName *string `json:"connection_name,omitempty"`
	// Custom maximum retention period in hours for the catalog
	CustomMaxRetentionHours *int64 `json:"custom_max_retention_hours,omitempty"`
	// Control CMK encryption for managed catalog data
	ManagedEncryptionSettings *catalogs.EncryptionSettings `json:"managed_encryption_settings,omitempty"`
	// Name of catalog.
	Name *string `json:"name,omitempty"`
	// A map of key-value properties attached to the securable.
	Options map[string]string `json:"options,omitempty"`
	// A map of key-value properties attached to the securable.
	Properties map[string]string `json:"properties,omitempty"`
	// The name of delta sharing provider.
	//
	// A Delta Sharing catalog is a catalog that is based on a Delta share on a
	// remote sharing server.
	ProviderName *string `json:"provider_name,omitempty"`
	// The name of the share under the share provider.
	ShareName *string `json:"share_name,omitempty"`
	// Storage root URL for managed tables within catalog.
	StorageRoot *string `json:"storage_root,omitempty"`
}

// AsRequest returns the create request for this configuration.
func (c *CreateCatalog) AsRequest() *catalogs.CreateCatalogRequest {
	return &catalogs.CreateCatalogRequest{
		Comment:                   c.Comment,
		ConnectionName:            c.ConnectionName,
		CustomMaxRetentionHours:   c.CustomMaxRetentionHours,
		ManagedEncryptionSettings: c.ManagedEncryptionSettings,
		Name:                      c.Name,
		Options:                   c.Options,
		Properties:                c.Properties,
		ProviderName:              c.ProviderName,
		ShareName:                 c.ShareName,
		StorageRoot:               c.StorageRoot,
	}
}

type CreateSchema struct {
	// Name of parent catalog.
	CatalogName *string `json:"catalog_name,omitempty"`
	// User-provided free-form text description.
	Comment *string `json:"comment,omitempty"`
	// Custom maximum retention period in hours for the schema.
	CustomMaxRetentionHours *int64 `json:"custom_max_retention_hours,omitempty"`
	// Name of schema, relative to parent catalog.
	Name *string `json:"name,omitempty"`
	// A map of key-value properties attached to the securable.
	Properties map[string]string `json:"properties,omitempty"`
	// Storage root URL for managed tables within schema.
	StorageRoot *string `json:"storage_root,omitempty"`
}

// AsRequest returns the create request for this configuration.
func (c *CreateSchema) AsRequest() *schemas.CreateSchemaRequest {
	return &schemas.CreateSchemaRequest{
		CatalogName:             c.CatalogName,
		Comment:                 c.Comment,
		CustomMaxRetentionHours: c.CustomMaxRetentionHours,
		Name:                    c.Name,
		Properties:              c.Properties,
		StorageRoot:             c.StorageRoot,
	}
}

type CreateVolumeRequestContent struct {
	// The name of the catalog where the schema and the volume are
	CatalogName *string `json:"catalog_name,omitempty"`
	// The comment attached to the volume
	Comment *string `json:"comment,omitempty"`
	// The name of the volume
	Name *string `json:"name,omitempty"`
	// The name of the schema where the volume is
	SchemaName *string `json:"schema_name,omitempty"`
	// The storage location on the cloud
	StorageLocation *string `json:"storage_location,omitempty"`
	// The type of the volume. An external volume is located in the specified
	// external location. A managed volume is located in the default location which
	// is specified by the parent schema, or the parent catalog, or the Metastore.
	// [Learn more]
	//
	// [Learn more]: https://docs.databricks.com/aws/en/volumes/managed-vs-external
	VolumeType volumes.VolumeType `json:"volume_type,omitempty"`
}

// AsRequest returns the create request for this configuration.
func (c *CreateVolumeRequestContent) AsRequest() *volumes.CreateVolumeRequest {
	return &volumes.CreateVolumeRequest{
		CatalogName:     c.CatalogName,
		Comment:         c.Comment,
		Name:            c.Name,
		SchemaName:      c.SchemaName,
		StorageLocation: c.StorageLocation,
		VolumeType:      c.VolumeType,
	}
}

type CreateExternalLocation struct {
	// User-provided free-form text description.
	Comment *string `json:"comment,omitempty"`
	// Name of the storage credential used with this location.
	CredentialName *string `json:"credential_name,omitempty"`
	// The effective value of `enable_file_events` after applying server-side
	// defaults.
	EffectiveEnableFileEvents *bool `json:"effective_enable_file_events,omitempty"`
	// The effective file event queue configuration after applying server-side
	// defaults. Always populated when a queue is provisioned, regardless of whether
	// the user explicitly set `enable_file_events`. Use this field instead of
	// `file_event_queue` for reading the actual queue state.
	EffectiveFileEventQueue *externallocations.FileEventQueue `json:"effective_file_event_queue,omitempty"`
	// Whether to enable file events on this external location. Default to `true`.
	// Set to `false` to disable file events. The actual applied value may differ
	// due to server-side defaults; check `effective_enable_file_events` for the
	// effective state.
	EnableFileEvents  *bool                                `json:"enable_file_events,omitempty"`
	EncryptionDetails *externallocations.EncryptionDetails `json:"encryption_details,omitempty"`
	// Indicates whether fallback mode is enabled for this external location. When
	// fallback mode is enabled, the access to the location falls back to cluster
	// credentials if UC credentials are not sufficient.
	Fallback *bool `json:"fallback,omitempty"`
	// File event queue settings. If `enable_file_events` is not `false`, must be
	// defined and have exactly one of the documented properties.
	FileEventQueue *externallocations.FileEventQueue `json:"file_event_queue,omitempty"`
	// Name of the external location.
	Name *string `json:"name,omitempty"`
	// Indicates whether the external location is read-only.
	ReadOnly *bool `json:"read_only,omitempty"`
	// Skips validation of the storage credential associated with the external
	// location.
	SkipValidation *bool `json:"skip_validation,omitempty"`
	// Path URL of the external location.
	Url *string `json:"url,omitempty"`
}

// AsRequest returns the create request for this configuration.
func (c *CreateExternalLocation) AsRequest() *externallocations.CreateExternalLocationRequest {
	return &externallocations.CreateExternalLocationRequest{
		Comment:                   c.Comment,
		CredentialName:            c.CredentialName,
		EffectiveEnableFileEvents: c.EffectiveEnableFileEvents,
		EffectiveFileEventQueue:   c.EffectiveFileEventQueue,
		EnableFileEvents:          c.EnableFileEvents,
		EncryptionDetails:         c.EncryptionDetails,
		Fallback:                  c.Fallback,
		FileEventQueue:            c.FileEventQueue,
		Name:                      c.Name,
		ReadOnly:                  c.ReadOnly,
		SkipValidation:            c.SkipValidation,
		Url:                       c.Url,
	}
}
