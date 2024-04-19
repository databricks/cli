package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TouchNotebook(t *testing.T, elems ...string) string {
	path := filepath.Join(elems...)
	os.MkdirAll(filepath.Dir(path), 0755)
	err := os.WriteFile(path, []byte("# Databricks notebook source"), 0644)
	require.NoError(t, err)
	return path
}

func Touch(t *testing.T, elems ...string) string {
	path := filepath.Join(elems...)
	os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
	return path
}
