package dyn

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkRead(t *testing.T) {
	tests := []struct {
		name           string
		input          Value
		expectedPaths  []Path
		expectedValues []Value
	}{
		{
			name: "simple map",
			input: V(map[string]Value{
				"a": V("1"),
				"b": V("2"),
			}),
			expectedPaths: []Path{
				EmptyPath,
				{Key("a")},
				{Key("b")},
			},
			expectedValues: []Value{
				V(map[string]Value{
					"a": V("1"),
					"b": V("2"),
				}),
				V("1"),
				V("2"),
			},
		},
		{
			name: "nested map",
			input: V(map[string]Value{
				"a": V(map[string]Value{
					"b": V("1"),
				}),
			}),
			expectedPaths: []Path{
				EmptyPath,
				{Key("a")},
				{Key("a"), Key("b")},
			},
			expectedValues: []Value{
				V(map[string]Value{
					"a": V(map[string]Value{
						"b": V("1"),
					}),
				}),
				V(map[string]Value{
					"b": V("1"),
				}),
				V("1"),
			},
		},
		{
			name: "sequence",
			input: V([]Value{
				V("1"),
				V("2"),
			}),
			expectedPaths: []Path{
				EmptyPath,
				{Index(0)},
				{Index(1)},
			},
			expectedValues: []Value{
				V([]Value{
					V("1"),
					V("2"),
				}),
				V("1"),
				V("2"),
			},
		},
		{
			name: "nested sequence",
			input: V([]Value{
				V([]Value{
					V("1"),
				}),
			}),
			expectedPaths: []Path{
				EmptyPath,
				{Index(0)},
				{Index(0), Index(0)},
			},
			expectedValues: []Value{
				V([]Value{
					V([]Value{
						V("1"),
					}),
				}),
				V([]Value{
					V("1"),
				}),
				V("1"),
			},
		},
		{
			name: "complex structure",
			input: V(map[string]Value{
				"a": V([]Value{
					V(map[string]Value{
						"b": V("1"),
					}),
				}),
			}),
			expectedPaths: []Path{
				EmptyPath,
				{Key("a")},
				{Key("a"), Index(0)},
				{Key("a"), Index(0), Key("b")},
			},
			expectedValues: []Value{
				V(map[string]Value{
					"a": V([]Value{
						V(map[string]Value{
							"b": V("1"),
						}),
					}),
				}),
				V([]Value{
					V(map[string]Value{
						"b": V("1"),
					}),
				}),
				V(map[string]Value{
					"b": V("1"),
				}),
				V("1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var visited []struct {
				path  Path
				value Value
			}

			err := WalkRead(tt.input, func(p Path, v Value) error {
				visited = append(visited, struct {
					path  Path
					value Value
				}{p, v})
				return nil
			})
			if err != nil {
				t.Errorf("WalkRead() error = %v", err)
			}

			if len(visited) != len(tt.expectedPaths) {
				t.Errorf("WalkRead() visited %d nodes, want %d", len(visited), len(tt.expectedPaths))
			}

			for i, v := range visited {
				if !v.path.Equal(tt.expectedPaths[i]) {
					t.Errorf("WalkRead() path[%d] = %v, want %v", i, v.path, tt.expectedPaths[i])
				}
				if v.value.AsAny() != tt.expectedValues[i].AsAny() {
					t.Errorf("WalkRead() value[%d] = %v, want %v", i, v.value.AsAny(), tt.expectedValues[i].AsAny())
				}
			}
		})
	}
}

func TestWalkReadError(t *testing.T) {
	testErr := errors.New("test error")
	input := V(map[string]Value{
		"a": V("1"),
	})

	err := WalkRead(input, func(p Path, v Value) error {
		if p.Equal(Path{Key("a")}) {
			return testErr
		}
		return nil
	})

	assert.Equal(t, err, testErr)
}

func TestWalkReadSkipPaths(t *testing.T) {
	input := V(map[string]Value{
		"a": V(map[string]Value{
			"b": V("1"),
			"c": V("2"),
		}),
		"d": V(map[string]Value{
			"e": V("3"),
		}),
		"f": V("4"),
	})

	skipPaths := map[string]bool{
		"a.b": true,
		"d":   true,
	}

	var visitedPaths []Path
	var visitedValues []Value

	err := WalkRead(input, func(p Path, v Value) error {
		_, ok := skipPaths[p.String()]
		if ok {
			return ErrSkip
		}

		visitedPaths = append(visitedPaths, p)
		visitedValues = append(visitedValues, v)
		return nil
	})
	require.NoError(t, err)

	expectedPaths := []Path{
		EmptyPath,
		{Key("a")},
		{Key("a"), Key("c")},
		{Key("f")},
	}
	expectedValues := []Value{
		V(map[string]Value{
			"a": V(map[string]Value{
				"b": V("1"),
				"c": V("2"),
			}),
			"d": V(map[string]Value{
				"e": V("3"),
			}),
			"f": V("4"),
		}),
		V(map[string]Value{
			"b": V("1"),
			"c": V("2"),
		}),
		V("2"),
		V("4"),
	}

	assert.Equal(t, visitedPaths, expectedPaths)
	assert.Equal(t, visitedValues, expectedValues)
}
