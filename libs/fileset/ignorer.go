package fileset

// Ignorer is the interface for what determines if a path
// in the [FileSet] must be ignored or not.
type Ignorer interface {
	IgnoreFile(path string) (bool, error)
	IgnoreDirectory(path string) (bool, error)
}

// nopIgnorer implements an [Ignorer] that doesn't ignore anything.
type nopIgnorer struct{}

func (nopIgnorer) IgnoreFile(path string) (bool, error) {
	return false, nil
}

func (nopIgnorer) IgnoreDirectory(path string) (bool, error) {
	return false, nil
}
