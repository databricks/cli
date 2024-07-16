package sync

import (
	"path"
	"sort"
)

// DirSet is a set of directories.
type DirSet map[string]struct{}

// MakeDirSet turns a list of file paths into the complete set of directories
// that is needed to store them (including parent directories).
func MakeDirSet(files []string) DirSet {
	out := map[string]struct{}{}

	// Iterate over all files.
	for _, f := range files {
		// Get the directory of the file.
		dir := path.Dir(f)

		// Add this directory and its parents until it is either "." or already in the set.
		for dir != "." {
			if _, ok := out[dir]; ok {
				break
			}
			out[dir] = struct{}{}
			dir = path.Dir(dir)
		}
	}

	return out
}

// Slice returns a sorted copy of the dirset elements as a slice.
func (dirset DirSet) Slice() []string {
	out := make([]string, 0, len(dirset))
	for dir := range dirset {
		out = append(out, dir)
	}
	sort.Strings(out)
	return out
}

// Remove returns the set difference of two DirSets.
func (dirset DirSet) Remove(other DirSet) DirSet {
	out := map[string]struct{}{}
	for dir := range dirset {
		if _, ok := other[dir]; !ok {
			out[dir] = struct{}{}
		}
	}
	return out
}
