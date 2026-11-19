package dresources

import "github.com/databricks/cli/libs/structs/structcopy"

// fieldCopier is the part of a structcopy.Copier that does not depend on the copied types.
type fieldCopier interface {
	Describe() (string, error)
}

// fieldCopiers holds every Copier of this package, for TestFieldCopies.
var fieldCopiers []fieldCopier

// newCopier returns a Copier for the fields that Src and Dst share; see structcopy.New.
// Declare it in a package-level variable, so that TestFieldCopies covers it.
func newCopier[Src, Dst any](opts ...structcopy.Option) *structcopy.Copier[Src, Dst] {
	copier := structcopy.New[Src, Dst](opts...)
	fieldCopiers = append(fieldCopiers, copier)
	return copier
}
