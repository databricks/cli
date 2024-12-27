package yamlloader_test

import (
	"bytes"
	"math"
	"os"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/require"
)

const NL = "\n"

func loadExample(t *testing.T, file string) dyn.Value {
	input, err := os.ReadFile(file)
	require.NoError(t, err)
	self, err := yamlloader.LoadYAML(file, bytes.NewBuffer(input))
	require.NoError(t, err)
	return self
}

func TestYAMLSpecExample_2_1(t *testing.T) {
	file := "testdata/spec_example_2.1.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue("Mark McGwire", []dyn.Location{{File: file, Line: 3, Column: 3}}),
			dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 4, Column: 3}}),
			dyn.NewValue("Ken Griffey", []dyn.Location{{File: file, Line: 5, Column: 3}}),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_2(t *testing.T) {
	file := "testdata/spec_example_2.2.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"hr":  dyn.NewValue(65, []dyn.Location{{File: file, Line: 3, Column: 6}}),
			"avg": dyn.NewValue(0.278, []dyn.Location{{File: file, Line: 4, Column: 6}}),
			"rbi": dyn.NewValue(147, []dyn.Location{{File: file, Line: 5, Column: 6}}),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_3(t *testing.T) {
	file := "testdata/spec_example_2.3.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"american": dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("Boston Red Sox", []dyn.Location{{File: file, Line: 4, Column: 3}}),
					dyn.NewValue("Detroit Tigers", []dyn.Location{{File: file, Line: 5, Column: 3}}),
					dyn.NewValue("New York Yankees", []dyn.Location{{File: file, Line: 6, Column: 3}}),
				},
				[]dyn.Location{{File: file, Line: 4, Column: 1}},
			),
			"national": dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("New York Mets", []dyn.Location{{File: file, Line: 8, Column: 3}}),
					dyn.NewValue("Chicago Cubs", []dyn.Location{{File: file, Line: 9, Column: 3}}),
					dyn.NewValue("Atlanta Braves", []dyn.Location{{File: file, Line: 10, Column: 3}}),
				},
				[]dyn.Location{{File: file, Line: 8, Column: 1}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_4(t *testing.T) {
	file := "testdata/spec_example_2.4.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue(
				map[string]dyn.Value{
					"name": dyn.NewValue("Mark McGwire", []dyn.Location{{File: file, Line: 4, Column: 9}}),
					"hr":   dyn.NewValue(65, []dyn.Location{{File: file, Line: 5, Column: 9}}),
					"avg":  dyn.NewValue(0.278, []dyn.Location{{File: file, Line: 6, Column: 9}}),
				},
				[]dyn.Location{{File: file, Line: 4, Column: 3}},
			),
			dyn.NewValue(
				map[string]dyn.Value{
					"name": dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 8, Column: 9}}),
					"hr":   dyn.NewValue(63, []dyn.Location{{File: file, Line: 9, Column: 9}}),
					"avg":  dyn.NewValue(0.288, []dyn.Location{{File: file, Line: 10, Column: 9}}),
				},
				[]dyn.Location{{File: file, Line: 8, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_5(t *testing.T) {
	file := "testdata/spec_example_2.5.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("name", []dyn.Location{{File: file, Line: 3, Column: 4}}),
					dyn.NewValue("hr", []dyn.Location{{File: file, Line: 3, Column: 18}}),
					dyn.NewValue("avg", []dyn.Location{{File: file, Line: 3, Column: 22}}),
				},
				[]dyn.Location{{File: file, Line: 3, Column: 3}},
			),
			dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("Mark McGwire", []dyn.Location{{File: file, Line: 4, Column: 4}}),
					dyn.NewValue(65, []dyn.Location{{File: file, Line: 4, Column: 18}}),
					dyn.NewValue(0.278, []dyn.Location{{File: file, Line: 4, Column: 22}}),
				},
				[]dyn.Location{{File: file, Line: 4, Column: 3}},
			),
			dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 5, Column: 4}}),
					dyn.NewValue(63, []dyn.Location{{File: file, Line: 5, Column: 18}}),
					dyn.NewValue(0.288, []dyn.Location{{File: file, Line: 5, Column: 22}}),
				},
				[]dyn.Location{{File: file, Line: 5, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_6(t *testing.T) {
	file := "testdata/spec_example_2.6.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"Mark McGwire": dyn.NewValue(
				map[string]dyn.Value{
					"hr":  dyn.NewValue(65, []dyn.Location{{File: file, Line: 3, Column: 20}}),
					"avg": dyn.NewValue(0.278, []dyn.Location{{File: file, Line: 3, Column: 29}}),
				},
				[]dyn.Location{{File: file, Line: 3, Column: 15}},
			),
			"Sammy Sosa": dyn.NewValue(
				map[string]dyn.Value{
					"hr":  dyn.NewValue(63, []dyn.Location{{File: file, Line: 5, Column: 9}}),
					"avg": dyn.NewValue(0.288, []dyn.Location{{File: file, Line: 6, Column: 10}}),
				},
				[]dyn.Location{{File: file, Line: 4, Column: 13}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_7(t *testing.T) {
	file := "testdata/spec_example_2.7.yml"
	self := loadExample(t, file)

	// Note: we do not support multiple documents in a single YAML file.

	assert.Equal(t, dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue(
				"Mark McGwire",
				[]dyn.Location{{File: file, Line: 5, Column: 3}},
			),
			dyn.NewValue(
				"Sammy Sosa",
				[]dyn.Location{{File: file, Line: 6, Column: 3}},
			),
			dyn.NewValue(
				"Ken Griffey",
				[]dyn.Location{{File: file, Line: 7, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 5, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_8(t *testing.T) {
	file := "testdata/spec_example_2.8.yml"
	self := loadExample(t, file)

	// Note: we do not support multiple documents in a single YAML file.

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"time":   dyn.NewValue("20:03:20", []dyn.Location{{File: file, Line: 4, Column: 7}}),
			"player": dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 5, Column: 9}}),
			"action": dyn.NewValue("strike (miss)", []dyn.Location{{File: file, Line: 6, Column: 9}}),
		},
		[]dyn.Location{{File: file, Line: 4, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_9(t *testing.T) {
	file := "testdata/spec_example_2.9.yml"
	self := loadExample(t, file)

	// Note: we do not support multiple documents in a single YAML file.

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"hr": dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("Mark McGwire", []dyn.Location{{File: file, Line: 5, Column: 3}}),
					dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 6, Column: 3}}),
				},
				[]dyn.Location{{File: file, Line: 5, Column: 1}},
			),
			"rbi": dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 9, Column: 3}}),
					dyn.NewValue("Ken Griffey", []dyn.Location{{File: file, Line: 10, Column: 3}}),
				},
				[]dyn.Location{{File: file, Line: 9, Column: 1}},
			),
		},
		[]dyn.Location{{File: file, Line: 4, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_10(t *testing.T) {
	file := "testdata/spec_example_2.10.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"hr": dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue("Mark McGwire", []dyn.Location{{File: file, Line: 5, Column: 3}}),
					dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 7, Column: 3}}),
				},
				[]dyn.Location{{File: file, Line: 5, Column: 1}},
			),
			"rbi": dyn.NewValue(
				[]dyn.Value{
					// The location for an anchored value refers to the anchor, not the reference.
					// This is the same location as the anchor that appears in the "hr" mapping.
					dyn.NewValue("Sammy Sosa", []dyn.Location{{File: file, Line: 7, Column: 3}}),
					dyn.NewValue("Ken Griffey", []dyn.Location{{File: file, Line: 10, Column: 3}}),
				},
				[]dyn.Location{{File: file, Line: 9, Column: 1}},
			),
		},
		[]dyn.Location{{File: file, Line: 4, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_11(t *testing.T) {
	file := "testdata/spec_example_2.11.yml"
	input, err := os.ReadFile(file)
	require.NoError(t, err)

	// Note: non-string mapping keys are not supported by "gopkg.in/yaml.v3".
	_, err = yamlloader.LoadYAML(file, bytes.NewBuffer(input))
	assert.ErrorContains(t, err, `: key is not a scalar`)
}

func TestYAMLSpecExample_2_12(t *testing.T) {
	file := "testdata/spec_example_2.12.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue(
				map[string]dyn.Value{
					"item":     dyn.NewValue("Super Hoop", []dyn.Location{{File: file, Line: 5, Column: 13}}),
					"quantity": dyn.NewValue(1, []dyn.Location{{File: file, Line: 6, Column: 13}}),
				},
				[]dyn.Location{{File: file, Line: 5, Column: 3}},
			),
			dyn.NewValue(
				map[string]dyn.Value{
					"item":     dyn.NewValue("Basketball", []dyn.Location{{File: file, Line: 7, Column: 13}}),
					"quantity": dyn.NewValue(4, []dyn.Location{{File: file, Line: 8, Column: 13}}),
				},
				[]dyn.Location{{File: file, Line: 7, Column: 3}},
			),
			dyn.NewValue(
				map[string]dyn.Value{
					"item":     dyn.NewValue("Big Shoes", []dyn.Location{{File: file, Line: 9, Column: 13}}),
					"quantity": dyn.NewValue(1, []dyn.Location{{File: file, Line: 10, Column: 13}}),
				},
				[]dyn.Location{{File: file, Line: 9, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 5, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_13(t *testing.T) {
	file := "testdata/spec_example_2.13.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		``+
			`\//||\/||`+NL+
			"// ||  ||__"+NL,
		[]dyn.Location{{File: file, Line: 4, Column: 5}},
	), self)
}

func TestYAMLSpecExample_2_14(t *testing.T) {
	file := "testdata/spec_example_2.14.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		`Mark McGwire's year was crippled by a knee injury.`+NL,
		[]dyn.Location{{File: file, Line: 3, Column: 5}},
	), self)
}

func TestYAMLSpecExample_2_15(t *testing.T) {
	file := "testdata/spec_example_2.15.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		``+
			`Sammy Sosa completed another fine season with great stats.`+NL+
			NL+
			`  63 Home Runs`+NL+
			`  0.288 Batting Average`+NL+
			NL+
			`What a year!`+NL,
		[]dyn.Location{{File: file, Line: 3, Column: 5}},
	), self)
}

func TestYAMLSpecExample_2_16(t *testing.T) {
	file := "testdata/spec_example_2.16.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"name": dyn.NewValue(
				"Mark McGwire",
				[]dyn.Location{{File: file, Line: 3, Column: 7}},
			),
			"accomplishment": dyn.NewValue(
				`Mark set a major league home run record in 1998.`+NL,
				[]dyn.Location{{File: file, Line: 4, Column: 17}},
			),
			"stats": dyn.NewValue(
				``+
					`65 Home Runs`+NL+
					`0.278 Batting Average`+NL,
				[]dyn.Location{{File: file, Line: 7, Column: 8}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_17(t *testing.T) {
	file := "testdata/spec_example_2.17.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"unicode": dyn.NewValue(
				`Sosa did fine.`+"\u263A",
				[]dyn.Location{{File: file, Line: 3, Column: 10}},
			),
			"control": dyn.NewValue(
				"\b1998\t1999\t2000\n",
				[]dyn.Location{{File: file, Line: 4, Column: 10}},
			),
			"hex esc": dyn.NewValue(
				"\x0d\x0a is \r\n",
				[]dyn.Location{{File: file, Line: 5, Column: 10}},
			),
			"single": dyn.NewValue(
				`"Howdy!" he cried.`,
				[]dyn.Location{{File: file, Line: 7, Column: 9}},
			),
			"quoted": dyn.NewValue(
				` # Not a 'comment'.`,
				[]dyn.Location{{File: file, Line: 8, Column: 9}},
			),
			"tie-fighter": dyn.NewValue(
				`|\-*-/|`,
				[]dyn.Location{{File: file, Line: 9, Column: 14}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_18(t *testing.T) {
	file := "testdata/spec_example_2.18.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"plain": dyn.NewValue(
				`This unquoted scalar spans many lines.`,
				[]dyn.Location{{File: file, Line: 4, Column: 3}},
			),
			"quoted": dyn.NewValue(
				`So does this quoted scalar.`+NL,
				[]dyn.Location{{File: file, Line: 7, Column: 9}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_19(t *testing.T) {
	file := "testdata/spec_example_2.19.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"canonical": dyn.NewValue(
				12345,
				[]dyn.Location{{File: file, Line: 3, Column: 12}},
			),
			"decimal": dyn.NewValue(
				12345,
				[]dyn.Location{{File: file, Line: 4, Column: 10}},
			),
			"octal": dyn.NewValue(
				12,
				[]dyn.Location{{File: file, Line: 5, Column: 8}},
			),
			"hexadecimal": dyn.NewValue(
				12,
				[]dyn.Location{{File: file, Line: 6, Column: 14}},
			),
			"octal11": dyn.NewValue(
				12345,
				[]dyn.Location{{File: file, Line: 15, Column: 10}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_20(t *testing.T) {
	file := "testdata/spec_example_2.20.yml"
	self := loadExample(t, file)

	// Equality assertion doesn't work with NaNs.
	// See https://github.com/stretchr/testify/issues/624.
	//
	// Remove the NaN entry.
	self, _ = dyn.Walk(self, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if f, ok := v.AsFloat(); ok && math.IsNaN(f) {
			return dyn.InvalidValue, dyn.ErrDrop
		}
		return v, nil
	})

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"canonical": dyn.NewValue(
				1230.15,
				[]dyn.Location{{File: file, Line: 3, Column: 12}},
			),
			"exponential": dyn.NewValue(
				1230.15,
				[]dyn.Location{{File: file, Line: 4, Column: 14}},
			),
			"fixed": dyn.NewValue(
				1230.15,
				[]dyn.Location{{File: file, Line: 5, Column: 8}},
			),
			"negative infinity": dyn.NewValue(
				math.Inf(-1),
				[]dyn.Location{{File: file, Line: 6, Column: 20}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_21(t *testing.T) {
	file := "testdata/spec_example_2.21.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"null": dyn.NewValue(
				nil,
				[]dyn.Location{{File: file, Line: 3, Column: 6}},
			),
			"booleans": dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(true, []dyn.Location{{File: file, Line: 4, Column: 13}}),
					dyn.NewValue(false, []dyn.Location{{File: file, Line: 4, Column: 19}}),
				},
				[]dyn.Location{{File: file, Line: 4, Column: 11}},
			),
			"string": dyn.NewValue(
				"012345",
				[]dyn.Location{{File: file, Line: 5, Column: 9}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_22(t *testing.T) {
	file := "testdata/spec_example_2.22.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"canonical": dyn.NewValue(
				dyn.MustTime("2001-12-15T02:59:43.1Z"),
				[]dyn.Location{{File: file, Line: 3, Column: 12}},
			),
			"iso8601": dyn.NewValue(
				dyn.MustTime("2001-12-14t21:59:43.10-05:00"),
				[]dyn.Location{{File: file, Line: 4, Column: 10}},
			),
			"spaced": dyn.NewValue(
				// This is parsed as a string, not a timestamp,
				// both by "gopkg.in/yaml.v3" and by our implementation.
				"2001-12-14 21:59:43.10 -5",
				[]dyn.Location{{File: file, Line: 5, Column: 9}},
			),
			"date": dyn.NewValue(
				dyn.MustTime("2002-12-14"),
				[]dyn.Location{{File: file, Line: 6, Column: 7}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 1}},
	), self)
}

func TestYAMLSpecExample_2_23(t *testing.T) {
	file := "testdata/spec_example_2.23.yml"
	input, err := os.ReadFile(file)
	require.NoError(t, err)

	// Note: the !!binary tag is not supported by us.

	_, err = yamlloader.LoadYAML(file, bytes.NewBuffer(input))
	assert.ErrorContains(t, err, `: unknown tag: !!binary`)
}

func TestYAMLSpecExample_2_24(t *testing.T) {
	file := "testdata/spec_example_2.24.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue(
				map[string]dyn.Value{
					"center": dyn.NewValue(
						map[string]dyn.Value{
							"x": dyn.NewValue(73, []dyn.Location{{File: file, Line: 8, Column: 23}}),
							"y": dyn.NewValue(129, []dyn.Location{{File: file, Line: 8, Column: 30}}),
						},
						[]dyn.Location{{File: file, Line: 8, Column: 11}},
					),
					"radius": dyn.NewValue(7, []dyn.Location{{File: file, Line: 9, Column: 11}}),
				},
				[]dyn.Location{{File: file, Line: 7, Column: 3}},
			),
			dyn.NewValue(
				map[string]dyn.Value{
					"start": dyn.NewValue(
						map[string]dyn.Value{
							"x": dyn.NewValue(73, []dyn.Location{{File: file, Line: 8, Column: 23}}),
							"y": dyn.NewValue(129, []dyn.Location{{File: file, Line: 8, Column: 30}}),
						},
						[]dyn.Location{{File: file, Line: 8, Column: 11}},
					),
					"finish": dyn.NewValue(
						map[string]dyn.Value{
							"x": dyn.NewValue(89, []dyn.Location{{File: file, Line: 12, Column: 16}}),
							"y": dyn.NewValue(102, []dyn.Location{{File: file, Line: 12, Column: 23}}),
						},
						[]dyn.Location{{File: file, Line: 12, Column: 11}},
					),
				},
				[]dyn.Location{{File: file, Line: 10, Column: 3}},
			),
			dyn.NewValue(
				map[string]dyn.Value{
					"start": dyn.NewValue(
						map[string]dyn.Value{
							"x": dyn.NewValue(73, []dyn.Location{{File: file, Line: 8, Column: 23}}),
							"y": dyn.NewValue(129, []dyn.Location{{File: file, Line: 8, Column: 30}}),
						},
						[]dyn.Location{{File: file, Line: 8, Column: 11}},
					),
					"color": dyn.NewValue(16772795, []dyn.Location{{File: file, Line: 15, Column: 10}}),
					"text":  dyn.NewValue("Pretty vector drawing.", []dyn.Location{{File: file, Line: 16, Column: 9}}),
				},
				[]dyn.Location{{File: file, Line: 13, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 4, Column: 5}},
	), self)
}

func TestYAMLSpecExample_2_25(t *testing.T) {
	file := "testdata/spec_example_2.25.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"Mark McGwire": dyn.NewValue(nil, []dyn.Location{{File: file, Line: 8, Column: 1}}),
			"Sammy Sosa":   dyn.NewValue(nil, []dyn.Location{{File: file, Line: 9, Column: 1}}),
			"Ken Griffey":  dyn.NewValue(nil, []dyn.Location{{File: file, Line: 10, Column: 1}}),
		},
		[]dyn.Location{{File: file, Line: 6, Column: 5}},
	), self)
}

func TestYAMLSpecExample_2_26(t *testing.T) {
	file := "testdata/spec_example_2.26.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue(
				map[string]dyn.Value{
					"Mark McGwire": dyn.NewValue(65, []dyn.Location{{File: file, Line: 7, Column: 17}}),
				},
				[]dyn.Location{{File: file, Line: 7, Column: 3}},
			),
			dyn.NewValue(
				map[string]dyn.Value{
					"Sammy Sosa": dyn.NewValue(63, []dyn.Location{{File: file, Line: 8, Column: 15}}),
				},
				[]dyn.Location{{File: file, Line: 8, Column: 3}},
			),
			dyn.NewValue(
				map[string]dyn.Value{
					"Ken Griffey": dyn.NewValue(58, []dyn.Location{{File: file, Line: 9, Column: 16}}),
				},
				[]dyn.Location{{File: file, Line: 9, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 6, Column: 5}},
	), self)
}

func TestYAMLSpecExample_2_27(t *testing.T) {
	file := "testdata/spec_example_2.27.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"invoice": dyn.NewValue(
				34843,
				[]dyn.Location{{File: file, Line: 4, Column: 10}},
			),
			"date": dyn.NewValue(
				dyn.MustTime("2001-01-23"),
				[]dyn.Location{{File: file, Line: 5, Column: 10}},
			),
			"bill-to": dyn.NewValue(
				map[string]dyn.Value{
					"given": dyn.NewValue(
						"Chris",
						[]dyn.Location{{File: file, Line: 7, Column: 12}},
					),
					"family": dyn.NewValue(
						"Dumars",
						[]dyn.Location{{File: file, Line: 8, Column: 12}},
					),
					"address": dyn.NewValue(
						map[string]dyn.Value{
							"lines": dyn.NewValue(
								"458 Walkman Dr.\nSuite #292\n",
								[]dyn.Location{{File: file, Line: 10, Column: 12}},
							),
							"city": dyn.NewValue(
								"Royal Oak",
								[]dyn.Location{{File: file, Line: 13, Column: 15}},
							),
							"state": dyn.NewValue(
								"MI",
								[]dyn.Location{{File: file, Line: 14, Column: 15}},
							),
							"postal": dyn.NewValue(
								48046,
								[]dyn.Location{{File: file, Line: 15, Column: 15}},
							),
						},
						[]dyn.Location{{File: file, Line: 10, Column: 5}},
					),
				},
				[]dyn.Location{{File: file, Line: 6, Column: 10}},
			),
			"ship-to": dyn.NewValue(
				map[string]dyn.Value{
					"given": dyn.NewValue(
						"Chris",
						[]dyn.Location{{File: file, Line: 7, Column: 12}},
					),
					"family": dyn.NewValue(
						"Dumars",
						[]dyn.Location{{File: file, Line: 8, Column: 12}},
					),
					"address": dyn.NewValue(
						map[string]dyn.Value{
							"lines": dyn.NewValue(
								"458 Walkman Dr.\nSuite #292\n",
								[]dyn.Location{{File: file, Line: 10, Column: 12}},
							),
							"city": dyn.NewValue(
								"Royal Oak",
								[]dyn.Location{{File: file, Line: 13, Column: 15}},
							),
							"state": dyn.NewValue(
								"MI",
								[]dyn.Location{{File: file, Line: 14, Column: 15}},
							),
							"postal": dyn.NewValue(
								48046,
								[]dyn.Location{{File: file, Line: 15, Column: 15}},
							),
						},
						[]dyn.Location{{File: file, Line: 10, Column: 5}},
					),
				},
				[]dyn.Location{{File: file, Line: 6, Column: 10}},
			),
			"product": dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(
						map[string]dyn.Value{
							"sku": dyn.NewValue(
								"BL394D",
								[]dyn.Location{{File: file, Line: 18, Column: 17}},
							),
							"quantity": dyn.NewValue(
								4,
								[]dyn.Location{{File: file, Line: 19, Column: 17}},
							),
							"description": dyn.NewValue(
								"Basketball",
								[]dyn.Location{{File: file, Line: 20, Column: 17}},
							),
							"price": dyn.NewValue(
								450.0,
								[]dyn.Location{{File: file, Line: 21, Column: 17}},
							),
						},
						[]dyn.Location{{File: file, Line: 18, Column: 3}},
					), dyn.NewValue(
						map[string]dyn.Value{
							"sku": dyn.NewValue(
								"BL4438H",
								[]dyn.Location{{File: file, Line: 22, Column: 17}},
							),
							"quantity": dyn.NewValue(
								1,
								[]dyn.Location{{File: file, Line: 23, Column: 17}},
							),
							"description": dyn.NewValue(
								"Super Hoop",
								[]dyn.Location{{File: file, Line: 24, Column: 17}},
							),
							"price": dyn.NewValue(
								2392.0,
								[]dyn.Location{{File: file, Line: 25, Column: 17}},
							),
						},
						[]dyn.Location{{File: file, Line: 22, Column: 3}},
					),
				},
				[]dyn.Location{{File: file, Line: 18, Column: 1}},
			),
			"tax": dyn.NewValue(
				251.42,
				[]dyn.Location{{File: file, Line: 26, Column: 8}},
			),
			"total": dyn.NewValue(
				4443.52,
				[]dyn.Location{{File: file, Line: 27, Column: 8}},
			),
			"comments": dyn.NewValue(
				"Late afternoon is best. Backup contact is Nancy Billsmer @ 338-4338.",
				[]dyn.Location{{File: file, Line: 29, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 3, Column: 5}},
	), self)
}

func TestYAMLSpecExample_2_28(t *testing.T) {
	file := "testdata/spec_example_2.28.yml"
	self := loadExample(t, file)

	assert.Equal(t, dyn.NewValue(
		map[string]dyn.Value{
			"Time": dyn.NewValue(
				"2001-11-23 15:01:42 -5",
				[]dyn.Location{{File: file, Line: 4, Column: 7}},
			),
			"User": dyn.NewValue(
				"ed",
				[]dyn.Location{{File: file, Line: 5, Column: 7}},
			),
			"Warning": dyn.NewValue(
				"This is an error message for the log file",
				[]dyn.Location{{File: file, Line: 7, Column: 3}},
			),
		},
		[]dyn.Location{{File: file, Line: 4, Column: 1}},
	), self)
}
