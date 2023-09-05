package config

import "path/filepath"

type Sync struct {
	// Include contains a list of globs evaluated relative to the bundle root path
	// to explicitly include files that were excluded by the user's gitignore.
	Include []string `json:"include,omitempty"`

	// Exclude contains a list of globs evaluated relative to the bundle root path
	// to explicitly exclude files that were included by
	// 1) the default that observes the user's gitignore, or
	// 2) the `Include` field above.
	Exclude []string `json:"exclude,omitempty"`
}

func (s *Sync) Merge(root *Root, other *Root) error {
	path, err := filepath.Rel(root.Path, other.Path)
	if err != nil {
		return err
	}
	for _, include := range other.Sync.Include {
		s.Include = append(s.Include, filepath.Join(path, include))
	}

	for _, exclude := range other.Sync.Exclude {
		s.Exclude = append(s.Exclude, filepath.Join(path, exclude))
	}

	return nil
}
