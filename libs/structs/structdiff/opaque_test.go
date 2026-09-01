package structdiff

import (
	"reflect"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsOpaqueStruct(t *testing.T) {
	tests := []struct {
		typ  reflect.Type
		want bool
	}{
		// Payload hidden in an unexported protobuf pointer.
		{reflect.TypeFor[duration.Duration](), true},
		{reflect.TypeFor[sdktime.Time](), true},

		// Empty proto messages: no exported fields either, but nothing to hide
		// and no marshaler of their own.
		{reflect.TypeFor[catalog.MonitorSnapshot](), false},
		{reflect.TypeFor[jobs.ModelTriggerState](), false},
		{reflect.TypeFor[jobs.ScheduleTriggerState](), false},
		{reflect.TypeFor[pipelines.ManualTrigger](), false},

		// FieldMask marshals to a string but exposes Paths, so the walk sees it.
		{reflect.TypeFor[fieldmask.FieldMask](), false},

		{reflect.TypeFor[A](), false},
		{reflect.TypeFor[string](), false},
		{reflect.TypeFor[*duration.Duration](), false},
	}

	for _, tt := range tests {
		t.Run(tt.typ.String(), func(t *testing.T) {
			assert.Equal(t, tt.want, IsOpaqueStruct(tt.typ))
			// Second call goes through the cache.
			assert.Equal(t, tt.want, IsOpaqueStruct(tt.typ))
		})
	}
}

// Opaque values compared at the root of the call, rather than reached through a
// struct field. This is the shape the original bug was reported in: two
// durations handed straight to GetStructDiff came back equal.
func TestGetStructDiffOpaqueAtRoot(t *testing.T) {
	a := duration.New(300 * time.Second)
	b := duration.New(600 * time.Second)

	for _, tt := range []struct {
		name string
		a, b any
	}{
		{"pointer", a, b},
		{"value", *a, *b},
	} {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := GetStructDiff(tt.a, tt.b, nil)
			require.NoError(t, err)
			require.Len(t, changes, 1)
			assert.Empty(t, changes[0].Path.String())
			assert.False(t, IsEqual(tt.a, tt.b))
			assert.True(t, IsEqual(tt.a, tt.a))
		})
	}
}
