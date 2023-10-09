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

func (i *includer) IgnoreDirectory(path string) (bool, error) {
	return false, nil
}

func (i *includer) ignore(path string) bool {
	matched := i.matcher.MatchesPath(path)
	// If matched, do not ignore the file because we want to include it
	return !matched
}
