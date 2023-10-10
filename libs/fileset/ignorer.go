package fileset

import (
	ignore "github.com/sabhiram/go-gitignore"
)

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

type includer struct {
	matcher *ignore.GitIgnore
}

func newIncluder(includes []string) *includer {
	matcher := ignore.CompileIgnoreLines(includes...)
	return &includer{
		matcher,
	}
}

func (i *includer) IgnoreFile(path string) (bool, error) {
	return i.ignore(path), nil
}

// In the context of 'include' functionality, the Ignorer logic appears to be reversed:
// For patterns like 'foo/bar/' which intends to match directories only, we still need to traverse into the directory for potential file matches.
// Ignoring the directory entirely isn't an option, especially when dealing with patterns like 'foo/bar/*.go'.
// While this pattern doesn't target directories, it does match all Go files within them and ignoring directories not matching the pattern
// Will result in missing file matches.
// During the tree traversal process, we call 'IgnoreDirectory' on ".", "./foo", and "./foo/bar",
// all while applying the 'foo/bar/*.go' pattern. To handle this situation effectively, it requires to make the code more complex.
// This could mean generating various prefix patterns to facilitate the exclusion of directories from traversal.
// It's worth noting that, in this particular case, opting for a simpler logic results in a performance trade-off.
func (i *includer) IgnoreDirectory(path string) (bool, error) {
	return false, nil
}

func (i *includer) ignore(path string) bool {
	matched := i.matcher.MatchesPath(path)
	// If matched, do not ignore the file because we want to include it
	return !matched
}
