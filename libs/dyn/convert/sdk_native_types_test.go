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

// Duration tests

func TestFromTypedDuration(t *testing.T) {
	src := sdkduration.New(300 * time.Second)
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("300s"), nv)
}

func TestFromTypedDurationPointer(t *testing.T) {
	src := sdkduration.New(5 * time.Minute)
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("300s"), nv)
}

func TestFromTypedDurationInStruct(t *testing.T) {
	type Tmp struct {
		Timeout *sdkduration.Duration `json:"timeout"`
		Ttl     *sdkduration.Duration `json:"ttl"`
	}

	src := Tmp{
		Timeout: sdkduration.New(300 * time.Second),
		Ttl:     sdkduration.New(7 * 24 * time.Hour),
	}

	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, "300s", nv.Get("timeout").MustString())
	assert.Equal(t, "604800s", nv.Get("ttl").MustString())
}

func TestFromTypedDurationNil(t *testing.T) {
	var src *sdkduration.Duration
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedDurationZero(t *testing.T) {
	var src sdkduration.Duration
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	// Zero value duration should return nil when not forced
	assert.Equal(t, dyn.NilValue, nv)
}

func TestNormalizeDuration(t *testing.T) {
	var typ sdkduration.Duration
	vin := dyn.NewValue("300s", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("300s", vin.Locations()), vout)
}

func TestNormalizeDurationPointer(t *testing.T) {
	var typ *sdkduration.Duration
	vin := dyn.NewValue("604800s", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("604800s", vin.Locations()), vout)
}

func TestNormalizeDurationInStruct(t *testing.T) {
	type Tmp struct {
		Timeout *sdkduration.Duration `json:"timeout"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"timeout": dyn.NewValue("300s", []dyn.Location{{File: "file", Line: 1, Column: 1}}),
	})

	vout, diags := Normalize(typ, vin)
	assert.Empty(t, diags)
	assert.Equal(t, "300s", vout.Get("timeout").MustString())
}

func TestNormalizeDurationNil(t *testing.T) {
	var typ *sdkduration.Duration
	vin := dyn.NilValue
	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   `expected a string value, found null`,
		Locations: []dyn.Location{vin.Location()},
		Paths:     []dyn.Path{dyn.EmptyPath},
	}, err[0])
	assert.Equal(t, dyn.InvalidValue, vout)
}

func TestToTypedDuration(t *testing.T) {
	var out sdkduration.Duration
	v := dyn.V("300s")

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, 300*time.Second, out.AsDuration())
}

func TestToTypedDurationPointer(t *testing.T) {
	var out *sdkduration.Duration
	v := dyn.V("300s") // 5 minutes in protobuf duration format

	err := ToTyped(&out, v)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 5*time.Minute, out.AsDuration())
}

func TestToTypedDurationInStruct(t *testing.T) {
	type Tmp struct {
		Timeout *sdkduration.Duration `json:"timeout"`
		Ttl     *sdkduration.Duration `json:"ttl"`
	}

	var out Tmp
	v := dyn.V(map[string]dyn.Value{
		"timeout": dyn.V("300s"),
		"ttl":     dyn.V("604800s"), // 7 days in seconds
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	require.NotNil(t, out.Timeout)
	require.NotNil(t, out.Ttl)
	assert.Equal(t, 300*time.Second, out.Timeout.AsDuration())
	assert.Equal(t, 7*24*time.Hour, out.Ttl.AsDuration())
}

func TestToTypedDurationNil(t *testing.T) {
	var out *sdkduration.Duration
	v := dyn.NilValue

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToTypedDurationError(t *testing.T) {
	var out sdkduration.Duration
	v := dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")})

	err := ToTyped(&out, v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected a string")
}

func TestToTypedDurationInvalidFormat(t *testing.T) {
	var out sdkduration.Duration
	// The SDK's duration.Duration uses protobuf format which only supports
	// seconds notation (e.g., "604800s"), not human-readable formats like "7d".
	v := dyn.V("7d")

	err := ToTyped(&out, v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid google.protobuf.Duration value")
}

// Time tests

func TestFromTypedSDKTime(t *testing.T) {
	src := sdktime.New(time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC))
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("2023-12-25T10:30:00Z"), nv)
}

func TestFromTypedSDKTimePointer(t *testing.T) {
	src := sdktime.New(time.Date(2023, 12, 25, 10, 30, 0, 123456789, time.UTC))
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("2023-12-25T10:30:00.123456789Z"), nv)
}

func TestFromTypedSDKTimeInStruct(t *testing.T) {
	type Tmp struct {
		CreatedAt *sdktime.Time `json:"created_at"`
		UpdatedAt *sdktime.Time `json:"updated_at"`
	}

	src := Tmp{
		CreatedAt: sdktime.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: sdktime.New(time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)),
	}

	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, "2023-01-01T00:00:00Z", nv.Get("created_at").MustString())
	assert.Equal(t, "2023-12-31T23:59:59Z", nv.Get("updated_at").MustString())
}

func TestFromTypedSDKTimeNil(t *testing.T) {
	var src *sdktime.Time
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestNormalizeSDKTime(t *testing.T) {
	var typ sdktime.Time
	vin := dyn.NewValue("2023-12-25T10:30:00Z", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("2023-12-25T10:30:00Z", vin.Locations()), vout)
}

func TestNormalizeSDKTimePointer(t *testing.T) {
	var typ *sdktime.Time
	vin := dyn.NewValue("2023-12-25T10:30:00.123456789Z", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("2023-12-25T10:30:00.123456789Z", vin.Locations()), vout)
}

func TestNormalizeSDKTimeInStruct(t *testing.T) {
	type Tmp struct {
		CreatedAt *sdktime.Time `json:"created_at"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"created_at": dyn.NewValue("2023-01-01T00:00:00Z", []dyn.Location{{File: "file", Line: 1, Column: 1}}),
	})

	vout, diags := Normalize(typ, vin)
	assert.Empty(t, diags)
	assert.Equal(t, "2023-01-01T00:00:00Z", vout.Get("created_at").MustString())
}

func TestNormalizeSDKTimeNil(t *testing.T) {
	var typ *sdktime.Time
	vin := dyn.NilValue
	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   `expected a string value, found null`,
		Locations: []dyn.Location{vin.Location()},
		Paths:     []dyn.Path{dyn.EmptyPath},
	}, err[0])
	assert.Equal(t, dyn.InvalidValue, vout)
}

func TestToTypedSDKTime(t *testing.T) {
	var out sdktime.Time
	v := dyn.V("2023-12-25T10:30:00Z")

	err := ToTyped(&out, v)
	require.NoError(t, err)
	expected := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
	assert.Equal(t, expected, out.AsTime())
}

func TestToTypedSDKTimePointer(t *testing.T) {
	var out *sdktime.Time
	v := dyn.V("2023-12-25T10:30:00.123456789Z")

	err := ToTyped(&out, v)
	require.NoError(t, err)
	require.NotNil(t, out)
	expected := time.Date(2023, 12, 25, 10, 30, 0, 123456789, time.UTC)
	assert.Equal(t, expected, out.AsTime())
}

func TestToTypedSDKTimeInStruct(t *testing.T) {
	type Tmp struct {
		CreatedAt *sdktime.Time `json:"created_at"`
		UpdatedAt *sdktime.Time `json:"updated_at"`
	}

	var out Tmp
	v := dyn.V(map[string]dyn.Value{
		"created_at": dyn.V("2023-01-01T00:00:00Z"),
		"updated_at": dyn.V("2023-12-31T23:59:59Z"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	require.NotNil(t, out.CreatedAt)
	require.NotNil(t, out.UpdatedAt)
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), out.CreatedAt.AsTime())
	assert.Equal(t, time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC), out.UpdatedAt.AsTime())
}

func TestToTypedSDKTimeNil(t *testing.T) {
	var out *sdktime.Time
	v := dyn.NilValue

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToTypedSDKTimeError(t *testing.T) {
	var out sdktime.Time
	v := dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")})

	err := ToTyped(&out, v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected a string")
}

// FieldMask tests

func TestFromTypedFieldMask(t *testing.T) {
	src := sdkfieldmask.New([]string{"name", "age", "email"})
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("name,age,email"), nv)
}

func TestFromTypedFieldMaskPointer(t *testing.T) {
	src := sdkfieldmask.New([]string{"first_name", "last_name"})
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("first_name,last_name"), nv)
}

func TestFromTypedFieldMaskInStruct(t *testing.T) {
	type Tmp struct {
		Fields *sdkfieldmask.FieldMask `json:"fields"`
	}

	src := Tmp{
		Fields: sdkfieldmask.New([]string{"id", "name", "status"}),
	}

	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, "id,name,status", nv.Get("fields").MustString())
}

func TestFromTypedFieldMaskNil(t *testing.T) {
	var src *sdkfieldmask.FieldMask
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedFieldMaskEmpty(t *testing.T) {
	src := sdkfieldmask.New([]string{})
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	// Empty field mask marshals to empty string
	assert.Equal(t, dyn.V(""), nv)
}

func TestNormalizeFieldMask(t *testing.T) {
	var typ sdkfieldmask.FieldMask
	vin := dyn.NewValue("name,age,email", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("name,age,email", vin.Locations()), vout)
}

func TestNormalizeFieldMaskPointer(t *testing.T) {
	var typ *sdkfieldmask.FieldMask
	vin := dyn.NewValue("first_name,last_name", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("first_name,last_name", vin.Locations()), vout)
}

func TestNormalizeFieldMaskInStruct(t *testing.T) {
	type Tmp struct {
		Fields *sdkfieldmask.FieldMask `json:"fields"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"fields": dyn.NewValue("id,name,status", []dyn.Location{{File: "file", Line: 1, Column: 1}}),
	})

	vout, diags := Normalize(typ, vin)
	assert.Empty(t, diags)
	assert.Equal(t, "id,name,status", vout.Get("fields").MustString())
}

func TestNormalizeFieldMaskNil(t *testing.T) {
	var typ *sdkfieldmask.FieldMask
	vin := dyn.NilValue
	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   `expected a string value, found null`,
		Locations: []dyn.Location{vin.Location()},
		Paths:     []dyn.Path{dyn.EmptyPath},
	}, err[0])
	assert.Equal(t, dyn.InvalidValue, vout)
}

func TestToTypedFieldMask(t *testing.T) {
	var out sdkfieldmask.FieldMask
	v := dyn.V("name,age,email")

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, []string{"name", "age", "email"}, out.Paths)
}

func TestToTypedFieldMaskPointer(t *testing.T) {
	var out *sdkfieldmask.FieldMask
	v := dyn.V("first_name,last_name")

	err := ToTyped(&out, v)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, []string{"first_name", "last_name"}, out.Paths)
}

func TestToTypedFieldMaskInStruct(t *testing.T) {
	type Tmp struct {
		Fields *sdkfieldmask.FieldMask `json:"fields"`
	}

	var out Tmp
	v := dyn.V(map[string]dyn.Value{
		"fields": dyn.V("id,name,status"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	require.NotNil(t, out.Fields)
	assert.Equal(t, []string{"id", "name", "status"}, out.Fields.Paths)
}

func TestToTypedFieldMaskNil(t *testing.T) {
	var out *sdkfieldmask.FieldMask
	v := dyn.NilValue

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToTypedFieldMaskError(t *testing.T) {
	var out sdkfieldmask.FieldMask
	v := dyn.V(map[string]dyn.Value{"foo": dyn.V("bar")})

	err := ToTyped(&out, v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected a string")
}

func TestToTypedFieldMaskEmpty(t *testing.T) {
	var out sdkfieldmask.FieldMask
	v := dyn.V("")

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, []string{}, out.Paths)
}

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
