package convert

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	sdkfieldmask "github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
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
				src := *sdkduration.New(tt.duration)
				dynValue, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out sdkduration.Duration
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				assert.Equal(t, src.AsDuration(), out.AsDuration())
			})

			// Test pointer type
			t.Run("pointer", func(t *testing.T) {
				src := *sdkduration.New(tt.duration)
				dynValue, err := FromTyped(&src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out *sdkduration.Duration
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
				src := *sdktime.New(tt.time)
				dynValue, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out sdktime.Time
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				assert.Equal(t, src.AsTime(), out.AsTime())
			})

			// Test pointer type
			t.Run("pointer", func(t *testing.T) {
				src := *sdktime.New(tt.time)
				dynValue, err := FromTyped(&src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out *sdktime.Time
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
		{"multiple", []string{"name", "age", "email"}, "name,age,email"},
		{"nested", []string{"user.name", "user.email"}, "user.name,user.email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test value type
			t.Run("value", func(t *testing.T) {
				src := *sdkfieldmask.New(tt.paths)
				dynValue, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out sdkfieldmask.FieldMask
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				assert.Equal(t, src.Paths, out.Paths)
			})

			// Test pointer type
			t.Run("pointer", func(t *testing.T) {
				src := *sdkfieldmask.New(tt.paths)
				dynValue, err := FromTyped(&src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				var out *sdkfieldmask.FieldMask
				err = ToTyped(&out, dynValue)
				require.NoError(t, err)
				require.NotNil(t, out)
				assert.Equal(t, src.Paths, out.Paths)
			})
		})
	}
}

// Edge case tests

func TestNilValuesFromTyped(t *testing.T) {
	t.Run("duration", func(t *testing.T) {
		var src *sdkduration.Duration
		nv, err := FromTyped(src, dyn.NilValue)
		require.NoError(t, err)
		assert.Equal(t, dyn.NilValue, nv)
	})

	t.Run("time", func(t *testing.T) {
		var src *sdktime.Time
		nv, err := FromTyped(src, dyn.NilValue)
		require.NoError(t, err)
		assert.Equal(t, dyn.NilValue, nv)
	})

	t.Run("fieldmask", func(t *testing.T) {
		var src *sdkfieldmask.FieldMask
		nv, err := FromTyped(src, dyn.NilValue)
		require.NoError(t, err)
		assert.Equal(t, dyn.NilValue, nv)
	})
}

func TestNilValuesNormalize(t *testing.T) {
	t.Run("duration", func(t *testing.T) {
		var typ *sdkduration.Duration
		vin := dyn.NilValue
		vout, diags := Normalize(typ, vin)
		assert.Len(t, diags, 1)
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Equal(t, `expected a string value, found null`, diags[0].Summary)
		assert.Equal(t, dyn.InvalidValue, vout)
	})

	t.Run("time", func(t *testing.T) {
		var typ *sdktime.Time
		vin := dyn.NilValue
		vout, diags := Normalize(typ, vin)
		assert.Len(t, diags, 1)
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Equal(t, `expected a string value, found null`, diags[0].Summary)
		assert.Equal(t, dyn.InvalidValue, vout)
	})

	t.Run("fieldmask", func(t *testing.T) {
		var typ *sdkfieldmask.FieldMask
		vin := dyn.NilValue
		vout, diags := Normalize(typ, vin)
		assert.Len(t, diags, 1)
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Equal(t, `expected a string value, found null`, diags[0].Summary)
		assert.Equal(t, dyn.InvalidValue, vout)
	})
}

func TestToTypedErrors(t *testing.T) {
	wrongTypeInput := dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")})

	t.Run("duration_wrong_type", func(t *testing.T) {
		var out sdkduration.Duration
		err := ToTyped(&out, wrongTypeInput)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected a string")
	})

	t.Run("duration_invalid_format", func(t *testing.T) {
		var out sdkduration.Duration
		err := ToTyped(&out, dyn.V("7d"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid google.protobuf.Duration value")
	})

	t.Run("time_wrong_type", func(t *testing.T) {
		var out sdktime.Time
		err := ToTyped(&out, wrongTypeInput)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected a string")
	})

	t.Run("time_invalid_format", func(t *testing.T) {
		var out sdktime.Time
		err := ToTyped(&out, dyn.V("not-a-time"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid google.protobuf.Timestamp value")
	})

	t.Run("fieldmask_wrong_type", func(t *testing.T) {
		var out sdkfieldmask.FieldMask
		err := ToTyped(&out, wrongTypeInput)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected a string")
	})
}

func TestSpecialCases(t *testing.T) {
	t.Run("duration_zero", func(t *testing.T) {
		var src sdkduration.Duration
		nv, err := FromTyped(src, dyn.NilValue)
		require.NoError(t, err)
		assert.Equal(t, dyn.NilValue, nv)
	})

	t.Run("fieldmask_empty_fromtyped", func(t *testing.T) {
		src := sdkfieldmask.New([]string{})
		nv, err := FromTyped(src, dyn.NilValue)
		require.NoError(t, err)
		// Empty field mask marshals to empty string
		assert.Equal(t, dyn.V(""), nv)
	})

	t.Run("fieldmask_empty_totyped", func(t *testing.T) {
		var out sdkfieldmask.FieldMask
		err := ToTyped(&out, dyn.V(""))
		require.NoError(t, err)
		assert.Empty(t, out.Paths)
	})
}

// End-to-end tests with structs containing SDK native types

func TestSDKTypesRoundTripWithPostgresBranchSpec(t *testing.T) {
	type BranchSpec struct {
		ExpireTime       *sdktime.Time         `json:"expire_time,omitempty"`
		SourceBranchTime *sdktime.Time         `json:"source_branch_time,omitempty"`
		Ttl              *sdkduration.Duration `json:"ttl,omitempty"`
		IsProtected      bool                  `json:"is_protected,omitempty"`
	}

	original := BranchSpec{
		ExpireTime:       sdktime.New(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)),
		SourceBranchTime: sdktime.New(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		Ttl:              sdkduration.New(7 * 24 * time.Hour),
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
		Name       string                 `json:"name"`
		UpdateMask sdkfieldmask.FieldMask `json:"update_mask"`
	}

	original := UpdateRequest{
		Name:       "projects/123/branches/456",
		UpdateMask: *sdkfieldmask.New([]string{"spec.ttl", "spec.is_protected"}),
	}

	dynValue, err := FromTyped(original, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, "projects/123/branches/456", dynValue.Get("name").MustString())
	assert.Equal(t, "spec.ttl,spec.is_protected", dynValue.Get("update_mask").MustString())

	var roundtrip UpdateRequest
	err = ToTyped(&roundtrip, dynValue)
	require.NoError(t, err)
	assert.Equal(t, original.Name, roundtrip.Name)
	assert.Equal(t, []string{"spec.ttl", "spec.is_protected"}, roundtrip.UpdateMask.Paths)
}

func TestSDKTypesNormalizeWithStruct(t *testing.T) {
	type BranchSpec struct {
		ExpireTime  *sdktime.Time         `json:"expire_time,omitempty"`
		Ttl         *sdkduration.Duration `json:"ttl,omitempty"`
		IsProtected bool                  `json:"is_protected,omitempty"`
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
