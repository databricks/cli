package main

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuildMarkdownAnchors(t *testing.T) {
	nodes := []rootNode{
		{
			Title:       "some_field",
			TopLevel:    true,
			Type:        "Map",
			Description: "This is a description",
			Attributes: []attributeNode{
				{
					Title:       "my_attribute",
					Type:        "Map",
					Description: "Desc with link",
					Link:        "some_field._name_.my_attribute",
				},
			},
		},
		{
			Title:       "some_field._name_.my_attribute",
			TopLevel:    false,
			Type:        "Boolean",
			Description: "Another description",
		},
	}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "output.md")

	err := buildMarkdown(nodes, path, "Header")
	require.NoError(t, err)

	expected := testutil.ReadFile(t, "testdata/anchors.md")
	testutil.AssertFileContents(t, path, expected)
}
