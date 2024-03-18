package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TouchNotebook(t *testing.T, path, file string) {
	os.MkdirAll(path, 0755)
	f, err := os.Create(filepath.Join(path, file))
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(path, file), []byte("# Databricks notebook source"), 0644)
	require.NoError(t, err)
	f.Close()
}

func Touch(t *testing.T, path, file string) {
	os.MkdirAll(path, 0755)
	f, err := os.Create(filepath.Join(path, file))
	require.NoError(t, err)
	f.Close()
}
