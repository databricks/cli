package structprefixtree

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrefixTreeMatchesSample(t *testing.T) {
	tree, err := NewPrefixTreeFromMap(map[string]any{
		"*":                   "star",
		"grants":              "grants slice",
		"grants[*]":           "grant",
		"grants[*].principal": "principal",
		"foo":                 "foo",
		"foo[*]":              "foo slice",
		"foo[*].id":           "foo slice id",
		"foo.bar.id":          "foo bar id",
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		path  string
		value any
	}{
		{
			name:  "fallback wildcard",
			path:  "some_path",
			value: "star",
		},
		{
			name:  "direct match",
			path:  "grants",
			value: "grants slice",
		},
		{
			name:  "wildcard element",
			path:  "grants[0]",
			value: "grant",
		},
		{
			name:  "deep wildcard element",
			path:  "grants[1].principal",
			value: "principal",
		},
		{
			name:  "pattern input",
			path:  "grants[*].principal",
			value: "principal",
		},
		{
			name:  "deepest concrete key",
			path:  "foo.bar.id",
			value: "foo bar id",
		},
		{
			name:  "fallback to wildcard at same depth",
			path:  "foo[4].id",
			value: "foo slice id",
		},
		{
			name:  "fallback to element wildcard",
			path:  "foo[3]",
			value: "foo slice",
		},
		{
			name:  "fallback to root wildcard",
			path:  "bar",
			value: "star",
		},
		{
			name: "no match",
			path: "some_path.no_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, ok, err := tree.MatchString(tt.path)
			require.NoError(t, err)
			wantMatch := tt.value != nil
			assert.Equal(t, wantMatch, ok)
			if wantMatch {
				require.NotNil(t, node)
				assert.Equal(t, tt.value, node.Value())
			} else {
				assert.Nil(t, node)
			}
		})
	}
}

func TestChildPrefersExactThenWildcard(t *testing.T) {
	tree := NewPrefixTree()
	mustInsert(t, tree, "*", "star")
	mustInsert(t, tree, "foo", "foo")

	root := tree.Root
	require.NotNil(t, root)

	exact := root.Child(mustParse(t, "foo"))
	require.NotNil(t, exact)
	assert.Equal(t, "foo", exact.Value())
	assert.Equal(t, root, exact.Parent())

	wildcard := root.Child(mustParse(t, "bar"))
	require.NotNil(t, wildcard)
	assert.Equal(t, "star", wildcard.Value())
	assert.Equal(t, root, wildcard.Parent())
}

func TestDotStarAndBracketStarAreEquivalent(t *testing.T) {
	tree := NewPrefixTree()
	mustInsert(t, tree, "items[*].name", "value")

	mustMatch(t, tree, "items.*.name", "value")
	mustMatch(t, tree, "items[5].name", "value")
}

func TestRootValueMatch(t *testing.T) {
	tree := NewPrefixTree()
	_, err := tree.Insert(nil, "root")
	require.NoError(t, err)

	node, ok := tree.Match(nil)
	require.True(t, ok)
	assert.Equal(t, "root", node.Value())

	node, ok, err = tree.MatchString("any")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, node)
}

func TestWildcardMatchesKeyValueAndIndex(t *testing.T) {
	tree := NewPrefixTree()
	mustInsert(t, tree, "items.*.name", "value")

	mustMatch(t, tree, "items[task_key='foo'].name", "value")
	mustMatch(t, tree, "items[3].name", "value")
}

func TestPatternMustConsumeEntirePath(t *testing.T) {
	tree := NewPrefixTree()
	mustInsert(t, tree, "*", "star")

	node, ok, err := tree.MatchString("foo.bar")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, node)
}

func TestInsertRejectsIndexAndKeyValue(t *testing.T) {
	tree := NewPrefixTree()

	_, err := tree.InsertString("foo[1]", "x")
	require.Error(t, err)

	_, err = tree.InsertString("foo[key='value']", "x")
	require.Error(t, err)
}

func TestNewPrefixTreeFromMapRejectsPathWithIndex(t *testing.T) {
	_, err := NewPrefixTreeFromMap(map[string]any{
		"foo":    "ok",
		"foo[1]": "bad",
	})
	require.EqualError(t, err, `insert "foo[1]": array indexes are not supported in prefix tree keys`)
}

func TestNewPrefixTreeFromMapRejectsDuplicateWildcardPaths(t *testing.T) {
	_, err := NewPrefixTreeFromMap(map[string]any{
		"items.*":  "value-1",
		"items[*]": "value-2",
	})
	require.ErrorContains(t, err, `already exists`)
}

func TestInsertRejectsDuplicatePaths(t *testing.T) {
	tree := NewPrefixTree()
	path, err := structpath.Parse("foo.bar")
	require.NoError(t, err)

	_, err = tree.Insert(path, "value-1")
	require.NoError(t, err)

	_, err = tree.Insert(path, "value-2")
	require.Error(t, err)
}

func TestInsertStringRejectsDuplicatePaths(t *testing.T) {
	tree := NewPrefixTree()

	_, err := tree.InsertString("foo.bar", "value-1")
	require.NoError(t, err)

	_, err = tree.InsertString("foo.bar", "value-2")
	require.Error(t, err)
}

func mustParse(t *testing.T, path string) *structpath.PathNode {
	t.Helper()
	if path == "" {
		return nil
	}
	p, err := structpath.Parse(path)
	require.NoError(t, err)
	return p
}

func mustInsert(t *testing.T, tree *PrefixTree, path string, value any) {
	t.Helper()
	_, err := tree.InsertString(path, value)
	require.NoError(t, err)
}

func mustMatch(t *testing.T, tree *PrefixTree, path string, expected any) *Node {
	t.Helper()
	node, ok, err := tree.MatchString(path)
	require.NoError(t, err)
	require.True(t, ok, "expected match for %s", path)
	require.NotNil(t, node)
	assert.Equal(t, expected, node.Value())
	return node
}
