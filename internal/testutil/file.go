package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TouchNotebook(t *testing.T, elems ...string) string {
	path := filepath.Join(elems...)
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(path, []byte("# Databricks notebook source"), 0o644)
	require.NoError(t, err)
	return path
}

func Touch(t *testing.T, elems ...string) string {
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
func WriteFile(t *testing.T, path, content string) {
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
func ReadFile(t require.TestingT, path string) string {
	b, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(b)
}
