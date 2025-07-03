package dyn_test

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestWalkReadOnly(t *testing.T) {
	tests := []struct {
		name           string
		input          dyn.Value
		expectedPaths  []dyn.Path
		expectedValues []dyn.Value
	}{
		{
			name: "simple map",
			input: dyn.V(dyn.NewMappingFromPairs(
				[]dyn.Pair{
					{Key: dyn.V("a"), Value: dyn.V("1")},
					{Key: dyn.V("b"), Value: dyn.V("2")},
				},
			)),
			expectedPaths: []dyn.Path{
				dyn.EmptyPath,
				{dyn.Key("a")},
				{dyn.Key("b")},
			},
			expectedValues: []dyn.Value{
				dyn.V(dyn.NewMappingFromPairs(
					[]dyn.Pair{
						{Key: dyn.V("a"), Value: dyn.V("1")},
						{Key: dyn.V("b"), Value: dyn.V("2")},
					},
				)),
				dyn.V("1"),
				dyn.V("2"),
			},
		},
		{
			name: "nested map",
			input: dyn.V(map[string]dyn.Value{
				"a": dyn.V(map[string]dyn.Value{
					"b": dyn.V("1"),
				}),
			}),
			expectedPaths: []dyn.Path{
				dyn.EmptyPath,
				{dyn.Key("a")},
				{dyn.Key("a"), dyn.Key("b")},
			},
			expectedValues: []dyn.Value{
				dyn.V(map[string]dyn.Value{
					"a": dyn.V(map[string]dyn.Value{
						"b": dyn.V("1"),
					}),
				}),
				dyn.V(map[string]dyn.Value{
					"b": dyn.V("1"),
				}),
				dyn.V("1"),
			},
		},
		{
			name: "sequence",
			input: dyn.V([]dyn.Value{
				dyn.V("1"),
				dyn.V("2"),
			}),
			expectedPaths: []dyn.Path{
				dyn.EmptyPath,
				{dyn.Index(0)},
				{dyn.Index(1)},
			},
			expectedValues: []dyn.Value{
				dyn.V([]dyn.Value{
					dyn.V("1"),
					dyn.V("2"),
				}),
				dyn.V("1"),
				dyn.V("2"),
			},
		},
		{
			name: "nested sequence",
			input: dyn.V([]dyn.Value{
				dyn.V([]dyn.Value{
					dyn.V("1"),
				}),
			}),
			expectedPaths: []dyn.Path{
				dyn.EmptyPath,
				{dyn.Index(0)},
				{dyn.Index(0), dyn.Index(0)},
			},
			expectedValues: []dyn.Value{
				dyn.V([]dyn.Value{
					dyn.V([]dyn.Value{
						dyn.V("1"),
					}),
				}),
				dyn.V([]dyn.Value{
					dyn.V("1"),
				}),
				dyn.V("1"),
			},
		},
		{
			name: "complex structure",
			input: dyn.V(map[string]dyn.Value{
				"a": dyn.V([]dyn.Value{
					dyn.V(map[string]dyn.Value{
						"b": dyn.V("1"),
					}),
				}),
			}),
			expectedPaths: []dyn.Path{
				dyn.EmptyPath,
				{dyn.Key("a")},
				{dyn.Key("a"), dyn.Index(0)},
				{dyn.Key("a"), dyn.Index(0), dyn.Key("b")},
			},
			expectedValues: []dyn.Value{
				dyn.V(map[string]dyn.Value{
					"a": dyn.V([]dyn.Value{
						dyn.V(map[string]dyn.Value{
							"b": dyn.V("1"),
						}),
					}),
				}),
				dyn.V([]dyn.Value{
					dyn.V(map[string]dyn.Value{
						"b": dyn.V("1"),
					}),
				}),
				dyn.V(map[string]dyn.Value{
					"b": dyn.V("1"),
				}),
				dyn.V("1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitedPaths := make([]dyn.Path, 0, len(tt.expectedPaths))
			visitedValues := make([]dyn.Value, 0, len(tt.expectedValues))

			err := dyn.WalkReadOnly(tt.input, func(p dyn.Path, v dyn.Value) error {
				visitedPaths = append(visitedPaths, p)
				visitedValues = append(visitedValues, v)
				return nil
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPaths, visitedPaths)
			assert.Equal(t, tt.expectedValues, visitedValues)
		})
	}
}

func TestWalkReadOnly_Error(t *testing.T) {
	testErr := errors.New("test error")
	input := dyn.V(map[string]dyn.Value{
		"a": dyn.V("1"),
	})

	err := dyn.WalkReadOnly(input, func(p dyn.Path, v dyn.Value) error {
		if p.Equal(dyn.Path{dyn.Key("a")}) {
			return testErr
		}
		return nil
	})

	assert.Equal(t, err, testErr)
}

func TestWalkReadOnly_SkipPaths(t *testing.T) {
	va := dyn.V(dyn.NewMappingFromPairs(
		[]dyn.Pair{
			{Key: dyn.V("b"), Value: dyn.V("1")},
			{Key: dyn.V("c"), Value: dyn.V("2")},
		},
	))

	vd := dyn.V(dyn.NewMappingFromPairs(
		[]dyn.Pair{
			{Key: dyn.V("e"), Value: dyn.V("3")},
		},
	))

	input := dyn.V(dyn.NewMappingFromPairs(
		[]dyn.Pair{
			{
				Key:   dyn.V("a"),
				Value: va,
			},
			{
				Key:   dyn.V("d"),
				Value: vd,
			},
			{
				Key:   dyn.V("f"),
				Value: dyn.V("4"),
			},
		},
	))

	skipPaths := map[string]bool{
		"a.b": true,
		"d":   true,
	}

	var visitedPaths []dyn.Path
	var visitedValues []dyn.Value

	err := dyn.WalkReadOnly(input, func(p dyn.Path, v dyn.Value) error {
		_, ok := skipPaths[p.String()]
		if ok {
			return dyn.ErrSkip
		}

		visitedPaths = append(visitedPaths, p)
		visitedValues = append(visitedValues, v)
		return nil
	})
	assert.NoError(t, err)

	expectedPaths := []dyn.Path{
		dyn.EmptyPath,
		{dyn.Key("a")},
		{dyn.Key("a"), dyn.Key("c")},
		{dyn.Key("f")},
	}
	expectedValues := []dyn.Value{
		input,
		va,
		dyn.V("2"),
		dyn.V("4"),
	}

	assert.Equal(t, expectedPaths, visitedPaths)
	assert.Equal(t, expectedValues, visitedValues)
}
