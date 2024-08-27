package config

type Sync struct {
	// Paths contains a list of paths to synchronize relative to the bundle root path.
	// If not configured, this defaults to synchronizing everything in the bundle root path (i.e. `.`).
	Paths []string `json:"paths,omitempty"`

	// Include contains a list of globs evaluated relative to the bundle root path
	// to explicitly include files that were excluded by the user's gitignore.
	Include []string `json:"include,omitempty"`

	// Exclude contains a list of globs evaluated relative to the bundle root path
	// to explicitly exclude files that were included by
	// 1) the default that observes the user's gitignore, or
	// 2) the `Include` field above.
	Exclude []string `json:"exclude,omitempty"`
}
