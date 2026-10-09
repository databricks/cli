package structvar_test

import (
	"math"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const NL = "\n"

func TestYAMLSpecExample_2_1(t *testing.T) {
	file := "testdata/yaml/spec_example_2.1.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		[]ev{
			nv("Mark McGwire", []diag.Location{{File: file, Line: 3, Column: 3}}),
			nv("Sammy Sosa", []diag.Location{{File: file, Line: 4, Column: 3}}),
			nv("Ken Griffey", []diag.Location{{File: file, Line: 5, Column: 3}}),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_2(t *testing.T) {
	file := "testdata/yaml/spec_example_2.2.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"hr":  nv(65, []diag.Location{{File: file, Line: 3, Column: 6}}),
			"avg": nv(0.278, []diag.Location{{File: file, Line: 4, Column: 6}}),
			"rbi": nv(147, []diag.Location{{File: file, Line: 5, Column: 6}}),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_3(t *testing.T) {
	file := "testdata/yaml/spec_example_2.3.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"american": nv(
				[]ev{
					nv("Boston Red Sox", []diag.Location{{File: file, Line: 4, Column: 3}}),
					nv("Detroit Tigers", []diag.Location{{File: file, Line: 5, Column: 3}}),
					nv("New York Yankees", []diag.Location{{File: file, Line: 6, Column: 3}}),
				},
				[]diag.Location{{File: file, Line: 4, Column: 1}},
			),
			"national": nv(
				[]ev{
					nv("New York Mets", []diag.Location{{File: file, Line: 8, Column: 3}}),
					nv("Chicago Cubs", []diag.Location{{File: file, Line: 9, Column: 3}}),
					nv("Atlanta Braves", []diag.Location{{File: file, Line: 10, Column: 3}}),
				},
				[]diag.Location{{File: file, Line: 8, Column: 1}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_4(t *testing.T) {
	file := "testdata/yaml/spec_example_2.4.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		[]ev{
			nv(
				map[string]ev{
					"name": nv("Mark McGwire", []diag.Location{{File: file, Line: 4, Column: 9}}),
					"hr":   nv(65, []diag.Location{{File: file, Line: 5, Column: 9}}),
					"avg":  nv(0.278, []diag.Location{{File: file, Line: 6, Column: 9}}),
				},
				[]diag.Location{{File: file, Line: 4, Column: 3}},
			),
			nv(
				map[string]ev{
					"name": nv("Sammy Sosa", []diag.Location{{File: file, Line: 8, Column: 9}}),
					"hr":   nv(63, []diag.Location{{File: file, Line: 9, Column: 9}}),
					"avg":  nv(0.288, []diag.Location{{File: file, Line: 10, Column: 9}}),
				},
				[]diag.Location{{File: file, Line: 8, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_5(t *testing.T) {
	file := "testdata/yaml/spec_example_2.5.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		[]ev{
			nv(
				[]ev{
					nv("name", []diag.Location{{File: file, Line: 3, Column: 4}}),
					nv("hr", []diag.Location{{File: file, Line: 3, Column: 18}}),
					nv("avg", []diag.Location{{File: file, Line: 3, Column: 22}}),
				},
				[]diag.Location{{File: file, Line: 3, Column: 3}},
			),
			nv(
				[]ev{
					nv("Mark McGwire", []diag.Location{{File: file, Line: 4, Column: 4}}),
					nv(65, []diag.Location{{File: file, Line: 4, Column: 18}}),
					nv(0.278, []diag.Location{{File: file, Line: 4, Column: 22}}),
				},
				[]diag.Location{{File: file, Line: 4, Column: 3}},
			),
			nv(
				[]ev{
					nv("Sammy Sosa", []diag.Location{{File: file, Line: 5, Column: 4}}),
					nv(63, []diag.Location{{File: file, Line: 5, Column: 18}}),
					nv(0.288, []diag.Location{{File: file, Line: 5, Column: 22}}),
				},
				[]diag.Location{{File: file, Line: 5, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_6(t *testing.T) {
	file := "testdata/yaml/spec_example_2.6.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"Mark McGwire": nv(
				map[string]ev{
					"hr":  nv(65, []diag.Location{{File: file, Line: 3, Column: 20}}),
					"avg": nv(0.278, []diag.Location{{File: file, Line: 3, Column: 29}}),
				},
				[]diag.Location{{File: file, Line: 3, Column: 15}},
			),
			"Sammy Sosa": nv(
				map[string]ev{
					"hr":  nv(63, []diag.Location{{File: file, Line: 5, Column: 9}}),
					"avg": nv(0.288, []diag.Location{{File: file, Line: 6, Column: 10}}),
				},
				[]diag.Location{{File: file, Line: 4, Column: 13}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_7(t *testing.T) {
	file := "testdata/yaml/spec_example_2.7.yml"
	v, sv := loadExample(t, file)

	// Note: we do not support multiple documents in a single YAML file.

	assertDecoded(t, nv(
		[]ev{
			nv(
				"Mark McGwire",
				[]diag.Location{{File: file, Line: 5, Column: 3}},
			),
			nv(
				"Sammy Sosa",
				[]diag.Location{{File: file, Line: 6, Column: 3}},
			),
			nv(
				"Ken Griffey",
				[]diag.Location{{File: file, Line: 7, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 5, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_8(t *testing.T) {
	file := "testdata/yaml/spec_example_2.8.yml"
	v, sv := loadExample(t, file)

	// Note: we do not support multiple documents in a single YAML file.

	assertDecoded(t, nv(
		map[string]ev{
			"time":   nv("20:03:20", []diag.Location{{File: file, Line: 4, Column: 7}}),
			"player": nv("Sammy Sosa", []diag.Location{{File: file, Line: 5, Column: 9}}),
			"action": nv("strike (miss)", []diag.Location{{File: file, Line: 6, Column: 9}}),
		},
		[]diag.Location{{File: file, Line: 4, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_9(t *testing.T) {
	file := "testdata/yaml/spec_example_2.9.yml"
	v, sv := loadExample(t, file)

	// Note: we do not support multiple documents in a single YAML file.

	assertDecoded(t, nv(
		map[string]ev{
			"hr": nv(
				[]ev{
					nv("Mark McGwire", []diag.Location{{File: file, Line: 5, Column: 3}}),
					nv("Sammy Sosa", []diag.Location{{File: file, Line: 6, Column: 3}}),
				},
				[]diag.Location{{File: file, Line: 5, Column: 1}},
			),
			"rbi": nv(
				[]ev{
					nv("Sammy Sosa", []diag.Location{{File: file, Line: 9, Column: 3}}),
					nv("Ken Griffey", []diag.Location{{File: file, Line: 10, Column: 3}}),
				},
				[]diag.Location{{File: file, Line: 9, Column: 1}},
			),
		},
		[]diag.Location{{File: file, Line: 4, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_10(t *testing.T) {
	file := "testdata/yaml/spec_example_2.10.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"hr": nv(
				[]ev{
					nv("Mark McGwire", []diag.Location{{File: file, Line: 5, Column: 3}}),
					nv("Sammy Sosa", []diag.Location{{File: file, Line: 7, Column: 3}}),
				},
				[]diag.Location{{File: file, Line: 5, Column: 1}},
			),
			"rbi": nv(
				[]ev{
					// The location for an anchored value refers to the anchor, not the reference.
					// This is the same location as the anchor that appears in the "hr" mapping.
					nv("Sammy Sosa", []diag.Location{{File: file, Line: 7, Column: 3}}),
					nv("Ken Griffey", []diag.Location{{File: file, Line: 10, Column: 3}}),
				},
				[]diag.Location{{File: file, Line: 9, Column: 1}},
			),
		},
		[]diag.Location{{File: file, Line: 4, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_11(t *testing.T) {
	file := "testdata/yaml/spec_example_2.11.yml"
	// Note: non-string mapping keys are not supported by "go.yaml.in/yaml/v3".
	_, _, err := decodeFile(t, file)
	assert.ErrorContains(t, err, `: key is not a scalar`)
}

func TestYAMLSpecExample_2_12(t *testing.T) {
	file := "testdata/yaml/spec_example_2.12.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		[]ev{
			nv(
				map[string]ev{
					"item":     nv("Super Hoop", []diag.Location{{File: file, Line: 5, Column: 13}}),
					"quantity": nv(1, []diag.Location{{File: file, Line: 6, Column: 13}}),
				},
				[]diag.Location{{File: file, Line: 5, Column: 3}},
			),
			nv(
				map[string]ev{
					"item":     nv("Basketball", []diag.Location{{File: file, Line: 7, Column: 13}}),
					"quantity": nv(4, []diag.Location{{File: file, Line: 8, Column: 13}}),
				},
				[]diag.Location{{File: file, Line: 7, Column: 3}},
			),
			nv(
				map[string]ev{
					"item":     nv("Big Shoes", []diag.Location{{File: file, Line: 9, Column: 13}}),
					"quantity": nv(1, []diag.Location{{File: file, Line: 10, Column: 13}}),
				},
				[]diag.Location{{File: file, Line: 9, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 5, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_13(t *testing.T) {
	file := "testdata/yaml/spec_example_2.13.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		``+
			`\//||\/||`+NL+
			"// ||  ||__"+NL,
		[]diag.Location{{File: file, Line: 4, Column: 5}},
	), v, sv)
}

func TestYAMLSpecExample_2_14(t *testing.T) {
	file := "testdata/yaml/spec_example_2.14.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		`Mark McGwire's year was crippled by a knee injury.`+NL,
		[]diag.Location{{File: file, Line: 3, Column: 5}},
	), v, sv)
}

func TestYAMLSpecExample_2_15(t *testing.T) {
	file := "testdata/yaml/spec_example_2.15.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		``+
			`Sammy Sosa completed another fine season with great stats.`+NL+
			NL+
			`  63 Home Runs`+NL+
			`  0.288 Batting Average`+NL+
			NL+
			`What a year!`+NL,
		[]diag.Location{{File: file, Line: 3, Column: 5}},
	), v, sv)
}

func TestYAMLSpecExample_2_16(t *testing.T) {
	file := "testdata/yaml/spec_example_2.16.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"name": nv(
				"Mark McGwire",
				[]diag.Location{{File: file, Line: 3, Column: 7}},
			),
			"accomplishment": nv(
				`Mark set a major league home run record in 1998.`+NL,
				[]diag.Location{{File: file, Line: 4, Column: 17}},
			),
			"stats": nv(
				``+
					`65 Home Runs`+NL+
					`0.278 Batting Average`+NL,
				[]diag.Location{{File: file, Line: 7, Column: 8}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_17(t *testing.T) {
	file := "testdata/yaml/spec_example_2.17.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"unicode": nv(
				`Sosa did fine.`+"\u263A",
				[]diag.Location{{File: file, Line: 3, Column: 10}},
			),
			"control": nv(
				"\b1998\t1999\t2000\n",
				[]diag.Location{{File: file, Line: 4, Column: 10}},
			),
			"hex esc": nv(
				"\x0d\x0a is \r\n",
				[]diag.Location{{File: file, Line: 5, Column: 10}},
			),
			"single": nv(
				`"Howdy!" he cried.`,
				[]diag.Location{{File: file, Line: 7, Column: 9}},
			),
			"quoted": nv(
				` # Not a 'comment'.`,
				[]diag.Location{{File: file, Line: 8, Column: 9}},
			),
			"tie-fighter": nv(
				`|\-*-/|`,
				[]diag.Location{{File: file, Line: 9, Column: 14}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_18(t *testing.T) {
	file := "testdata/yaml/spec_example_2.18.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"plain": nv(
				`This unquoted scalar spans many lines.`,
				[]diag.Location{{File: file, Line: 4, Column: 3}},
			),
			"quoted": nv(
				`So does this quoted scalar.`+NL,
				[]diag.Location{{File: file, Line: 7, Column: 9}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_19(t *testing.T) {
	file := "testdata/yaml/spec_example_2.19.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"canonical": nv(
				12345,
				[]diag.Location{{File: file, Line: 3, Column: 12}},
			),
			"decimal": nv(
				12345,
				[]diag.Location{{File: file, Line: 4, Column: 10}},
			),
			"octal": nv(
				12,
				[]diag.Location{{File: file, Line: 5, Column: 8}},
			),
			"hexadecimal": nv(
				12,
				[]diag.Location{{File: file, Line: 6, Column: 14}},
			),
			"octal11": nv(
				12345,
				[]diag.Location{{File: file, Line: 15, Column: 10}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_20(t *testing.T) {
	file := "testdata/yaml/spec_example_2.20.yml"
	v, sv := loadExample(t, file)

	// Equality assertion doesn't work with NaNs.
	// See https://github.com/stretchr/testify/issues/624.
	//
	// Remove the NaN entry.
	m := v.(map[string]any)
	nan, ok := m["not a number"].(float64)
	require.True(t, ok)
	assert.True(t, math.IsNaN(nan))
	delete(m, "not a number")

	assertDecoded(t, nv(
		map[string]ev{
			"canonical": nv(
				1230.15,
				[]diag.Location{{File: file, Line: 3, Column: 12}},
			),
			"exponential": nv(
				1230.15,
				[]diag.Location{{File: file, Line: 4, Column: 14}},
			),
			"fixed": nv(
				1230.15,
				[]diag.Location{{File: file, Line: 5, Column: 8}},
			),
			"negative infinity": nv(
				math.Inf(-1),
				[]diag.Location{{File: file, Line: 6, Column: 20}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_21(t *testing.T) {
	file := "testdata/yaml/spec_example_2.21.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"null": nv(
				nil,
				[]diag.Location{{File: file, Line: 3, Column: 6}},
			),
			"booleans": nv(
				[]ev{
					nv(true, []diag.Location{{File: file, Line: 4, Column: 13}}),
					nv(false, []diag.Location{{File: file, Line: 4, Column: 19}}),
				},
				[]diag.Location{{File: file, Line: 4, Column: 11}},
			),
			"string": nv(
				"012345",
				[]diag.Location{{File: file, Line: 5, Column: 9}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_22(t *testing.T) {
	file := "testdata/yaml/spec_example_2.22.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"canonical": nv(
				"2001-12-15T02:59:43.1Z",
				[]diag.Location{{File: file, Line: 3, Column: 12}},
			),
			"iso8601": nv(
				"2001-12-14t21:59:43.10-05:00",
				[]diag.Location{{File: file, Line: 4, Column: 10}},
			),
			"spaced": nv(
				// This is parsed as a string, not a timestamp,
				// both by "go.yaml.in/yaml/v3" and by our implementation.
				"2001-12-14 21:59:43.10 -5",
				[]diag.Location{{File: file, Line: 5, Column: 9}},
			),
			"date": nv(
				"2002-12-14",
				[]diag.Location{{File: file, Line: 6, Column: 7}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 1}},
	), v, sv)
}

func TestYAMLSpecExample_2_23(t *testing.T) {
	file := "testdata/yaml/spec_example_2.23.yml"
	// Note: the !!binary tag is not supported by us.
	_, _, err := decodeFile(t, file)
	assert.ErrorContains(t, err, `: unknown tag: !!binary`)
}

func TestYAMLSpecExample_2_24(t *testing.T) {
	file := "testdata/yaml/spec_example_2.24.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		[]ev{
			nv(
				map[string]ev{
					"center": nv(
						map[string]ev{
							"x": nv(73, []diag.Location{{File: file, Line: 8, Column: 23}}),
							"y": nv(129, []diag.Location{{File: file, Line: 8, Column: 30}}),
						},
						[]diag.Location{{File: file, Line: 8, Column: 11}},
					),
					"radius": nv(7, []diag.Location{{File: file, Line: 9, Column: 11}}),
				},
				[]diag.Location{{File: file, Line: 7, Column: 3}},
			),
			nv(
				map[string]ev{
					"start": nv(
						map[string]ev{
							"x": nv(73, []diag.Location{{File: file, Line: 8, Column: 23}}),
							"y": nv(129, []diag.Location{{File: file, Line: 8, Column: 30}}),
						},
						[]diag.Location{{File: file, Line: 8, Column: 11}},
					),
					"finish": nv(
						map[string]ev{
							"x": nv(89, []diag.Location{{File: file, Line: 12, Column: 16}}),
							"y": nv(102, []diag.Location{{File: file, Line: 12, Column: 23}}),
						},
						[]diag.Location{{File: file, Line: 12, Column: 11}},
					),
				},
				[]diag.Location{{File: file, Line: 10, Column: 3}},
			),
			nv(
				map[string]ev{
					"start": nv(
						map[string]ev{
							"x": nv(73, []diag.Location{{File: file, Line: 8, Column: 23}}),
							"y": nv(129, []diag.Location{{File: file, Line: 8, Column: 30}}),
						},
						[]diag.Location{{File: file, Line: 8, Column: 11}},
					),
					"color": nv(16772795, []diag.Location{{File: file, Line: 15, Column: 10}}),
					"text":  nv("Pretty vector drawing.", []diag.Location{{File: file, Line: 16, Column: 9}}),
				},
				[]diag.Location{{File: file, Line: 13, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 4, Column: 5}},
	), v, sv)
}

func TestYAMLSpecExample_2_25(t *testing.T) {
	file := "testdata/yaml/spec_example_2.25.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"Mark McGwire": nv(nil, []diag.Location{{File: file, Line: 8, Column: 1}}),
			"Sammy Sosa":   nv(nil, []diag.Location{{File: file, Line: 9, Column: 1}}),
			"Ken Griffey":  nv(nil, []diag.Location{{File: file, Line: 10, Column: 1}}),
		},
		[]diag.Location{{File: file, Line: 6, Column: 5}},
	), v, sv)
}

func TestYAMLSpecExample_2_26(t *testing.T) {
	file := "testdata/yaml/spec_example_2.26.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		[]ev{
			nv(
				map[string]ev{
					"Mark McGwire": nv(65, []diag.Location{{File: file, Line: 7, Column: 17}}),
				},
				[]diag.Location{{File: file, Line: 7, Column: 3}},
			),
			nv(
				map[string]ev{
					"Sammy Sosa": nv(63, []diag.Location{{File: file, Line: 8, Column: 15}}),
				},
				[]diag.Location{{File: file, Line: 8, Column: 3}},
			),
			nv(
				map[string]ev{
					"Ken Griffey": nv(58, []diag.Location{{File: file, Line: 9, Column: 16}}),
				},
				[]diag.Location{{File: file, Line: 9, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 6, Column: 5}},
	), v, sv)
}

func TestYAMLSpecExample_2_27(t *testing.T) {
	file := "testdata/yaml/spec_example_2.27.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"invoice": nv(
				34843,
				[]diag.Location{{File: file, Line: 4, Column: 10}},
			),
			"date": nv(
				"2001-01-23",
				[]diag.Location{{File: file, Line: 5, Column: 10}},
			),
			"bill-to": nv(
				map[string]ev{
					"given": nv(
						"Chris",
						[]diag.Location{{File: file, Line: 7, Column: 12}},
					),
					"family": nv(
						"Dumars",
						[]diag.Location{{File: file, Line: 8, Column: 12}},
					),
					"address": nv(
						map[string]ev{
							"lines": nv(
								"458 Walkman Dr.\nSuite #292\n",
								[]diag.Location{{File: file, Line: 10, Column: 12}},
							),
							"city": nv(
								"Royal Oak",
								[]diag.Location{{File: file, Line: 13, Column: 15}},
							),
							"state": nv(
								"MI",
								[]diag.Location{{File: file, Line: 14, Column: 15}},
							),
							"postal": nv(
								48046,
								[]diag.Location{{File: file, Line: 15, Column: 15}},
							),
						},
						[]diag.Location{{File: file, Line: 10, Column: 5}},
					),
				},
				[]diag.Location{{File: file, Line: 6, Column: 10}},
			),
			"ship-to": nv(
				map[string]ev{
					"given": nv(
						"Chris",
						[]diag.Location{{File: file, Line: 7, Column: 12}},
					),
					"family": nv(
						"Dumars",
						[]diag.Location{{File: file, Line: 8, Column: 12}},
					),
					"address": nv(
						map[string]ev{
							"lines": nv(
								"458 Walkman Dr.\nSuite #292\n",
								[]diag.Location{{File: file, Line: 10, Column: 12}},
							),
							"city": nv(
								"Royal Oak",
								[]diag.Location{{File: file, Line: 13, Column: 15}},
							),
							"state": nv(
								"MI",
								[]diag.Location{{File: file, Line: 14, Column: 15}},
							),
							"postal": nv(
								48046,
								[]diag.Location{{File: file, Line: 15, Column: 15}},
							),
						},
						[]diag.Location{{File: file, Line: 10, Column: 5}},
					),
				},
				[]diag.Location{{File: file, Line: 6, Column: 10}},
			),
			"product": nv(
				[]ev{
					nv(
						map[string]ev{
							"sku": nv(
								"BL394D",
								[]diag.Location{{File: file, Line: 18, Column: 17}},
							),
							"quantity": nv(
								4,
								[]diag.Location{{File: file, Line: 19, Column: 17}},
							),
							"description": nv(
								"Basketball",
								[]diag.Location{{File: file, Line: 20, Column: 17}},
							),
							"price": nv(
								450.0,
								[]diag.Location{{File: file, Line: 21, Column: 17}},
							),
						},
						[]diag.Location{{File: file, Line: 18, Column: 3}},
					), nv(
						map[string]ev{
							"sku": nv(
								"BL4438H",
								[]diag.Location{{File: file, Line: 22, Column: 17}},
							),
							"quantity": nv(
								1,
								[]diag.Location{{File: file, Line: 23, Column: 17}},
							),
							"description": nv(
								"Super Hoop",
								[]diag.Location{{File: file, Line: 24, Column: 17}},
							),
							"price": nv(
								2392.0,
								[]diag.Location{{File: file, Line: 25, Column: 17}},
							),
						},
						[]diag.Location{{File: file, Line: 22, Column: 3}},
					),
				},
				[]diag.Location{{File: file, Line: 18, Column: 1}},
			),
			"tax": nv(
				251.42,
				[]diag.Location{{File: file, Line: 26, Column: 8}},
			),
			"total": nv(
				4443.52,
				[]diag.Location{{File: file, Line: 27, Column: 8}},
			),
			"comments": nv(
				"Late afternoon is best. Backup contact is Nancy Billsmer @ 338-4338.",
				[]diag.Location{{File: file, Line: 29, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 3, Column: 5}},
	), v, sv)
}

func TestYAMLSpecExample_2_28(t *testing.T) {
	file := "testdata/yaml/spec_example_2.28.yml"
	v, sv := loadExample(t, file)

	assertDecoded(t, nv(
		map[string]ev{
			"Time": nv(
				"2001-11-23 15:01:42 -5",
				[]diag.Location{{File: file, Line: 4, Column: 7}},
			),
			"User": nv(
				"ed",
				[]diag.Location{{File: file, Line: 5, Column: 7}},
			),
			"Warning": nv(
				"This is an error message for the log file",
				[]diag.Location{{File: file, Line: 7, Column: 3}},
			),
		},
		[]diag.Location{{File: file, Line: 4, Column: 1}},
	), v, sv)
}
