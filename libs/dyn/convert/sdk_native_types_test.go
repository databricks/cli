package convert

import (
	"reflect"
	"testing"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/sdk-go/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Roundtrip tests - verify SDK native types convert to dyn.Value and back for both value and pointer types

func TestDurationRoundtrip(t *testing.T) {
	tests := []struct {
		name           string
		duration       time.Duration
		expectedString string
	}{
		{"5min", 5 * time.Minute, "300s"},
		{"7days", 7 * 24 * time.Hour, "604800s"},
		{"1hour", 1 * time.Hour, "3600s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test value type
			t.Run("value", func(t *testing.T) {
				src := *types.NewFromDuration(tt.duration)
				dynValue, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out types.Duration
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				assert.Equal(t, src.AsDuration(), out.AsDuration())
			})

			// Test pointer type
			t.Run("pointer", func(t *testing.T) {
				src := *types.NewFromDuration(tt.duration)
				dynValue, err := FromTyped(&src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out *types.Duration
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				require.NotNil(t, out)
				assert.Equal(t, src.AsDuration(), out.AsDuration())
			})
		})
	}
}

func TestTimeRoundtrip(t *testing.T) {
	tests := []struct {
		name           string
		time           time.Time
		expectedString string
	}{
		{
			"no_nanos",
			time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
			"2023-12-25T10:30:00Z",
		},
		{
			"with_nanos",
			time.Date(2023, 12, 25, 10, 30, 0, 123456789, time.UTC),
			"2023-12-25T10:30:00.123456789Z",
		},
		{
			"epoch",
			time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			"1970-01-01T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test value type
			t.Run("value", func(t *testing.T) {
				src := *types.NewFromTime(tt.time)
				dynValue, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				if reflect.ValueOf(src).IsZero() {
					// The epoch is the zero value of types.Time; by value it means "unset".
					assert.Equal(t, dyn.NilValue, dynValue)
					return
				}
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out types.Time
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				assert.Equal(t, src.AsTime(), out.AsTime())
			})

			// Test pointer type
			t.Run("pointer", func(t *testing.T) {
				src := *types.NewFromTime(tt.time)
				dynValue, err := FromTyped(&src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out *types.Time
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				require.NotNil(t, out)
				assert.Equal(t, src.AsTime(), out.AsTime())
			})
		})
	}
}

func TestFieldMaskRoundtrip(t *testing.T) {
	tests := []struct {
		name           string
		paths          []string
		expectedString string
	}{
		{"single", []string{"name"}, "name"},
		{"multiple", []string{"name", "age", "email"}, "age,email,name"},
		{"nested", []string{"user.name", "user.email"}, "user.email,user.name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test value type
			t.Run("value", func(t *testing.T) {
				src := *newFieldMask(t, tt.paths...)
				dynValue, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out fieldMask
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				assert.Equal(t, src.String(), out.String())
			})

			// Test pointer type
			t.Run("pointer", func(t *testing.T) {
				src := *newFieldMask(t, tt.paths...)
				dynValue, err := FromTyped(&src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out *fieldMask
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				require.NotNil(t, out)
				assert.Equal(t, src.String(), out.String())
			})
		})
	}
}

// Edge case tests

func TestNilValues(t *testing.T) {
	tests := []struct {
		name string
		typ  any
	}{
		{"duration", (*types.Duration)(nil)},
		{"time", (*types.Time)(nil)},
		{"fieldmask", (*fieldMask)(nil)},
	}

	t.Run("from_typed", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				nv, err := FromTyped(tt.typ, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, dyn.NilValue, nv)
			})
		}
	})

	t.Run("normalize", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				vout, diags := Normalize(tt.typ, dyn.NilValue)
				assert.Len(t, diags, 1)
				assert.Equal(t, diag.Warning, diags[0].Severity)
				assert.Equal(t, `expected a string value, found null`, diags[0].Summary)
				assert.Equal(t, dyn.InvalidValue, vout)
			})
		}
	})
}

func TestToTypedErrors(t *testing.T) {
	wrongTypeInput := dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")})

	tests := []struct {
		name          string
		destValue     any
		input         dyn.Value
		expectedError string
	}{
		{
			name:          "duration_wrong_type",
			destValue:     new(types.Duration),
			input:         wrongTypeInput,
			expectedError: "expected a string",
		},
		{
			name:          "duration_invalid_format",
			destValue:     new(types.Duration),
			input:         dyn.V("7d"),
			expectedError: "parse duration",
		},
		{
			name:          "time_wrong_type",
			destValue:     new(types.Time),
			input:         wrongTypeInput,
			expectedError: "expected a string",
		},
		{
			name:          "time_invalid_format",
			destValue:     new(types.Time),
			input:         dyn.V("not-a-time"),
			expectedError: "parse timestamp",
		},
		{
			name:          "fieldmask_wrong_type",
			destValue:     new(fieldMask),
			input:         wrongTypeInput,
			expectedError: "expected a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ToTyped(tt.destValue, tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestFromTypedVariableReferences(t *testing.T) {
	tests := []struct {
		name string
		src  any
	}{
		{
			name: "duration",
			src:  *types.NewFromDuration(5 * time.Minute),
		},
		{
			name: "time",
			src:  *types.NewFromTime(time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)),
		},
		{
			name: "fieldmask",
			src:  *newFieldMask(t, []string{"name", "age"}...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := dyn.V("${var.foo}")
			nv, err := FromTyped(tt.src, ref)
			require.NoError(t, err)
			assert.Equal(t, dyn.V("${var.foo}"), nv)
		})
	}
}

func TestToTypedVariableReferences(t *testing.T) {
	tests := []struct {
		name string
		dst  any
		zero any
	}{
		{
			name: "duration",
			dst:  types.NewFromDuration(5 * time.Minute),
			zero: types.Duration{},
		},
		{
			name: "time",
			dst:  types.NewFromTime(time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)),
			zero: types.Time{},
		},
		{
			name: "fieldmask",
			dst:  newFieldMask(t, []string{"name", "age"}...),
			zero: fieldMask{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ToTyped(tt.dst, dyn.V("${var.foo}"))
			require.NoError(t, err)
			assert.Equal(t, tt.zero, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestSpecialCases(t *testing.T) {
	t.Run("duration_zero", func(t *testing.T) {
		var src types.Duration
		nv, err := FromTyped(src, dyn.NilValue)
		require.NoError(t, err)
		assert.Equal(t, dyn.NilValue, nv)
	})

	t.Run("fieldmask_empty_fromtyped", func(t *testing.T) {
		src := newFieldMask(t, []string{}...)
		nv, err := FromTyped(src, dyn.NilValue)
		require.NoError(t, err)
		// Empty field mask marshals to empty string
		assert.Equal(t, dyn.V(""), nv)
	})

	t.Run("fieldmask_empty_totyped", func(t *testing.T) {
		var out fieldMask
		err := ToTyped(&out, dyn.V(""))
		require.NoError(t, err)
		assert.Empty(t, out.String())
	})
}

// End-to-end tests with structs containing SDK native types

func TestSDKTypesRoundTripWithPostgresBranchSpec(t *testing.T) {
	type BranchSpec struct {
		ExpireTime       *types.Time     `json:"expire_time,omitempty"`
		SourceBranchTime *types.Time     `json:"source_branch_time,omitempty"`
		Ttl              *types.Duration `json:"ttl,omitempty"`
		IsProtected      bool            `json:"is_protected,omitempty"`
	}

	original := BranchSpec{
		ExpireTime:       types.NewFromTime(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)),
		SourceBranchTime: types.NewFromTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		Ttl:              types.NewFromDuration(7 * 24 * time.Hour),
		IsProtected:      true,
	}

	dynValue, err := FromTyped(original, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, "2024-12-31T23:59:59Z", dynValue.Get("expire_time").MustString())
	assert.Equal(t, "2024-01-01T00:00:00Z", dynValue.Get("source_branch_time").MustString())
	assert.Equal(t, "604800s", dynValue.Get("ttl").MustString())
	assert.True(t, dynValue.Get("is_protected").MustBool())

	var roundtrip BranchSpec
	err = ToTyped(&roundtrip, dynValue)
	require.NoError(t, err)
	require.NotNil(t, roundtrip.ExpireTime)
	require.NotNil(t, roundtrip.SourceBranchTime)
	require.NotNil(t, roundtrip.Ttl)
	assert.Equal(t, original.ExpireTime.AsTime(), roundtrip.ExpireTime.AsTime())
	assert.Equal(t, original.SourceBranchTime.AsTime(), roundtrip.SourceBranchTime.AsTime())
	assert.Equal(t, original.Ttl.AsDuration(), roundtrip.Ttl.AsDuration())
	assert.Equal(t, original.IsProtected, roundtrip.IsProtected)
}

func TestSDKTypesRoundTripWithUpdateRequest(t *testing.T) {
	type UpdateRequest struct {
		Name       string    `json:"name"`
		UpdateMask fieldMask `json:"update_mask"`
	}

	original := UpdateRequest{
		Name:       "projects/123/branches/456",
		UpdateMask: *newFieldMask(t, []string{"spec.ttl", "spec.is_protected"}...),
	}

	dynValue, err := FromTyped(original, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, "projects/123/branches/456", dynValue.Get("name").MustString())
	assert.Equal(t, "spec.is_protected,spec.ttl", dynValue.Get("update_mask").MustString())

	var roundtrip UpdateRequest
	err = ToTyped(&roundtrip, dynValue)
	require.NoError(t, err)
	assert.Equal(t, original.Name, roundtrip.Name)
	assert.Equal(t, "spec.is_protected,spec.ttl", roundtrip.UpdateMask.String())
}

func TestSDKTypesNormalizeWithStruct(t *testing.T) {
	type BranchSpec struct {
		ExpireTime  *types.Time     `json:"expire_time,omitempty"`
		Ttl         *types.Duration `json:"ttl,omitempty"`
		IsProtected bool            `json:"is_protected,omitempty"`
	}

	var typ BranchSpec
	vin := dyn.V(map[string]dyn.Value{
		"expire_time":  dyn.V("2024-12-31T23:59:59Z"),
		"ttl":          dyn.V("604800s"),
		"is_protected": dyn.V(true),
	})

	vout, diags := Normalize(typ, vin)
	assert.Empty(t, diags)
	assert.Equal(t, "2024-12-31T23:59:59Z", vout.Get("expire_time").MustString())
	assert.Equal(t, "604800s", vout.Get("ttl").MustString())
	assert.True(t, vout.Get("is_protected").MustBool())

	var out BranchSpec
	err := ToTyped(&out, vout)
	require.NoError(t, err)
	require.NotNil(t, out.ExpireTime)
	require.NotNil(t, out.Ttl)
	assert.Equal(t, time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), out.ExpireTime.AsTime())
	assert.Equal(t, 7*24*time.Hour, out.Ttl.AsDuration())
	assert.True(t, out.IsProtected)
}

type fieldMaskUser struct {
	Name  *string `fieldmask:"name"`
	Email *string `fieldmask:"email"`
}

type fieldMaskSpec struct {
	Ttl         *string `fieldmask:"ttl"`
	IsProtected *bool   `fieldmask:"is_protected"`
}

// fieldMaskTarget mirrors the fieldmask tags sdk-go generates on its models,
// which types.FieldMask validates paths against.
type fieldMaskTarget struct {
	Name  *string        `fieldmask:"name"`
	Age   *int           `fieldmask:"age"`
	Email *string        `fieldmask:"email"`
	User  *fieldMaskUser `fieldmask:"user"`
	Spec  *fieldMaskSpec `fieldmask:"spec"`
}

type fieldMask = types.FieldMask[fieldMaskTarget]

func newFieldMask(t *testing.T, paths ...string) *fieldMask {
	m, err := types.NewFieldMask[fieldMaskTarget](paths...)
	require.NoError(t, err)
	return m
}
