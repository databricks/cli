package dynvar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type referenceVectorsFile struct {
	Vectors []referenceVector `json:"vectors"`
}

type referenceVector struct {
	ID         string   `json:"id"`
	Input      string   `json:"input"`
	Match      bool     `json:"match"`
	References []string `json:"references,omitempty"`
	Pure       *bool    `json:"pure,omitempty"`
	Path       *string  `json:"path,omitempty"`
	PathOK     *bool    `json:"path_ok,omitempty"`
}

func loadReferenceVectors(t *testing.T) []referenceVector {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", "reference_vectors.json"))
	require.NoError(t, err)

	var file referenceVectorsFile
	require.NoError(t, json.Unmarshal(b, &file))
	require.NotEmpty(t, file.Vectors)

	return file.Vectors
}

func TestReferenceVectors(t *testing.T) {
	for _, v := range loadReferenceVectors(t) {
		t.Run(v.ID, func(t *testing.T) {
			ref, ok := NewRef(dyn.V(v.Input))
			assert.Equal(t, v.Match, ok, "NewRef match")

			if v.Match {
				require.NotEmpty(t, ref.References())
				if v.References != nil {
					assert.Equal(t, v.References, ref.References(), "References")
				}
			}

			if v.Pure != nil {
				assert.Equal(t, *v.Pure, IsPureVariableReference(v.Input), "IsPureVariableReference")
			}

			switch {
			case v.Path != nil:
				path, ok := PureReferenceToPath(v.Input)
				require.True(t, ok, "PureReferenceToPath")
				assert.Equal(t, dyn.MustPathFromString(*v.Path), path)
			case v.PathOK != nil:
				_, ok := PureReferenceToPath(v.Input)
				assert.Equal(t, *v.PathOK, ok, "PureReferenceToPath")
			}
		})
	}
}
