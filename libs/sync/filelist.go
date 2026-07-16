package sync

import (
	"context"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/set"
	"github.com/databricks/cli/libs/vfs"
)

// FileList enumerates the local files selected for sync: git-tracked files,
// plus explicit includes, minus excludes. It is the "what would be synced"
// listing concern, independent of the sync snapshot and remote upload.
type FileList struct {
	fileSet        *git.FileSet
	includeFileSet *fileset.FileSet
	excludeFileSet *fileset.FileSet
}

// NewFileList builds a FileList for the directory tree at root, located within
// the Git worktree at worktreeRoot so gitignore rules are honored. paths selects
// which entries under root to list; empty paths select nothing, so callers that
// want the whole root must pass ["."] explicitly.
func NewFileList(ctx context.Context, worktreeRoot, root vfs.Path, paths, include, exclude []string) (*FileList, error) {
	fileSet, err := git.NewFileSet(ctx, worktreeRoot, root, paths)
	if err != nil {
		return nil, err
	}

	includeFileSet, err := fileset.NewGlobSet(root, include)
	if err != nil {
		return nil, err
	}

	excludeFileSet, err := fileset.NewGlobSet(root, exclude)
	if err != nil {
		return nil, err
	}

	return &FileList{
		fileSet:        fileSet,
		includeFileSet: includeFileSet,
		excludeFileSet: excludeFileSet,
	}, nil
}

// Files returns the deduped set of files (git ∪ include) − exclude.
// Files are deduped by their relative path; the order is whatever the
// underlying set yields.
func (l *FileList) Files(ctx context.Context) ([]fileset.File, error) {
	all := set.NewSetF(func(f fileset.File) string {
		return f.Relative
	})
	gitFiles, err := l.fileSet.Files()
	if err != nil {
		log.Errorf(ctx, "cannot list files: %s", err)
		return nil, err
	}

	all.Add(gitFiles...)

	include, err := l.includeFileSet.Files()
	if err != nil {
		log.Errorf(ctx, "cannot list include files: %s", err)
		return nil, err
	}

	all.Add(include...)

	exclude, err := l.excludeFileSet.Files()
	if err != nil {
		log.Errorf(ctx, "cannot list exclude files: %s", err)
		return nil, err
	}

	for _, f := range exclude {
		all.Remove(f)
	}

	return all.Iter(), nil
}
