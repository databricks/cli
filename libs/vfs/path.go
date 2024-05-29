package vfs

import "io/fs"

// FS combines the fs.FS, fs.StatFS, fs.ReadDirFS, and fs.ReadFileFS interfaces.
// It mandates that Path implementations must support all these interfaces.
type FS interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
}

// Path defines a read-only virtual file system interface for:
//
// 1. Intercepting file operations to inject custom logic (e.g., logging, access control).
// 2. Traversing directories to find specific leaf directories (e.g., .git).
// 3. Converting virtual paths to OS-native paths.
//
// Options 2 and 3 are not possible with the standard fs.FS interface.
// They are needed such that we can provide an instance to the sync package
// and still detect the containing .git directory and convert paths to native paths.
type Path interface {
	FS

	Parent() Path

	Native() string
}
