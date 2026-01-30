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

func TestSDKNativeTypesNilValues(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "duration_fromtyped",
			testFunc: func(t *testing.T) {
				var src *sdkduration.Duration
				nv, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, dyn.NilValue, nv)
			},
		},
		{
			name: "duration_normalize",
			testFunc: func(t *testing.T) {
				var typ *sdkduration.Duration
				vin := dyn.NilValue
				vout, diags := Normalize(typ, vin)
				assert.Len(t, diags, 1)
				assert.Equal(t, diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   `expected a string value, found null`,
					Locations: []dyn.Location{vin.Location()},
					Paths:     []dyn.Path{dyn.EmptyPath},
				}, diags[0])
				assert.Equal(t, dyn.InvalidValue, vout)
			},
		},
		{
			name: "time_fromtyped",
			testFunc: func(t *testing.T) {
				var src *sdktime.Time
				nv, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, dyn.NilValue, nv)
			},
		},
		{
			name: "time_normalize",
			testFunc: func(t *testing.T) {
				var typ *sdktime.Time
				vin := dyn.NilValue
				vout, diags := Normalize(typ, vin)
				assert.Len(t, diags, 1)
				assert.Equal(t, diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   `expected a string value, found null`,
					Locations: []dyn.Location{vin.Location()},
					Paths:     []dyn.Path{dyn.EmptyPath},
				}, diags[0])
				assert.Equal(t, dyn.InvalidValue, vout)
			},
		},
		{
			name: "fieldmask_fromtyped",
			testFunc: func(t *testing.T) {
				var src *sdkfieldmask.FieldMask
				nv, err := FromTyped(src, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, dyn.NilValue, nv)
			},
		},
		{
			name: "fieldmask_normalize",
			testFunc: func(t *testing.T) {
				var typ *sdkfieldmask.FieldMask
				vin := dyn.NilValue
				vout, diags := Normalize(typ, vin)
				assert.Len(t, diags, 1)
				assert.Equal(t, diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   `expected a string value, found null`,
					Locations: []dyn.Location{vin.Location()},
					Paths:     []dyn.Path{dyn.EmptyPath},
				}, diags[0])
				assert.Equal(t, dyn.InvalidValue, vout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

func TestSDKNativeTypesErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         dyn.Value
		expectedError string
		testFunc      func(t *testing.T, input dyn.Value, expectedError string)
	}{
		{
			name:          "duration_wrong_type",
			input:         dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")}),
			expectedError: "expected a string",
			testFunc: func(t *testing.T, input dyn.Value, expectedError string) {
				var out sdkduration.Duration
				err := ToTyped(&out, input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), expectedError)
			},
		},
		{
			name:          "duration_invalid_format",
			input:         dyn.V("7d"),
			expectedError: "invalid google.protobuf.Duration value",
			testFunc: func(t *testing.T, input dyn.Value, expectedError string) {
				var out sdkduration.Duration
				err := ToTyped(&out, input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), expectedError)
			},
		},
		{
			name:          "time_wrong_type",
			input:         dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")}),
			expectedError: "expected a string",
			testFunc: func(t *testing.T, input dyn.Value, expectedError string) {
				var out sdktime.Time
				err := ToTyped(&out, input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), expectedError)
			},
		},
		{
			name:          "fieldmask_wrong_type",
			input:         dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")}),
			expectedError: "expected a string",
			testFunc: func(t *testing.T, input dyn.Value, expectedError string) {
				var out sdkfieldmask.FieldMask
				err := ToTyped(&out, input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), expectedError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, tt.input, tt.expectedError)
		})
	}
}

func TestSDKNativeTypesSpecialCases(t *testing.T) {
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
		v := dyn.V("")

		err := ToTyped(&out, v)
		require.NoError(t, err)
		assert.Empty(t, out.Paths)
	})
}


// Duration tests

// Time tests

// FieldMask tests

// End-to-end tests using actual SDK types

func TestSDKTypesRoundTripWithPostgresBranchSpec(t *testing.T) {
	// Import the postgres package types to test real SDK usage
	// postgres.BranchSpec uses time.Time and duration.Duration
	type BranchSpec struct {
		ExpireTime       *sdktime.Time         `json:"expire_time,omitempty"`
		SourceBranchTime *sdktime.Time         `json:"source_branch_time,omitempty"`
		Ttl              *sdkduration.Duration `json:"ttl,omitempty"`
		IsProtected      bool                  `json:"is_protected,omitempty"`
	}

	// Create a BranchSpec with SDK native types
	original := BranchSpec{
		ExpireTime:       sdktime.New(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)),
		SourceBranchTime: sdktime.New(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		Ttl:              sdkduration.New(7 * 24 * time.Hour),
		IsProtected:      true,
	}

	// Convert to dyn.Value
	dynValue, err := FromTyped(original, dyn.NilValue)
	require.NoError(t, err)

	// Verify the dyn.Value has string representations
	assert.Equal(t, "2024-12-31T23:59:59Z", dynValue.Get("expire_time").MustString())
	assert.Equal(t, "2024-01-01T00:00:00Z", dynValue.Get("source_branch_time").MustString())
	assert.Equal(t, "604800s", dynValue.Get("ttl").MustString())
	assert.True(t, dynValue.Get("is_protected").MustBool())

	// Convert back to typed
	var roundtrip BranchSpec
	err = ToTyped(&roundtrip, dynValue)
	require.NoError(t, err)

	// Verify round-trip preserves values
	require.NotNil(t, roundtrip.ExpireTime)
	require.NotNil(t, roundtrip.SourceBranchTime)
	require.NotNil(t, roundtrip.Ttl)
	assert.Equal(t, original.ExpireTime.AsTime(), roundtrip.ExpireTime.AsTime())
	assert.Equal(t, original.SourceBranchTime.AsTime(), roundtrip.SourceBranchTime.AsTime())
	assert.Equal(t, original.Ttl.AsDuration(), roundtrip.Ttl.AsDuration())
	assert.Equal(t, original.IsProtected, roundtrip.IsProtected)
}

func TestSDKTypesRoundTripWithUpdateRequest(t *testing.T) {
	// Test with a struct similar to postgres.UpdateBranchRequest
	// which uses fieldmask.FieldMask
	type UpdateRequest struct {
		Name       string                 `json:"name"`
		UpdateMask sdkfieldmask.FieldMask `json:"update_mask"`
	}

	// Create an update request with FieldMask
	original := UpdateRequest{
		Name:       "projects/123/branches/456",
		UpdateMask: *sdkfieldmask.New([]string{"spec.ttl", "spec.is_protected"}),
	}

	// Convert to dyn.Value
	dynValue, err := FromTyped(original, dyn.NilValue)
	require.NoError(t, err)

	// Verify the dyn.Value has string representation for field mask
	assert.Equal(t, "projects/123/branches/456", dynValue.Get("name").MustString())
	assert.Equal(t, "spec.ttl,spec.is_protected", dynValue.Get("update_mask").MustString())

	// Convert back to typed
	var roundtrip UpdateRequest
	err = ToTyped(&roundtrip, dynValue)
	require.NoError(t, err)

	// Verify round-trip preserves values
	assert.Equal(t, original.Name, roundtrip.Name)
	assert.Equal(t, []string{"spec.ttl", "spec.is_protected"}, roundtrip.UpdateMask.Paths)
}

func TestSDKTypesNormalizeWithPostgresBranchSpec(t *testing.T) {
	// Test normalization with postgres.BranchSpec-like structure
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

	// Verify normalized values preserve string representations
	assert.Equal(t, "2024-12-31T23:59:59Z", vout.Get("expire_time").MustString())
	assert.Equal(t, "604800s", vout.Get("ttl").MustString())
	assert.True(t, vout.Get("is_protected").MustBool())

	// Convert to typed to verify it works
	var out BranchSpec
	err := ToTyped(&out, vout)
	require.NoError(t, err)
	require.NotNil(t, out.ExpireTime)
	require.NotNil(t, out.Ttl)
	assert.Equal(t, time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), out.ExpireTime.AsTime())
	assert.Equal(t, 7*24*time.Hour, out.Ttl.AsDuration())
	assert.True(t, out.IsProtected)
}
