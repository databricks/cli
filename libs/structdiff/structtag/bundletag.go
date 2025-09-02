package structtag

// BundleTag represents a struct field's `bundle` tag as a string.
// It provides methods to extract information from the tag.
type BundleTag string

func (tag BundleTag) ReadOnly() bool {
	return hasOption(string(tag), "readonly")
}

func (tag BundleTag) Internal() bool {
	return hasOption(string(tag), "internal")
}
