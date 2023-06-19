package filer

// RootPath can be joined with a relative path and ensures that
// the returned path is always a strict child of the root path.
type RootPath interface {
	// Join returns the specified path name joined to the root.
	// It returns an error if the resulting path is not a strict child of the root path.
	Join(string) (string, error)

	Root() string
}

type NopRootPath struct{}

func (rp NopRootPath) Join(name string) (string, error) {
	return name, nil
}

func (rp NopRootPath) Root() string {
	return ""
}
