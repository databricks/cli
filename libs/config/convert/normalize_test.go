package convert

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeStruct(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	var typ Tmp
	vin := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V("baz"),
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
	vin := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V(map[string]config.Value{"an": config.V("error")}),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: config.Location{},
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
	vin := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V("baz"),
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
	vin := config.NilValue
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeStructError(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	var typ Tmp
	vin := config.V("string")
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
	vin := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V("baz"),
	})

	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeMapElementDiagnostic(t *testing.T) {
	var typ map[string]string
	vin := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V(map[string]config.Value{"an": config.V("error")}),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: config.Location{},
	}, err[0])

	// Elements that encounter an error during normalization are dropped.
	assert.Equal(t, map[string]any{
		"foo": "bar",
	}, vout.AsAny())
}

func TestNormalizeMapNil(t *testing.T) {
	var typ map[string]string
	vin := config.NilValue
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeMapError(t *testing.T) {
	var typ map[string]string
	vin := config.V("string")
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
	vin := config.V([]config.Value{
		config.V("foo"),
		config.V("bar"),
	})

	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeSliceElementDiagnostic(t *testing.T) {
	var typ []string
	vin := config.V([]config.Value{
		config.V("foo"),
		config.V("bar"),
		config.V(map[string]config.Value{"an": config.V("error")}),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: config.Location{},
	}, err[0])

	// Elements that encounter an error during normalization are dropped.
	assert.Equal(t, []any{"foo", "bar"}, vout.AsAny())
}

func TestNormalizeSliceNil(t *testing.T) {
	var typ []string
	vin := config.NilValue
	vout, err := Normalize(typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeSliceError(t *testing.T) {
	var typ []string
	vin := config.V("string")
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
	vin := config.V("string")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeStringNil(t *testing.T) {
	var typ string
	vin := config.NewValue(nil, config.Location{File: "file", Line: 1, Column: 1})
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
	vin := config.NewValue(true, config.Location{File: "file", Line: 1, Column: 1})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.NewValue("true", vin.Location()), vout)
}

func TestNormalizeStringFromInt(t *testing.T) {
	var typ string
	vin := config.NewValue(123, config.Location{File: "file", Line: 1, Column: 1})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.NewValue("123", vin.Location()), vout)
}

func TestNormalizeStringFromFloat(t *testing.T) {
	var typ string
	vin := config.NewValue(1.20, config.Location{File: "file", Line: 1, Column: 1})
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.NewValue("1.2", vin.Location()), vout)
}

func TestNormalizeStringError(t *testing.T) {
	var typ string
	vin := config.V(map[string]config.Value{"an": config.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected string, found map`,
		Location: config.Location{},
	}, err[0])
}

func TestNormalizeBool(t *testing.T) {
	var typ bool
	vin := config.V(true)
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.V(true), vout)
}

func TestNormalizeBoolNil(t *testing.T) {
	var typ bool
	vin := config.NewValue(nil, config.Location{File: "file", Line: 1, Column: 1})
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
		vin := config.V(c.Input)
		vout, err := Normalize(&typ, vin)
		assert.Empty(t, err)
		assert.Equal(t, config.V(c.Output), vout)
	}
}

func TestNormalizeBoolFromStringError(t *testing.T) {
	var typ bool
	vin := config.V("abc")
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
	vin := config.V(map[string]config.Value{"an": config.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected bool, found map`,
		Location: config.Location{},
	}, err[0])
}

func TestNormalizeInt(t *testing.T) {
	var typ int
	vin := config.V(123)
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.V(int64(123)), vout)
}

func TestNormalizeIntNil(t *testing.T) {
	var typ int
	vin := config.NewValue(nil, config.Location{File: "file", Line: 1, Column: 1})
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
	vin := config.V("123")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.V(int64(123)), vout)
}

func TestNormalizeIntFromStringError(t *testing.T) {
	var typ int
	vin := config.V("abc")
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
	vin := config.V(map[string]config.Value{"an": config.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected int, found map`,
		Location: config.Location{},
	}, err[0])
}

func TestNormalizeFloat(t *testing.T) {
	var typ float64
	vin := config.V(1.2)
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.V(1.2), vout)
}

func TestNormalizeFloatNil(t *testing.T) {
	var typ float64
	vin := config.NewValue(nil, config.Location{File: "file", Line: 1, Column: 1})
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
	vin := config.V("1.2")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, config.V(1.2), vout)
}

func TestNormalizeFloatFromStringError(t *testing.T) {
	var typ float64
	vin := config.V("abc")
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
	vin := config.V(map[string]config.Value{"an": config.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  `expected float, found map`,
		Location: config.Location{},
	}, err[0])
}
