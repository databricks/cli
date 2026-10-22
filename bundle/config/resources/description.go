package resources

type ResourceDescription struct {
	// Singular and plural name when used to refer to the configuration.
	SingularName string
	PluralName   string

	// Singular and plural title when used in summaries / terminal UI.
	SingularTitle string
	PluralTitle   string

	// Permissions is zero for resources without a permissions sub-resource.
	Permissions PermissionsDescription
}

// PermissionsDescription keeps bundle-level mappings, API routing, and principal
// fixups separate: secret scopes use their own ACL API, and cluster policies
// support IAM permissions without accepting bundle-level permissions.
type PermissionsDescription struct {
	BundleLevels BundlePermissionLevels

	// RequestObjectType is the IAM Permissions API path segment (e.g. "sql/warehouses").
	// An empty value means the resource uses a different ACL API.
	// https://github.com/databricks/terraform-provider-databricks/blob/430902d/permissions/permission_definitions.go#L775C24-L775C32
	RequestObjectType string
	// ResponseObjectType is the object_type returned by that API; it can differ from the path.
	ResponseObjectType string
	// ObjectIDField is the parent field used in IAM object IDs, including "id" for the default.
	ObjectIDField string

	// HasIsOwner indicates whether FixPermissions can add the current user as IS_OWNER.
	HasIsOwner bool
	// SkipFixup leaves the resource's permission levels and principals unchanged.
	SkipFixup bool
}

// BundlePermissionLevels maps the three bundle-level permissions to API levels.
// An empty value means that level is not supported by the resource.
type BundlePermissionLevels struct {
	CanManage string
	CanView   string
	CanRun    string
}
