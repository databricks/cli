package structwalk

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/sdk-go/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sdk-go native types are JSON strings, so WalkType must treat them as leaves
// instead of reporting their Go fields (e.g. Time.Seconds) as paths.
func TestWalkTypeSDKNativeTypesAreLeaves(t *testing.T) {
	type S struct {
		Time *types.Time                `json:"time,omitempty"`
		Dur  types.Duration             `json:"dur"`
		Mask *types.FieldMask[struct{}] `json:"mask,omitempty"`
		Name *string                    `json:"name,omitempty"`
	}

	var paths []string
	err := WalkType(reflect.TypeFor[S](), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
		if path != nil {
			paths = append(paths, path.String())
		}
		return true
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"time", "dur", "mask", "name"}, paths)
}
