package structdiff_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/structs/structdiff"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	sdkfieldmask "github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type structWithDuration struct {
	Name    string
	Timeout *sdkduration.Duration `json:"timeout,omitempty"`
}

type structWithTime struct {
	Name      string
	CreatedAt *sdktime.Time `json:"created_at,omitempty"`
}

type structWithFieldMask struct {
	Name  string
	Paths *sdkfieldmask.FieldMask `json:"paths,omitempty"`
}

func TestGetStructDiffSDKNativeTypes(t *testing.T) {
	dur5m := sdkduration.New(5 * time.Minute)
	dur10m := sdkduration.New(10 * time.Minute)
	dur2m := sdkduration.New(2 * time.Minute)
	dur120s := sdkduration.New(120 * time.Second)

	ts1 := sdktime.New(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	ts2 := sdktime.New(time.Date(2024, 1, 16, 10, 30, 0, 0, time.UTC))

	fm1 := sdkfieldmask.New([]string{"field1"})
	fm2 := sdkfieldmask.New([]string{"field1", "field2"})

	tests := []struct {
		name        string
		a, b        any
		wantChanges int
		wantPath    string
		wantOld     any
		wantNew     any
	}{
		// Duration
		{
			name:        "duration/same_value",
			a:           structWithDuration{Timeout: dur5m},
			b:           structWithDuration{Timeout: dur5m},
			wantChanges: 0,
		},
		{
			name:        "duration/different_value",
			a:           structWithDuration{Timeout: dur5m},
			b:           structWithDuration{Timeout: dur10m},
			wantChanges: 1,
			wantPath:    "timeout",
			wantOld:     "300s",
			wantNew:     "600s",
		},
		{
			name:        "duration/equivalent_values",
			a:           structWithDuration{Timeout: dur2m},
			b:           structWithDuration{Timeout: dur120s},
			wantChanges: 0,
		},
		{
			name:        "duration/add_field",
			a:           structWithDuration{Timeout: nil},
			b:           structWithDuration{Timeout: dur5m},
			wantChanges: 1,
			wantPath:    "timeout",
			wantOld:     nil,
			wantNew:     "300s",
		},
		{
			name:        "duration/remove_field",
			a:           structWithDuration{Timeout: dur5m},
			b:           structWithDuration{Timeout: nil},
			wantChanges: 1,
			wantPath:    "timeout",
			wantOld:     "300s",
			wantNew:     nil,
		},
		{
			name:        "duration/both_nil",
			a:           structWithDuration{Timeout: nil},
			b:           structWithDuration{Timeout: nil},
			wantChanges: 0,
		},

		// Time
		{
			name:        "time/same_value",
			a:           structWithTime{CreatedAt: ts1},
			b:           structWithTime{CreatedAt: ts1},
			wantChanges: 0,
		},
		{
			name:        "time/different_value",
			a:           structWithTime{CreatedAt: ts1},
			b:           structWithTime{CreatedAt: ts2},
			wantChanges: 1,
			wantPath:    "created_at",
			wantOld:     "2024-01-15T10:30:00Z",
			wantNew:     "2024-01-16T10:30:00Z",
		},

		// FieldMask
		{
			name:        "fieldmask/same_value",
			a:           structWithFieldMask{Paths: fm2},
			b:           structWithFieldMask{Paths: fm2},
			wantChanges: 0,
		},
		{
			name:        "fieldmask/different_value",
			a:           structWithFieldMask{Paths: fm1},
			b:           structWithFieldMask{Paths: fm2},
			wantChanges: 1,
			wantPath:    "paths",
			wantOld:     "field1",
			wantNew:     "field1,field2",
		},
		{
			name:        "fieldmask/add_field",
			a:           structWithFieldMask{Paths: nil},
			b:           structWithFieldMask{Paths: fm2},
			wantChanges: 1,
			wantPath:    "paths",
			wantOld:     nil,
			wantNew:     "field1,field2",
		},
		{
			name:        "fieldmask/remove_field",
			a:           structWithFieldMask{Paths: fm2},
			b:           structWithFieldMask{Paths: nil},
			wantChanges: 1,
			wantPath:    "paths",
			wantOld:     "field1,field2",
			wantNew:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := structdiff.GetStructDiff(tt.a, tt.b, nil)
			require.NoError(t, err)
			require.Len(t, changes, tt.wantChanges)

			if tt.wantChanges > 0 {
				assert.Equal(t, tt.wantPath, changes[0].Path.String())
				assert.Equal(t, tt.wantOld, changes[0].Old)
				assert.Equal(t, tt.wantNew, changes[0].New)
			}
		})
	}
}

func TestIsEqualSDKNativeTypes(t *testing.T) {
	dur5m := sdkduration.New(5 * time.Minute)
	dur10m := sdkduration.New(10 * time.Minute)
	dur2m := sdkduration.New(2 * time.Minute)
	dur120s := sdkduration.New(120 * time.Second)

	ts1 := sdktime.New(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	ts2 := sdktime.New(time.Date(2024, 1, 16, 10, 30, 0, 0, time.UTC))

	fm1 := sdkfieldmask.New([]string{"field1"})
	fm2 := sdkfieldmask.New([]string{"field1", "field2"})

	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"duration/same_value", structWithDuration{Timeout: dur5m}, structWithDuration{Timeout: dur5m}, true},
		{"duration/different_value", structWithDuration{Timeout: dur5m}, structWithDuration{Timeout: dur10m}, false},
		{"duration/equivalent_values", structWithDuration{Timeout: dur2m}, structWithDuration{Timeout: dur120s}, true},
		{"time/same_value", structWithTime{CreatedAt: ts1}, structWithTime{CreatedAt: ts1}, true},
		{"time/different_value", structWithTime{CreatedAt: ts1}, structWithTime{CreatedAt: ts2}, false},
		{"fieldmask/same_value", structWithFieldMask{Paths: fm2}, structWithFieldMask{Paths: fm2}, true},
		{"fieldmask/different_value", structWithFieldMask{Paths: fm1}, structWithFieldMask{Paths: fm2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, structdiff.IsEqual(tt.a, tt.b))
		})
	}
}
