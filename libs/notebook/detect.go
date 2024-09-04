package notebook

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// FileInfoWithWorkspaceObjectInfo is an interface implemented by [fs.FileInfo] values that
// contain a file's underlying [workspace.ObjectInfo].
//
// This may be the case when working with a [filer.Filer] backed by the workspace API.
// For these files we do not need to read a file's header to know if it is a notebook;
// we can use the [workspace.ObjectInfo] value directly.
type FileInfoWithWorkspaceObjectInfo interface {
	WorkspaceObjectInfo() workspace.ObjectInfo
}

// Maximum length in bytes of the notebook header.
const headerLength = 32

// file wraps an fs.File and implements a few helper methods such that
// they don't need to be inlined in the [DetectWithFS] function below.
type file struct {
	f fs.File
}

func openFile(fsys fs.FS, name string) (*file, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}

	return &file{f: f}, nil
}

func (f file) close() error {
	return f.f.Close()
}

func (f file) readHeader() (string, error) {
	// Scan header line with some padding.
	var buf = make([]byte, headerLength)
	n, err := f.f.Read([]byte(buf))
	if err != nil && err != io.EOF {
		return "", err
	}

	// Trim buffer to actual read bytes.
	buf = buf[:n]

	// Read the first line from the buffer.
	scanner := bufio.NewScanner(bytes.NewReader(buf))
	scanner.Scan()
	return scanner.Text(), nil
}

// getObjectInfo returns the [workspace.ObjectInfo] for the file if it is
// part of the [fs.FileInfo] value returned by the [fs.Stat] call.
func (f file) getObjectInfo() (oi workspace.ObjectInfo, ok bool, err error) {
	stat, err := f.f.Stat()
	if err != nil {
		return workspace.ObjectInfo{}, false, err
	}

	// Use object info if available.
	if i, ok := stat.(FileInfoWithWorkspaceObjectInfo); ok {
		return i.WorkspaceObjectInfo(), true, nil
	}

	return workspace.ObjectInfo{}, false, nil
}

// Detect returns whether the file at path is a Databricks notebook.
// If it is, it returns the notebook language.
func DetectWithFS(fsys fs.FS, name string) (notebook bool, language workspace.Language, err error) {
	header := ""

	f, err := openFile(fsys, name)
	if err != nil {
		return false, "", err
	}

	defer f.close()

	// Use object info if available.
	oi, ok, err := f.getObjectInfo()
	if err != nil {
		return false, "", err
	}
	if ok {
		return oi.ObjectType == workspace.ObjectTypeNotebook, oi.Language, nil
	}

	// Read the first line of the file.
	fileHeader, err := f.readHeader()
	if err != nil {
		return false, "", err
	}

	// Determine which header to expect based on filename extension.
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".py":
		header = `# Databricks notebook source`
		language = workspace.LanguagePython
	case ".r":
		header = `# Databricks notebook source`
		language = workspace.LanguageR
	case ".scala":
		header = "// Databricks notebook source"
		language = workspace.LanguageScala
	case ".sql":
		header = "-- Databricks notebook source"
		language = workspace.LanguageSql
	case ".ipynb":
		return DetectJupyterWithFS(fsys, name)
	default:
		return false, "", nil
	}

	if fileHeader != header {
		return false, "", nil
	}

	return true, language, nil
}

// Detect calls DetectWithFS with the local filesystem.
// The name argument may be a local relative path or a local absolute path.
func Detect(name string) (notebook bool, language workspace.Language, err error) {
	d := filepath.ToSlash(filepath.Dir(name))
	b := filepath.Base(name)
	return DetectWithFS(os.DirFS(d), b)
}

type inMemoryFile struct {
	content   []byte
	readIndex int64
}

type inMemoryFS struct {
	content []byte
}

func (f *inMemoryFile) Close() error {
	return nil
}

func (f *inMemoryFile) Stat() (fs.FileInfo, error) {
	return nil, nil
}

func (f *inMemoryFile) Read(b []byte) (n int, err error) {
	if f.readIndex >= int64(len(f.content)) {
		err = io.EOF
		return
	}

	n = copy(b, f.content[f.readIndex:])
	f.readIndex += int64(n)
	return
}

func (fs inMemoryFS) Open(name string) (fs.File, error) {
	return &inMemoryFile{
		content:   fs.content,
		readIndex: 0,
	}, nil
}

func DetectWithContent(name string, content []byte) (notebook bool, language workspace.Language, err error) {
	fs := inMemoryFS{
		content: content,
	}

	return DetectWithFS(fs, name)
}
