package convert

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeStruct(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeStructElementDiagnostic(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V(map[string]dyn.Value{"an": dyn.V("error")}),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
	}, err[0])

	// Elements that encounter an error during normalization are dropped.
	assert.Equal(t, map[string]any{
		"foo": "bar",
	}, vout.AsAny())
}

func TestNormalizeStructUnknownField(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `unknown field: bar`,
		Location: vin.Get("foo").Location(),
	}, err[0])

	// The field that can be mapped to the struct field is retained.
	assert.Equal(t, map[string]any{
		"foo": "bar",
	}, vout.AsAny())
}

func TestNormalizeStructNil(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	var typ Tmp
	vin := dyn.NilValue
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeStructError(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	var typ Tmp
	vin := dyn.V("string")
	_, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected map, found string`,
		Location: vin.Get("foo").Location(),
	}, err[0])
}

func TestNormalizeMap(t *testing.T) {
	var typ map[string]string
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeMapElementDiagnostic(t *testing.T) {
	var typ map[string]string
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V(map[string]dyn.Value{"an": dyn.V("error")}),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
	}, err[0])

	// Elements that encounter an error during normalization are dropped.
	assert.Equal(t, map[string]any{
		"foo": "bar",
	}, vout.AsAny())
}

func TestNormalizeMapNil(t *testing.T) {
	var typ map[string]string
	vin := dyn.NilValue
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeMapError(t *testing.T) {
	var typ map[string]string
	vin := dyn.V("string")
	_, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected map, found string`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeSlice(t *testing.T) {
	var typ []string
	vin := dyn.V([]dyn.Value{
		dyn.V("foo"),
		dyn.V("bar"),
	})

	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeSliceElementDiagnostic(t *testing.T) {
	var typ []string
	vin := dyn.V([]dyn.Value{
		dyn.V("foo"),
		dyn.V("bar"),
		dyn.V(map[string]dyn.Value{"an": dyn.V("error")}),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
	}, err[0])

	// Elements that encounter an error during normalization are dropped.
	assert.Equal(t, []any{"foo", "bar"}, vout.AsAny())
}

func TestNormalizeSliceNil(t *testing.T) {
	var typ []string
	vin := dyn.NilValue
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeSliceError(t *testing.T) {
	var typ []string
	vin := dyn.V("string")
	_, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected sequence, found string`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeString(t *testing.T) {
	var typ string
	vin := dyn.V("string")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeStringNil(t *testing.T) {
	var typ string
	vin := dyn.NewValue(nil, dyn.Location{File: "file", Line: 1, Column: 1})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found nil`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeStringFromBool(t *testing.T) {
	var typ string
	vin := dyn.NewValue(true, dyn.Location{File: "file", Line: 1, Column: 1})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("true", vin.Location()), vout)
}

func TestNormalizeStringFromInt(t *testing.T) {
	var typ string
	vin := dyn.NewValue(123, dyn.Location{File: "file", Line: 1, Column: 1})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("123", vin.Location()), vout)
}

func TestNormalizeStringFromFloat(t *testing.T) {
	var typ string
	vin := dyn.NewValue(1.20, dyn.Location{File: "file", Line: 1, Column: 1})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.NewValue("1.2", vin.Location()), vout)
}

func TestNormalizeStringError(t *testing.T) {
	var typ string
	vin := dyn.V(map[string]dyn.Value{"an": dyn.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
	}, err[0])
}

func TestNormalizeBool(t *testing.T) {
	var typ bool
	vin := dyn.V(true)
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(true), vout)
}

func TestNormalizeBoolNil(t *testing.T) {
	var typ bool
	vin := dyn.NewValue(nil, dyn.Location{File: "file", Line: 1, Column: 1})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected bool, found nil`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeBoolFromString(t *testing.T) {
	var typ bool

	for _, c := range []struct {
		Input  string
		Output bool
	}{
		{"true", true},
		{"false", false},
		{"Y", true},
		{"N", false},
		{"on", true},
		{"off", false},
	} {
		vin := dyn.V(c.Input)
		vout, err := Normalize(&typ, vin)
		assert.Empty(t, err)
		assert.Equal(t, dyn.V(c.Output), vout)
	}
}

func TestNormalizeBoolFromStringError(t *testing.T) {
	var typ bool
	vin := dyn.V("abc")
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected bool, found string`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeBoolError(t *testing.T) {
	var typ bool
	vin := dyn.V(map[string]dyn.Value{"an": dyn.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected bool, found map`,
		Location: dyn.Location{},
	}, err[0])
}

func TestNormalizeInt(t *testing.T) {
	var typ int
	vin := dyn.V(123)
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(int64(123)), vout)
}

func TestNormalizeIntNil(t *testing.T) {
	var typ int
	vin := dyn.NewValue(nil, dyn.Location{File: "file", Line: 1, Column: 1})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected int, found nil`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeIntFromString(t *testing.T) {
	var typ int
	vin := dyn.V("123")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(int64(123)), vout)
}

func TestNormalizeIntFromStringError(t *testing.T) {
	var typ int
	vin := dyn.V("abc")
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `cannot parse "abc" as an integer`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeIntError(t *testing.T) {
	var typ int
	vin := dyn.V(map[string]dyn.Value{"an": dyn.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected int, found map`,
		Location: dyn.Location{},
	}, err[0])
}

func TestNormalizeFloat(t *testing.T) {
	var typ float64
	vin := dyn.V(1.2)
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(1.2), vout)
}

func TestNormalizeFloatNil(t *testing.T) {
	var typ float64
	vin := dyn.NewValue(nil, dyn.Location{File: "file", Line: 1, Column: 1})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected float, found nil`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeFloatFromString(t *testing.T) {
	var typ float64
	vin := dyn.V("1.2")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(1.2), vout)
}

func TestNormalizeFloatFromStringError(t *testing.T) {
	var typ float64
	vin := dyn.V("abc")
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `cannot parse "abc" as a floating point number`,
		Location: vin.Location(),
	}, err[0])
}

func TestNormalizeFloatError(t *testing.T) {
	var typ float64
	vin := dyn.V(map[string]dyn.Value{"an": dyn.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected float, found map`,
		Location: dyn.Location{},
	}, err[0])
}
