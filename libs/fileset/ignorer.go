package fileset

// Ignorer is the interface for what determines if a path
// in the [FileSet] must be ignored or not.
type Ignorer interface {
	IgnoreFile(path string) bool
	IgnoreDirectory(path string) bool
}

// nopIgnorer implements an [Ignorer] that doesn't ignore anything.
type nopIgnorer struct{}

func (nopIgnorer) IgnoreFile(path string) bool {
	return false
}

func (nopIgnorer) IgnoreDirectory(path string) bool {
	return false
}
