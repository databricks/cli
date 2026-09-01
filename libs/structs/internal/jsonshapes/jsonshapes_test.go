package jsonshapes_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/internal/jsonshapes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpusMatchesEncodingJSON keeps the corpus honest: every shape's declared JSONFields
// are what encoding/json actually emits, and nothing it declares unreachable shows up.
// Without this the corpus could teach every consumer the same wrong answer.
func TestCorpusMatchesEncodingJSON(t *testing.T) {
	for _, shape := range jsonshapes.Shapes() {
		t.Run(shape.Name, func(t *testing.T) {
			blob, err := json.Marshal(shape.Value)
			require.NoError(t, err)

			got, err := jsonshapes.Leaves(shape.Value)
			require.NoError(t, err)

			var names []string
			for name := range got {
				names = append(names, name)
			}
			slices.Sort(names)
			want := append([]string(nil), shape.JSONFields...)
			slices.Sort(want)
			assert.Equal(t, want, names, "encoding/json emitted %s", blob)

			for _, name := range shape.Unreachable {
				assert.NotContains(t, got, name,
					"%q is declared unreachable but encoding/json emitted it", name)
			}
		})
	}
}
