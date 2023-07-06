package filer

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/exp/slices"
)

// LocalClient implements the [Filer] interface for the local filesystem.
type LocalClient struct {
	// File operations will be relative to this path.
	root localRootPath
}

func NewLocalClient(root string) (Filer, error) {
	return &LocalClient{
		root: NewLocalRootPath(root),
	}, nil
}

func (w *LocalClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	flags := os.O_WRONLY | os.O_CREATE
	if slices.Contains(mode, OverwriteIfExists) {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	absPath = filepath.FromSlash(absPath)
	f, err := os.OpenFile(absPath, flags, 0644)
	if os.IsNotExist(err) && slices.Contains(mode, CreateParentDirectories) {
		// Create parent directories if they don't exist.
		err = os.MkdirAll(filepath.Dir(absPath), 0755)
		if err != nil {
			return err
		}
		// Try again.
		f, err = os.OpenFile(absPath, flags, 0644)
	}

	if err != nil {
		switch {
		case os.IsNotExist(err):
			return NoSuchDirectoryError{path: absPath}
		case os.IsExist(err):
			return FileAlreadyExistsError{path: absPath}
		default:
			return err
		}
	}

	_, err = io.Copy(f, reader)
	cerr := f.Close()
	if err == nil {
		err = cerr
	}

	return err

}

func (w *LocalClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	// This stat call serves two purposes:
	// 1. Checks file at path exists, and throws an error if it does not
	// 2. Allows us to error out if the path is a directory
	absPath = filepath.FromSlash(absPath)
	stat, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, FileDoesNotExistError{path: absPath}
		}
		return nil, err
	}

	if stat.IsDir() {
		return nil, NotAFile{path: absPath}
	}

	return os.Open(absPath)
}

func (w *LocalClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return CannotDeleteRootError{}
	}

	absPath = filepath.FromSlash(absPath)
	err = os.Remove(absPath)

	// Return early on success.
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) {
		return FileDoesNotExistError{path: absPath}
	}

	if os.IsExist(err) {
		if slices.Contains(mode, DeleteRecursively) {
			return os.RemoveAll(absPath)
		}
		return DirectoryNotEmptyError{path: absPath}
	}

	return err
}

func (w *LocalClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	absPath = filepath.FromSlash(absPath)
	stat, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NoSuchDirectoryError{path: absPath}
		}
		return nil, err
	}

	if !stat.IsDir() {
		return nil, NotADirectory{path: absPath}
	}

	return os.ReadDir(absPath)
}

func (w *LocalClient) Mkdir(ctx context.Context, name string) error {
	dirPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	dirPath = filepath.FromSlash(dirPath)
	return os.MkdirAll(dirPath, 0755)
}

func (w *LocalClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	absPath = filepath.FromSlash(absPath)
	stat, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return nil, FileDoesNotExistError{path: absPath}
	}
	return stat, err
}
