package resources

// PermissionsDeclarer is implemented by every resource type that has a permissions field.
type PermissionsDeclarer interface {
	// PermissionsSupport is called on a zero value and must not read the receiver.
	PermissionsSupport() PermissionsSupport
}

// PermissionsSupport describes how the permissions of a resource type are applied and deployed.
// Its parts are independent: secret scopes accept bundle-level permissions but have their own
// ACL API, and cluster policies use the permissions API but accept no bundle-level permissions.
type PermissionsSupport struct {
	// BundleLevels maps the bundle-level permissions to the levels of this type.
	BundleLevels BundlePermissionLevels

	// RequestObjectType is the object type in permissions API paths, for example "sql/warehouses".
	// It is empty if the type does not use the permissions API.
	// https://github.com/databricks/terraform-provider-databricks/blob/430902d/permissions/permission_definitions.go#L775C24-L775C32
	RequestObjectType string

	// ResponseObjectType is the object_type in permissions API responses. It can differ from
	// RequestObjectType, for example "warehouses" for "sql/warehouses".
	ResponseObjectType string

	// ObjectIDField is the field of the deployed resource that holds the ID in permissions API paths.
	ObjectIDField string

	// HasIsOwner is set if FixPermissions makes the current user IS_OWNER rather than CAN_MANAGE.
	HasIsOwner bool

	// SkipFixup is set if FixPermissions must leave the permissions of this type unchanged.
	SkipFixup bool

	// CreatorLevel is a level the backend grants the creator of a resource that the
	// configuration never needs to declare. config-remote-sync does not import it.
	CreatorLevel string
}

// BundlePermissionLevels maps the bundle-level permissions CAN_MANAGE, CAN_VIEW and CAN_RUN
// to the levels of a resource type. An empty level is not applied to resources of the type.
type BundlePermissionLevels struct {
	CanManage string
	CanView   string
	CanRun    string
}
