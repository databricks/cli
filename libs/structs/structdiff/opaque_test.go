package structdiff

import (
	"reflect"
	"testing"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
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
