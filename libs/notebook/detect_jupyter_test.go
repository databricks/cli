package notebook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectJupyter(t *testing.T) {
	var nb bool
	var lang workspace.Language
	var err error

	nb, lang, err = DetectJupyter("./testdata/py_ipynb.ipynb")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguagePython, lang)

	nb, lang, err = DetectJupyter("./testdata/r_ipynb.ipynb")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguageR, lang)

	nb, lang, err = DetectJupyter("./testdata/scala_ipynb.ipynb")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguageScala, lang)

	nb, lang, err = DetectJupyter("./testdata/sql_ipynb.ipynb")
	require.NoError(t, err)
	assert.True(t, nb)
	assert.Equal(t, workspace.LanguageSql, lang)
}

func TestDetectJupyterInvalidJSON(t *testing.T) {
	// Create garbage file.
	dir := t.TempDir()
	path := filepath.Join(dir, "file.ipynb")
	buf := make([]byte, 128)
	err := os.WriteFile(path, buf, 0o644)
	require.NoError(t, err)

	// Garbage contents means not a notebook.
	nb, _, err := DetectJupyter(path)
	require.ErrorContains(t, err, "error loading Jupyter notebook file")
	assert.False(t, nb)
}

func TestDetectJupyterNoCells(t *testing.T) {
	// Create empty JSON file.
	dir := t.TempDir()
	path := filepath.Join(dir, "file.ipynb")
	buf := []byte("{}")
	err := os.WriteFile(path, buf, 0o644)
	require.NoError(t, err)

	// Garbage contents means not a notebook.
	nb, _, err := DetectJupyter(path)
	require.ErrorContains(t, err, "invalid Jupyter notebook file")
	assert.False(t, nb)
}

func TestDetectJupyterOldVersion(t *testing.T) {
	// Create empty JSON file.
	dir := t.TempDir()
	path := filepath.Join(dir, "file.ipynb")
	buf := []byte(`{ "cells": [], "metadata": {}, "nbformat": 3 }`)
	err := os.WriteFile(path, buf, 0o644)
	require.NoError(t, err)

	// Garbage contents means not a notebook.
	nb, _, err := DetectJupyter(path)
	require.ErrorContains(t, err, "unsupported Jupyter notebook version")
	assert.False(t, nb)
}
