package testutil

import (
	"os"
	"path/filepath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TouchNotebook(t TestingT, elems ...string) string {
	path := filepath.Join(elems...)
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(path, []byte("# Databricks notebook source"), 0o644)
	require.NoError(t, err)
	return path
}

func Touch(t TestingT, elems ...string) string {
	path := filepath.Join(elems...)
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)

	f, err := os.Create(path)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)
	return path
}

// WriteFile writes content to a file.
func WriteFile(t TestingT, path, content string) {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)

	f, err := os.Create(path)
	require.NoError(t, err)

	_, err = f.WriteString(content)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)
}

// ReadFile reads a file and returns its content as a string.
func ReadFile(t TestingT, path string) string {
	b, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(b)
}

// StatFile returns the file info for a file.
func StatFile(t TestingT, path string) os.FileInfo {
	fi, err := os.Stat(path)
	require.NoError(t, err)

	return fi
}

// AssertFileContents asserts that the file at path has the expected content.
func AssertFileContents(t TestingT, path, expected string) bool {
	actual := ReadFile(t, path)
	return assert.Equal(t, expected, actual)
}

// AssertFilePermissions asserts that the file at path has the expected permissions.
func AssertFilePermissions(t TestingT, path string, expected os.FileMode) bool {
	fi := StatFile(t, path)
	assert.False(t, fi.Mode().IsDir(), "expected a file, got a directory")
	return assert.Equal(t, expected, fi.Mode().Perm(), "expected 0%o, got 0%o", expected, fi.Mode().Perm())
}

// AssertDirPermissions asserts that the file at path has the expected permissions.
func AssertDirPermissions(t TestingT, path string, expected os.FileMode) bool {
	fi := StatFile(t, path)
	assert.True(t, fi.Mode().IsDir(), "expected a directory, got a file")
	return assert.Equal(t, expected, fi.Mode().Perm(), "expected 0%o, got 0%o", expected, fi.Mode().Perm())
}
