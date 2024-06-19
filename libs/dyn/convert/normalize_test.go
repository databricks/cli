package convert

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
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
		Severity: diag.Warning,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
		Path:     dyn.NewPath(dyn.Key("bar")),
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
		Location: vin.GetTODO("foo").Location(),
		Path:     dyn.EmptyPath,
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
		Severity: diag.Warning,
		Summary:  `expected map, found string`,
		Location: vin.GetTODO("foo").Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeStructNestedError(t *testing.T) {
	type Nested struct {
		F1 int `json:"f1"`
		F2 int `json:"f2"`
	}
	type Tmp struct {
		Foo Nested `json:"foo"`
		Bar Nested `json:"bar"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(map[string]dyn.Value{
			"f1": dyn.V("error"),
			"f2": dyn.V(1),
		}),
		"bar": dyn.V(map[string]dyn.Value{
			"f1": dyn.V(1),
			"f2": dyn.V("error"),
		}),
	})
	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 2)

	// Verify that valid fields are retained.
	assert.Equal(t,
		dyn.V(map[string]dyn.Value{
			"foo": dyn.V(map[string]dyn.Value{
				"f2": dyn.V(int64(1)),
			}),
			"bar": dyn.V(map[string]dyn.Value{
				"f1": dyn.V(int64(1)),
			}),
		}),
		vout,
	)
}

func TestNormalizeStructIncludeMissingFields(t *testing.T) {
	type Nested struct {
		String string `json:"string"`
	}

	type Tmp struct {
		// Verify that fields that are already set in the dynamic value are not overridden.
		Existing string `json:"existing"`

		// Verify that structs are recursively normalized if not set.
		Nested Nested  `json:"nested"`
		Ptr    *Nested `json:"ptr"`

		// Verify that containers are also zero-initialized if not set.
		Map   map[string]string `json:"map"`
		Slice []string          `json:"slice"`

		// Verify that primitive types are zero-initialized if not set.
		String string  `json:"string"`
		Bool   bool    `json:"bool"`
		Int    int     `json:"int"`
		Float  float64 `json:"float"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"existing": dyn.V("already set"),
	})
	vout, err := Normalize(typ, vin, IncludeMissingFields)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"existing": dyn.V("already set"),
		"nested": dyn.V(map[string]dyn.Value{
			"string": dyn.V(""),
		}),
		"ptr": dyn.V(map[string]dyn.Value{
			"string": dyn.V(""),
		}),
		"map":    dyn.V(map[string]dyn.Value{}),
		"slice":  dyn.V([]dyn.Value{}),
		"string": dyn.V(""),
		"bool":   dyn.V(false),
		"int":    dyn.V(int64(0)),
		"float":  dyn.V(float64(0)),
	}), vout)
}

func TestNormalizeStructIncludeMissingFieldsOnRecursiveType(t *testing.T) {
	type Tmp struct {
		// Verify that structs are recursively normalized if not set.
		Ptr *Tmp `json:"ptr"`

		// Verify that primitive types are zero-initialized if not set.
		String string `json:"string"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"ptr": dyn.V(map[string]dyn.Value{
			"ptr": dyn.V(map[string]dyn.Value{
				"string": dyn.V("already set"),
			}),
		}),
	})
	vout, err := Normalize(typ, vin, IncludeMissingFields)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"ptr": dyn.V(map[string]dyn.Value{
			"ptr": dyn.V(map[string]dyn.Value{
				// Note: the ptr field is not zero-initialized because that would recurse.
				"string": dyn.V("already set"),
			}),
			"string": dyn.V(""),
		}),
		"string": dyn.V(""),
	}), vout)
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
		Severity: diag.Warning,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
		Path:     dyn.NewPath(dyn.Key("bar")),
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
		Severity: diag.Warning,
		Summary:  `expected map, found string`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeMapNestedError(t *testing.T) {
	type Nested struct {
		F1 int `json:"f1"`
		F2 int `json:"f2"`
	}

	var typ map[string]Nested
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(map[string]dyn.Value{
			"f1": dyn.V("error"),
			"f2": dyn.V(1),
		}),
		"bar": dyn.V(map[string]dyn.Value{
			"f1": dyn.V(1),
			"f2": dyn.V("error"),
		}),
	})
	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 2)

	// Verify that valid fields are retained.
	assert.Equal(t,
		dyn.V(map[string]dyn.Value{
			"foo": dyn.V(map[string]dyn.Value{
				"f2": dyn.V(int64(1)),
			}),
			"bar": dyn.V(map[string]dyn.Value{
				"f1": dyn.V(int64(1)),
			}),
		}),
		vout,
	)
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
		Severity: diag.Warning,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
		Path:     dyn.NewPath(dyn.Index(2)),
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
		Severity: diag.Warning,
		Summary:  `expected sequence, found string`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeSliceNestedError(t *testing.T) {
	type Nested struct {
		F1 int `json:"f1"`
		F2 int `json:"f2"`
	}

	var typ []Nested
	vin := dyn.V([]dyn.Value{
		dyn.V(map[string]dyn.Value{
			"f1": dyn.V("error"),
			"f2": dyn.V(1),
		}),
		dyn.V(map[string]dyn.Value{
			"f1": dyn.V(1),
			"f2": dyn.V("error"),
		}),
	})
	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 2)

	// Verify that valid fields are retained.
	assert.Equal(t,
		dyn.V([]dyn.Value{
			dyn.V(map[string]dyn.Value{
				"f2": dyn.V(int64(1)),
			}),
			dyn.V(map[string]dyn.Value{
				"f1": dyn.V(int64(1)),
			}),
		}),
		vout,
	)
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
		Severity: diag.Warning,
		Summary:  `expected a string value, found null`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
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
		Severity: diag.Warning,
		Summary:  `expected string, found map`,
		Location: dyn.Location{},
		Path:     dyn.EmptyPath,
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
		Severity: diag.Warning,
		Summary:  `expected a bool value, found null`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
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

func TestNormalizeBoolFromStringVariableReference(t *testing.T) {
	var typ bool
	vin := dyn.V("${var.foo}")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeBoolFromStringError(t *testing.T) {
	var typ bool
	vin := dyn.V("abc")
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `expected bool, found string`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeBoolError(t *testing.T) {
	var typ bool
	vin := dyn.V(map[string]dyn.Value{"an": dyn.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `expected bool, found map`,
		Location: dyn.Location{},
		Path:     dyn.EmptyPath,
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
		Severity: diag.Warning,
		Summary:  `expected a int value, found null`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeIntFromFloat(t *testing.T) {
	var typ int
	vin := dyn.V(float64(1.0))
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(int64(1)), vout)
}

func TestNormalizeIntFromFloatError(t *testing.T) {
	var typ int
	vin := dyn.V(1.5)
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `cannot accurately represent "1.5" as integer due to precision loss`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeIntFromString(t *testing.T) {
	var typ int
	vin := dyn.V("123")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(int64(123)), vout)
}

func TestNormalizeIntFromStringVariableReference(t *testing.T) {
	var typ int
	vin := dyn.V("${var.foo}")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeIntFromStringError(t *testing.T) {
	var typ int
	vin := dyn.V("abc")
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `cannot parse "abc" as an integer`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeIntError(t *testing.T) {
	var typ int
	vin := dyn.V(map[string]dyn.Value{"an": dyn.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `expected int, found map`,
		Location: dyn.Location{},
		Path:     dyn.EmptyPath,
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
		Severity: diag.Warning,
		Summary:  `expected a float value, found null`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeFloatFromInt(t *testing.T) {
	var typ float64

	// Maximum safe integer that can be accurately represented as a float.
	vin := dyn.V(int64(9007199254740992))
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(float64(9007199254740992)), vout)
}

func TestNormalizeFloatFromIntError(t *testing.T) {
	var typ float64

	// Minimum integer that cannot be accurately represented as a float.
	vin := dyn.V(9007199254740992 + 1)
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `cannot accurately represent "9007199254740993" as floating point number due to precision loss`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeFloatFromString(t *testing.T) {
	var typ float64
	vin := dyn.V("1.2")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, dyn.V(1.2), vout)
}

func TestNormalizeFloatFromStringVariableReference(t *testing.T) {
	var typ float64
	vin := dyn.V("${var.foo}")
	vout, err := Normalize(&typ, vin)
	assert.Empty(t, err)
	assert.Equal(t, vin, vout)
}

func TestNormalizeFloatFromStringError(t *testing.T) {
	var typ float64
	vin := dyn.V("abc")
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `cannot parse "abc" as a floating point number`,
		Location: vin.Location(),
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeFloatError(t *testing.T) {
	var typ float64
	vin := dyn.V(map[string]dyn.Value{"an": dyn.V("error")})
	_, err := Normalize(&typ, vin)
	assert.Len(t, err, 1)
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  `expected float, found map`,
		Location: dyn.Location{},
		Path:     dyn.EmptyPath,
	}, err[0])
}

func TestNormalizeAnchors(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	var typ Tmp
	vin := dyn.V(map[string]dyn.Value{
		"foo":    dyn.V("bar"),
		"anchor": dyn.V("anchor").MarkAnchor(),
	})

	vout, err := Normalize(typ, vin)
	assert.Len(t, err, 0)

	// The field that can be mapped to the struct field is retained.
	assert.Equal(t, map[string]any{
		"foo": "bar",
	}, vout.AsAny())
}
