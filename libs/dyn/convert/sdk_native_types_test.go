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

// Equality check functions for SDK native types

func assertDurationEqual(t *testing.T, expected, actual any) {
	e := expected.(sdkduration.Duration)
	a := actual.(sdkduration.Duration)
	assert.Equal(t, e.AsDuration(), a.AsDuration())
}

func assertTimeEqual(t *testing.T, expected, actual any) {
	e := expected.(sdktime.Time)
	a := actual.(sdktime.Time)
	assert.Equal(t, e.AsTime(), a.AsTime())
}

func assertFieldMaskEqual(t *testing.T, expected, actual any) {
	e := expected.(sdkfieldmask.FieldMask)
	a := actual.(sdkfieldmask.FieldMask)
	assert.Equal(t, e.Paths, a.Paths)
}

// Roundtrip test - verifies SDK native types convert to dyn.Value and back for both value and pointer types

func TestSDKNativeTypesRoundtrip(t *testing.T) {
	tests := []struct {
		name           string
		typed          any
		expectedString string
		assertEqual    func(t *testing.T, expected, actual any)
	}{
		{
			name:           "duration_5min",
			typed:          *sdkduration.New(5 * time.Minute),
			expectedString: "300s",
			assertEqual:    assertDurationEqual,
		},
		{
			name:           "duration_7days",
			typed:          *sdkduration.New(7 * 24 * time.Hour),
			expectedString: "604800s",
			assertEqual:    assertDurationEqual,
		},
		{
			name:           "duration_1hour",
			typed:          *sdkduration.New(1 * time.Hour),
			expectedString: "3600s",
			assertEqual:    assertDurationEqual,
		},
		{
			name:           "time_no_nanos",
			typed:          *sdktime.New(time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)),
			expectedString: "2023-12-25T10:30:00Z",
			assertEqual:    assertTimeEqual,
		},
		{
			name:           "time_with_nanos",
			typed:          *sdktime.New(time.Date(2023, 12, 25, 10, 30, 0, 123456789, time.UTC)),
			expectedString: "2023-12-25T10:30:00.123456789Z",
			assertEqual:    assertTimeEqual,
		},
		{
			name:           "time_epoch",
			typed:          *sdktime.New(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			expectedString: "1970-01-01T00:00:00Z",
			assertEqual:    assertTimeEqual,
		},
		{
			name:           "fieldmask_single",
			typed:          *sdkfieldmask.New([]string{"name"}),
			expectedString: "name",
			assertEqual:    assertFieldMaskEqual,
		},
		{
			name:           "fieldmask_multiple",
			typed:          *sdkfieldmask.New([]string{"name", "age", "email"}),
			expectedString: "name,age,email",
			assertEqual:    assertFieldMaskEqual,
		},
		{
			name:           "fieldmask_nested",
			typed:          *sdkfieldmask.New([]string{"user.name", "user.email"}),
			expectedString: "user.name,user.email",
			assertEqual:    assertFieldMaskEqual,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test value type
			t.Run("value", func(t *testing.T) {
				dynValue, err := FromTyped(tt.typed, dyn.NilValue)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				// Convert back and check equality based on type
				switch tt.typed.(type) {
				case sdkduration.Duration:
					var out sdkduration.Duration
					err = ToTyped(&out, dynValue)
					require.NoError(t, err)
					tt.assertEqual(t, tt.typed, out)
				case sdktime.Time:
					var out sdktime.Time
					err = ToTyped(&out, dynValue)
					require.NoError(t, err)
					tt.assertEqual(t, tt.typed, out)
				case sdkfieldmask.FieldMask:
					var out sdkfieldmask.FieldMask
					err = ToTyped(&out, dynValue)
					require.NoError(t, err)
					tt.assertEqual(t, tt.typed, out)
				}
			})

			// Test pointer type
			t.Run("pointer", func(t *testing.T) {
				// Create a typed pointer for FromTyped based on the actual type
				var dynValue dyn.Value
				var err error
				switch v := tt.typed.(type) {
				case sdkduration.Duration:
					src := v
					dynValue, err = FromTyped(&src, dyn.NilValue)
					require.NoError(t, err)
				case sdktime.Time:
					src := v
					dynValue, err = FromTyped(&src, dyn.NilValue)
					require.NoError(t, err)
				case sdkfieldmask.FieldMask:
					src := v
					dynValue, err = FromTyped(&src, dyn.NilValue)
					require.NoError(t, err)
				default:
					t.Fatalf("unsupported type: %T", tt.typed)
				}
				assert.Equal(t, tt.expectedString, dynValue.MustString())

				// Convert back and check equality based on type
				switch tt.typed.(type) {
				case sdkduration.Duration:
					var out *sdkduration.Duration
					err = ToTyped(&out, dynValue)
					require.NoError(t, err)
					require.NotNil(t, out)
					tt.assertEqual(t, tt.typed, *out)
				case sdktime.Time:
					var out *sdktime.Time
					err = ToTyped(&out, dynValue)
					require.NoError(t, err)
					require.NotNil(t, out)
					tt.assertEqual(t, tt.typed, *out)
				case sdkfieldmask.FieldMask:
					var out *sdkfieldmask.FieldMask
					err = ToTyped(&out, dynValue)
					require.NoError(t, err)
					require.NotNil(t, out)
					tt.assertEqual(t, tt.typed, *out)
				}
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
