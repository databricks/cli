package main

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tgRoot struct {
	First  tgNested            `json:"first"`
	ByName map[string]tgNested `json:"by_name,omitempty"`
	Items  []tgItem            `json:"items,omitempty"`
	Plain  string              `json:"plain,omitempty"`
	Mode   tgMode              `json:"mode,omitempty"`
	Skip   string              `json:"-"`
	Hidden string              `json:"hidden,omitempty" bundle:"internal"`
}

type tgNested struct {
	Description string    `json:"description,omitempty"`
	Fields      string    `json:"fields,omitempty"`
	Again       *tgNested `json:"again,omitempty"`
}

type tgItem struct {
	tgEmbedded
	Name string `json:"name,omitempty"`
}

type tgEmbedded struct {
	Inner tgNested `json:"inner,omitempty"`
}

type tgMode string

func TestTypeGraph(t *testing.T) {
	g, err := newTypeGraph(reflect.TypeFor[tgRoot]())
	require.NoError(t, err)

	rootKey := getPath(reflect.TypeFor[tgRoot]())
	nestedKey := getPath(reflect.TypeFor[tgNested]())
	itemKey := getPath(reflect.TypeFor[tgItem]())
	modeKey := getPath(reflect.TypeFor[tgMode]())

	assert.Equal(t, rootKey, g.root)

	// Properties are in struct declaration order; map and slice fields resolve
	// to their element type; fields skipped by the schema generator are absent.
	assert.Equal(t, []fieldEdge{
		{name: "first", typ: nestedKey},
		{name: "by_name", typ: nestedKey},
		{name: "items", typ: itemKey},
		{name: "plain", typ: ""},
		{name: "mode", typ: modeKey},
	}, g.fields[rootKey])

	// Top-level fields precede embedded ones, matching the schema generator.
	assert.Equal(t, []fieldEdge{
		{name: "name", typ: ""},
		{name: "inner", typ: nestedKey},
	}, g.fields[itemKey])

	// Cyclic self-reference resolves to the type itself.
	assert.Equal(t, []fieldEdge{
		{name: "description", typ: ""},
		{name: "fields", typ: ""},
		{name: "again", typ: nestedKey},
	}, g.fields[nestedKey])

	// Named string types (enums) are annotatable types without properties.
	edges, ok := g.fields[modeKey]
	assert.True(t, ok)
	assert.Empty(t, edges)
}
