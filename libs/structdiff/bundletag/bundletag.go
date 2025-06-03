package bundletag

import (
	"slices"
	"strings"
)

// BundleTag represents a struct field's `json` tag as a string.
// It provides methods to extract information from the tag.
type BundleTag string

func (tag BundleTag) hasAnnotation(option string) bool {
	s := string(tag)
	if s == "" {
		return false
	}

	parts := strings.Split(s, ",")
	return slices.Contains(parts, option)
}

func (tag BundleTag) ReadOnly() bool {
	return tag.hasAnnotation("readonly")
}

func (tag BundleTag) Internal() bool {
	return tag.hasAnnotation("internal")
}

func (tag BundleTag) Deprecated() bool {
	return tag.hasAnnotation("deprecated")
}
