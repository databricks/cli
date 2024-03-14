package fileset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotebookFileIsNotebook(t *testing.T) {
	f := NewNotebookFile(nil, "", "")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.True(t, isNotebook)
}

func TestSourceFileIsNotNotebook(t *testing.T) {
	f := NewSourceFile(nil, "", "")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.False(t, isNotebook)
}

func touch(t *testing.T, path, file string) {
	os.MkdirAll(path, 0755)
	f, err := os.Create(filepath.Join(path, file))
	require.NoError(t, err)
	f.Close()
}

func touchNotebook(t *testing.T, path, file string) {
	os.MkdirAll(path, 0755)
	f, err := os.Create(filepath.Join(path, file))
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(path, file), []byte("# Databricks notebook source"), 0644)
	require.NoError(t, err)
	f.Close()
}

func TestUnknownFileDetectsNotebook(t *testing.T) {
	tmpDir := t.TempDir()
	touch(t, tmpDir, "test.py")
	touchNotebook(t, tmpDir, "notebook.py")

	f := NewFile(nil, filepath.Join(tmpDir, "test.py"), "test.py")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.False(t, isNotebook)

	f = NewFile(nil, filepath.Join(tmpDir, "notebook.py"), "notebook.py")
	isNotebook, err = f.IsNotebook()
	require.NoError(t, err)
	require.True(t, isNotebook)
}
