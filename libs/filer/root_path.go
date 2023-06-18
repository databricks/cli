package filer

type RootPath interface {
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
