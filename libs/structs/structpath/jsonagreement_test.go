package structpath_test

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMapKeyRoundTripsThroughRendering checks that a path built from a JSON object key
// survives being rendered and parsed again. Every libs/structs package identifies a field
// by the rendered string, and the corpus tests compare those strings across packages, so a
// key that renders to something ParsePath reads back differently silently makes two
// packages "disagree" about a field they both found.
//
// The keys here are the ones a Databricks API can really return: a Spark conf key has dots,
// a tag value can contain almost anything.
func TestMapKeyRoundTripsThroughRendering(t *testing.T) {
	keys := []string{
		"simple",
		"spark.databricks.delta.retentionDurationCheck.enabled",
		"with space",
		"with'quote",
		"with\"doublequote",
		"with[bracket]",
		"with.dot",
		"",
		"ünïcode",
		"123",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			node := structpath.NewBracketString(structpath.NewStringKey(nil, "conf"), key)
			rendered := node.String()

			parsed, err := structpath.ParsePath(rendered)
			require.NoError(t, err, "rendered as %q", rendered)
			assert.Equal(t, rendered, parsed.String(),
				"%q does not survive render -> parse -> render", key)
		})
	}
}

// TestIndexRoundTripsThroughRendering does the same for a slice index, which is how every
// package refers to an element of a JSON array.
func TestIndexRoundTripsThroughRendering(t *testing.T) {
	for _, index := range []int{0, 1, 9, 10, 12345} {
		node := structpath.NewIndex(structpath.NewStringKey(nil, "tasks"), index)
		rendered := node.String()

		parsed, err := structpath.ParsePath(rendered)
		require.NoError(t, err, "rendered as %q", rendered)
		assert.Equal(t, rendered, parsed.String())
	}
}

// TestRenderedPathAddressesTheSameJSONMember pins the dialect against encoding/json: the
// rendering of a struct field is the object key itself, and the rendering of a map entry is
// the key in brackets. The two are different syntax for the same kind of JSON member, which
// is exactly the distinction a flattener has to get right to compare paths at all.
func TestRenderedPathAddressesTheSameJSONMember(t *testing.T) {
	type inner struct {
		Conf map[string]string `json:"conf,omitempty"`
	}
	value := &inner{Conf: map[string]string{"a.b": "v"}}

	blob, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `{"conf":{"a.b":"v"}}`, string(blob))

	field := structpath.NewStringKey(nil, "conf")
	assert.Equal(t, "conf", field.String())

	entry := structpath.NewBracketString(field, "a.b")
	assert.Equal(t, `conf['a.b']`, entry.String(),
		"a map entry must not render as a dotted field, or a path with a dotted key reads as two fields")
}
