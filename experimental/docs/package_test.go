package main_test

import (
	"testing"

	main "github.com/databricks/cli/experimental/docs"
	"github.com/stretchr/testify/require"
)

func TestPackagesAll(t *testing.T) {
	pkgs := main.Packages()
	require.NotEmpty(t, pkgs)
}
