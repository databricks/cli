package fileset

import (
	"io/fs"
	"time"

	"github.com/databricks/cli/libs/notebook"
)

type fileType int

const (
	Unknown  fileType = iota
	Notebook          // Databricks notebook file
	Source            // Any other file type
)

type File struct {
	fs.DirEntry
	Absolute, Relative string
	fileType           fileType
}

func NewNotebookFile(entry fs.DirEntry, absolute string, relative string) File {
	return File{
		DirEntry: entry,
		Absolute: absolute,
		Relative: relative,
		fileType: Notebook,
	}
}

func NewFile(entry fs.DirEntry, absolute string, relative string) File {
	return File{
		DirEntry: entry,
		Absolute: absolute,
		Relative: relative,
		fileType: Unknown,
	}
}

func NewSourceFile(entry fs.DirEntry, absolute string, relative string) File {
	return File{
		DirEntry: entry,
		Absolute: absolute,
		Relative: relative,
		fileType: Source,
	}
}

func (f File) Modified() (ts time.Time) {
	info, err := f.Info()
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
	isNotebook, _, err := notebook.Detect(f.Absolute)
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
