package merge

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

type overrideTestCase struct {
	name     string
	left     dyn.Value
	right    dyn.Value
	state    visitorState
	expected dyn.Value
}

func TestOverride_Primitive(t *testing.T) {
	leftLocation := dyn.Location{File: "left.yml", Line: 1, Column: 1}
	rightLocation := dyn.Location{File: "right.yml", Line: 1, Column: 1}

	modifiedTestCases := []overrideTestCase{
		{
			name:     "string (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue("a", leftLocation),
			right:    dyn.NewValue("b", rightLocation),
			expected: dyn.NewValue("b", rightLocation),
		},
		{
			name:     "string (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue("a", leftLocation),
			right:    dyn.NewValue("a", rightLocation),
			expected: dyn.NewValue("a", leftLocation),
		},
		{
			name:     "bool (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(true, leftLocation),
			right:    dyn.NewValue(false, rightLocation),
			expected: dyn.NewValue(false, rightLocation),
		},
		{
			name:     "bool (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(true, leftLocation),
			right:    dyn.NewValue(true, rightLocation),
			expected: dyn.NewValue(true, leftLocation),
		},
		{
			name:     "int (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(1, leftLocation),
			right:    dyn.NewValue(2, rightLocation),
			expected: dyn.NewValue(2, rightLocation),
		},
		{
			name:     "int (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(int32(1), leftLocation),
			right:    dyn.NewValue(int64(1), rightLocation),
			expected: dyn.NewValue(int32(1), leftLocation),
		},
		{
			name:     "float (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(1.0, leftLocation),
			right:    dyn.NewValue(2.0, rightLocation),
			expected: dyn.NewValue(2.0, rightLocation),
		},
		{
			name:     "float (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(float32(1.0), leftLocation),
			right:    dyn.NewValue(float64(1.0), rightLocation),
			expected: dyn.NewValue(float32(1.0), leftLocation),
		},
		{
			name:     "time (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(time.UnixMilli(10000), leftLocation),
			right:    dyn.NewValue(time.UnixMilli(10001), rightLocation),
			expected: dyn.NewValue(time.UnixMilli(10001), rightLocation),
		},
		{
			name:     "time (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(time.UnixMilli(10000), leftLocation),
			right:    dyn.NewValue(time.UnixMilli(10000), rightLocation),
			expected: dyn.NewValue(time.UnixMilli(10000), leftLocation),
		},
		{
			name:     "different types (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue("a", leftLocation),
			right:    dyn.NewValue(42, rightLocation),
			expected: dyn.NewValue(42, rightLocation),
		},
		{
			name: "map - remove 'a', update 'b'",
			state: visitorState{
				removed: []string{"root.a"},
				updated: []string{"root.b"},
			},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, leftLocation),
					"b": dyn.NewValue(10, leftLocation),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(20, rightLocation),
				},
				rightLocation,
			),
			expected: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(20, rightLocation),
				},
				leftLocation,
			),
		},
		{
			name: "map - add 'a'",
			state: visitorState{
				added: []string{"root.a"},
			},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(10, leftLocation),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, rightLocation),
					"b": dyn.NewValue(10, rightLocation),
				},
				leftLocation,
			),
			expected: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, rightLocation),
					// location hasn't changed because value hasn't changed
					"b": dyn.NewValue(10, leftLocation),
				},
				leftLocation,
			),
		},
		{
			name: "map - remove 'a'",
			state: visitorState{
				removed: []string{"root.a"},
			},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, leftLocation),
					"b": dyn.NewValue(10, leftLocation),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(10, rightLocation),
				},
				leftLocation,
			),
			expected: dyn.NewValue(
				map[string]dyn.Value{
					// location hasn't changed because value hasn't changed
					"b": dyn.NewValue(10, leftLocation),
				},
				leftLocation,
			),
		},
		{
			name: "map - add 'jobs.job_1'",
			state: visitorState{
				added: []string{"root.jobs.job_1"},
			},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, leftLocation),
						},
						leftLocation,
					),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, rightLocation),
							"job_1": dyn.NewValue(1337, rightLocation),
						},
						rightLocation,
					),
				},
				rightLocation,
			),
			expected: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, leftLocation),
							"job_1": dyn.NewValue(1337, rightLocation),
						},
						leftLocation,
					),
				},
				leftLocation,
			),
		},
		{
			name:  "map - remove nested key",
			state: visitorState{removed: []string{"root.jobs.job_1"}},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, leftLocation),
							"job_1": dyn.NewValue(1337, rightLocation),
						},
						leftLocation,
					),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, rightLocation),
						},
						rightLocation,
					),
				},
				rightLocation,
			),
			expected: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, leftLocation),
						},
						leftLocation,
					),
				},
				leftLocation,
			),
		},
		{
			name:  "sequence - add",
			state: visitorState{added: []string{"root[1]"}},
			left: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, leftLocation),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, rightLocation),
					dyn.NewValue(10, rightLocation),
				},
				rightLocation,
			),
			expected: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, leftLocation),
					dyn.NewValue(10, rightLocation),
				},
				leftLocation,
			),
		},
		{
			name:  "sequence - remove",
			state: visitorState{removed: []string{"root[1]"}},
			left: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, leftLocation),
					dyn.NewValue(10, leftLocation),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, rightLocation),
				},
				rightLocation,
			),
			expected: dyn.NewValue(
				[]dyn.Value{
					// location hasn't changed because value hasn't changed
					dyn.NewValue(42, leftLocation),
				},
				leftLocation,
			),
		},
		{
			name:  "sequence (not updated)",
			state: visitorState{},
			left: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, leftLocation),
				},
				leftLocation,
			),
			right: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, rightLocation),
				},
				rightLocation,
			),
			expected: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, leftLocation),
				},
				leftLocation,
			),
		},
		{
			name:     "nil (not updated)",
			state:    visitorState{},
			left:     dyn.NilValue,
			right:    dyn.NilValue,
			expected: dyn.NilValue,
		},
		{
			name:     "nil (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NilValue,
			right:    dyn.NewValue(42, rightLocation),
			expected: dyn.NewValue(42, rightLocation),
		},
		{
			name:     "change kind (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(42.0, leftLocation),
			right:    dyn.NewValue(42, rightLocation),
			expected: dyn.NewValue(42, rightLocation),
		},
	}

	for _, tc := range modifiedTestCases {
		t.Run(tc.name, func(t *testing.T) {
			s, visitor := createVisitor(visitorOpts{})
			out, err := override(dyn.NewPath(dyn.Key("root")), tc.left, tc.right, visitor)

			assert.NoError(t, err)
			assert.Equal(t, tc.state, *s)
			assert.Equal(t, tc.expected, out)
		})

		modified := len(tc.state.removed)+len(tc.state.added)+len(tc.state.updated) > 0

		// visitor is not used unless there is a change

		if modified {
			t.Run(tc.name+" - visitor has error", func(t *testing.T) {
				_, visitor := createVisitor(visitorOpts{error: fmt.Errorf("unexpected change in test")})
				_, err := override(dyn.EmptyPath, tc.left, tc.right, visitor)

				assert.EqualError(t, err, "unexpected change in test")
			})

			t.Run(tc.name+" - visitor overrides value", func(t *testing.T) {
				expected := dyn.NewValue("return value", dyn.Location{})
				s, visitor := createVisitor(visitorOpts{returnValue: &expected})
				out, err := override(dyn.EmptyPath, tc.left, tc.right, visitor)

				assert.NoError(t, err)

				for _, added := range s.added {
					actual, err := dyn.GetByPath(out, dyn.MustPathFromString(added))

					assert.NoError(t, err)
					assert.Equal(t, expected, actual)
				}

				for _, updated := range s.updated {
					actual, err := dyn.GetByPath(out, dyn.MustPathFromString(updated))

					assert.NoError(t, err)
					assert.Equal(t, expected, actual)
				}
			})

			if len(tc.state.removed) > 0 {
				t.Run(tc.name+" - visitor can undo delete", func(t *testing.T) {
					s, visitor := createVisitor(visitorOpts{deleteError: ErrOverrideUndoDelete})
					out, err := override(dyn.EmptyPath, tc.left, tc.right, visitor)
					require.NoError(t, err)

					for _, removed := range s.removed {
						expected, err := dyn.GetByPath(tc.left, dyn.MustPathFromString(removed))
						require.NoError(t, err)

						actual, err := dyn.GetByPath(out, dyn.MustPathFromString(removed))

						assert.NoError(t, err)
						assert.Equal(t, expected, actual)
					}
				})
			}
		}
	}
}

func TestOverride_PreserveMappingKeys(t *testing.T) {
	leftLocation := dyn.Location{File: "left.yml", Line: 1, Column: 1}
	leftKeyLocation := dyn.Location{File: "left.yml", Line: 2, Column: 1}
	leftValueLocation := dyn.Location{File: "left.yml", Line: 3, Column: 1}

	rightLocation := dyn.Location{File: "right.yml", Line: 1, Column: 1}
	rightKeyLocation := dyn.Location{File: "right.yml", Line: 2, Column: 1}
	rightValueLocation := dyn.Location{File: "right.yml", Line: 3, Column: 1}

	left := dyn.NewMapping()
	left.Set(dyn.NewValue("a", leftKeyLocation), dyn.NewValue(42, leftValueLocation))

	right := dyn.NewMapping()
	right.Set(dyn.NewValue("a", rightKeyLocation), dyn.NewValue(7, rightValueLocation))

	state, visitor := createVisitor(visitorOpts{})

	out, err := override(
		dyn.EmptyPath,
		dyn.NewValue(left, leftLocation),
		dyn.NewValue(right, rightLocation),
		visitor,
	)

	assert.NoError(t, err)

	if err != nil {
		outPairs := out.MustMap().Pairs()

		assert.Equal(t, visitorState{updated: []string{"a"}}, state)
		assert.Equal(t, 1, len(outPairs))

		// mapping was first defined in left, so it should keep its location
		assert.Equal(t, leftLocation, out.Location())

		// if there is a validation error for key value, it should point
		// to where it was initially defined
		assert.Equal(t, leftKeyLocation, outPairs[0].Key.Location())

		// the value should have updated location, because it has changed
		assert.Equal(t, rightValueLocation, outPairs[0].Value.Location())
	}
}

type visitorState struct {
	added   []string
	removed []string
	updated []string
}

type visitorOpts struct {
	error       error
	deleteError error
	returnValue *dyn.Value
}

func createVisitor(opts visitorOpts) (*visitorState, OverrideVisitor) {
	s := visitorState{}

	return &s, OverrideVisitor{
		VisitUpdate: func(valuePath dyn.Path, left dyn.Value, right dyn.Value) (dyn.Value, error) {
			s.updated = append(s.updated, valuePath.String())

			if opts.error != nil {
				return dyn.NilValue, opts.error
			} else if opts.returnValue != nil {
				return *opts.returnValue, nil
			} else {
				return right, nil
			}
		},
		VisitDelete: func(valuePath dyn.Path, left dyn.Value) error {
			s.removed = append(s.removed, valuePath.String())

			if opts.error != nil {
				return opts.error
			} else if opts.deleteError != nil {
				return opts.deleteError
			} else {
				return nil
			}
		},
		VisitInsert: func(valuePath dyn.Path, right dyn.Value) (dyn.Value, error) {
			s.added = append(s.added, valuePath.String())

			if opts.error != nil {
				return dyn.NilValue, opts.error
			} else if opts.returnValue != nil {
				return *opts.returnValue, nil
			} else {
				return right, nil
			}
		},
	}
}
