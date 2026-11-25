package resources

// ResourcePathUser is implemented by resource types that need ${workspace.resource_path}
// to exist. Deploy creates that directory when the bundle declares a resource of such a
// type.
type ResourcePathUser interface {
	// UsesResourcePath declares that the type needs ${workspace.resource_path}.
	UsesResourcePath()
}
