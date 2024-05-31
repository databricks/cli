package fileset

import (
	"io/fs"
	"time"

	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/vfs"
)

type fileType int

const (
	Unknown  fileType = iota
	Notebook          // Databricks notebook file
	Source            // Any other file type
)

type File struct {
	// Root path of the fileset.
	root vfs.Path

	// File entry as returned by the [fs.WalkDir] function.
	entry fs.DirEntry

	// Type of the file.
	fileType fileType

	// Relative path within the fileset.
	// Combine with the [vfs.Path] to interact with the underlying file.
	Relative string
}

func NewNotebookFile(root vfs.Path, entry fs.DirEntry, relative string) File {
	return File{
		root:     root,
		entry:    entry,
		fileType: Notebook,
		Relative: relative,
	}
}

func NewFile(root vfs.Path, entry fs.DirEntry, relative string) File {
	return File{
		root:     root,
		entry:    entry,
		fileType: Unknown,
		Relative: relative,
	}
}

func NewSourceFile(root vfs.Path, entry fs.DirEntry, relative string) File {
	return File{
		root:     root,
		entry:    entry,
		fileType: Source,
		Relative: relative,
	}
}

func (f File) Modified() (ts time.Time) {
	info, err := f.entry.Info()
	if err != nil {
		// return default time, beginning of epoch
		return ts
	}
	return info.ModTime()
}

func (f *File) IsNotebook() (bool, error) {
	if f.fileType != Unknown {
		return f.fileType == Notebook, nil
	}

	// Otherwise, detect the notebook type.
	isNotebook, _, err := notebook.DetectWithFS(f.root, f.Relative)
	if err != nil {
		return false, err
	}
	if isNotebook {
		f.fileType = Notebook
	} else {
		f.fileType = Source
	}
	return isNotebook, nil
}
