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

// Maximum length in bytes of the notebook header.
const headerLength = 32

// readHeader reads the first N bytes from a file.
func readHeader(fsys fs.FS, name string) ([]byte, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	// Scan header line with some padding.
	var buf = make([]byte, headerLength)
	n, err := f.Read([]byte(buf))
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Trim buffer to actual read bytes.
	return buf[:n], nil
}

// Detect returns whether the file at path is a Databricks notebook.
// If it is, it returns the notebook language.
func DetectWithFS(fsys fs.FS, name string) (notebook bool, language workspace.Language, err error) {
	header := ""

	buf, err := readHeader(fsys, name)
	if err != nil {
		return false, "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(buf))
	scanner.Scan()
	fileHeader := scanner.Text()

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
