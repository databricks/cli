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

func readHeader(r io.Reader) (string, error) {
	// Scan header line with some padding.
	buf := make([]byte, headerLength)
	n, err := r.Read(buf)
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

	if strings.ToLower(filepath.Ext(name)) == ExtensionJupyter {
		return DetectJupyterWithFS(fsys, name)
	}

	// Read the first line of the file.
	fileHeader, err := readHeader(f.f)
	if err != nil {
		return false, "", err
	}

	notebook, language = detectSource(name, fileHeader)
	return notebook, language, nil
}

// DetectBytes is like [DetectWithFS] for file content that is already in memory.
func DetectBytes(name string, content []byte) (notebook bool, language workspace.Language, err error) {
	if strings.ToLower(filepath.Ext(name)) == ExtensionJupyter {
		return detectJupyter(name, bytes.NewReader(content))
	}

	fileHeader, err := readHeader(bytes.NewReader(content))
	if err != nil {
		return false, "", err
	}

	notebook, language = detectSource(name, fileHeader)
	return notebook, language, nil
}

// detectSource reports whether a source file with the given name and first line is a notebook.
func detectSource(name, fileHeader string) (notebook bool, language workspace.Language) {
	header := ""

	// Determine which header to expect based on filename extension.
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ExtensionPython:
		header = `# Databricks notebook source`
		language = workspace.LanguagePython
	case ExtensionR:
		header = `# Databricks notebook source`
		language = workspace.LanguageR
	case ExtensionScala:
		header = "// Databricks notebook source"
		language = workspace.LanguageScala
	case ExtensionSql:
		header = "-- Databricks notebook source"
		language = workspace.LanguageSql
	default:
		return false, ""
	}

	if fileHeader != header {
		return false, ""
	}

	return true, language
}

// Detect calls DetectWithFS with the local filesystem.
// The name argument may be a local relative path or a local absolute path.
func Detect(name string) (notebook bool, language workspace.Language, err error) {
	d := filepath.ToSlash(filepath.Dir(name))
	b := filepath.Base(name)
	return DetectWithFS(os.DirFS(d), b)
}
