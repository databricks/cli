package merge

import (
	"errors"
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
			left:     dyn.NewValue("a", []dyn.Location{leftLocation}),
			right:    dyn.NewValue("b", []dyn.Location{rightLocation}),
			expected: dyn.NewValue("b", []dyn.Location{rightLocation}),
		},
		{
			name:     "string (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue("a", []dyn.Location{leftLocation}),
			right:    dyn.NewValue("a", []dyn.Location{rightLocation}),
			expected: dyn.NewValue("a", []dyn.Location{leftLocation}),
		},
		{
			name:     "bool (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(true, []dyn.Location{leftLocation}),
			right:    dyn.NewValue(false, []dyn.Location{rightLocation}),
			expected: dyn.NewValue(false, []dyn.Location{rightLocation}),
		},
		{
			name:     "bool (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(true, []dyn.Location{leftLocation}),
			right:    dyn.NewValue(true, []dyn.Location{rightLocation}),
			expected: dyn.NewValue(true, []dyn.Location{leftLocation}),
		},
		{
			name:     "int (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(1, []dyn.Location{leftLocation}),
			right:    dyn.NewValue(2, []dyn.Location{rightLocation}),
			expected: dyn.NewValue(2, []dyn.Location{rightLocation}),
		},
		{
			name:     "int (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(int32(1), []dyn.Location{leftLocation}),
			right:    dyn.NewValue(int64(1), []dyn.Location{rightLocation}),
			expected: dyn.NewValue(int32(1), []dyn.Location{leftLocation}),
		},
		{
			name:     "float (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(1.0, []dyn.Location{leftLocation}),
			right:    dyn.NewValue(2.0, []dyn.Location{rightLocation}),
			expected: dyn.NewValue(2.0, []dyn.Location{rightLocation}),
		},
		{
			name:     "float (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(float32(1.0), []dyn.Location{leftLocation}),
			right:    dyn.NewValue(float64(1.0), []dyn.Location{rightLocation}),
			expected: dyn.NewValue(float32(1.0), []dyn.Location{leftLocation}),
		},
		{
			name:     "time (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(dyn.FromTime(time.UnixMilli(10000)), []dyn.Location{leftLocation}),
			right:    dyn.NewValue(dyn.FromTime(time.UnixMilli(10001)), []dyn.Location{rightLocation}),
			expected: dyn.NewValue(dyn.FromTime(time.UnixMilli(10001)), []dyn.Location{rightLocation}),
		},
		{
			name:     "time (not updated)",
			state:    visitorState{},
			left:     dyn.NewValue(dyn.FromTime(time.UnixMilli(10000)), []dyn.Location{leftLocation}),
			right:    dyn.NewValue(dyn.FromTime(time.UnixMilli(10000)), []dyn.Location{rightLocation}),
			expected: dyn.NewValue(dyn.FromTime(time.UnixMilli(10000)), []dyn.Location{leftLocation}),
		},
		{
			name:     "different types (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue("a", []dyn.Location{leftLocation}),
			right:    dyn.NewValue(42, []dyn.Location{rightLocation}),
			expected: dyn.NewValue(42, []dyn.Location{rightLocation}),
		},
		{
			name: "map - remove 'a', update 'b'",
			state: visitorState{
				removed: []string{"root.a"},
				updated: []string{"root.b"},
			},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, []dyn.Location{leftLocation}),
					"b": dyn.NewValue(10, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation}),

			right: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(20, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{rightLocation}),

			expected: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(20, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{leftLocation}),
		},
		{
			name: "map - add 'a'",
			state: visitorState{
				added: []string{"root.a"},
			},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(10, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),

			right: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, []dyn.Location{rightLocation}),
					"b": dyn.NewValue(10, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{leftLocation},
			),

			expected: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, []dyn.Location{rightLocation}),
					// location hasn't changed because value hasn't changed
					"b": dyn.NewValue(10, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),
		},
		{
			name: "map - remove 'a'",
			state: visitorState{
				removed: []string{"root.a"},
			},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"a": dyn.NewValue(42, []dyn.Location{leftLocation}),
					"b": dyn.NewValue(10, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),

			right: dyn.NewValue(
				map[string]dyn.Value{
					"b": dyn.NewValue(10, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{leftLocation},
			),

			expected: dyn.NewValue(
				map[string]dyn.Value{
					// location hasn't changed because value hasn't changed
					"b": dyn.NewValue(10, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
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
							"job_0": dyn.NewValue(42, []dyn.Location{leftLocation}),
						},
						[]dyn.Location{leftLocation},
					),
				},
				[]dyn.Location{leftLocation},
			),

			right: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, []dyn.Location{rightLocation}),
							"job_1": dyn.NewValue(1337, []dyn.Location{rightLocation}),
						},
						[]dyn.Location{rightLocation},
					),
				},
				[]dyn.Location{rightLocation},
			),

			expected: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, []dyn.Location{leftLocation}),
							"job_1": dyn.NewValue(1337, []dyn.Location{rightLocation}),
						},
						[]dyn.Location{leftLocation},
					),
				},
				[]dyn.Location{leftLocation},
			),
		},
		{
			name:  "map - remove nested key",
			state: visitorState{removed: []string{"root.jobs.job_1"}},
			left: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, []dyn.Location{leftLocation}),
							"job_1": dyn.NewValue(1337, []dyn.Location{rightLocation}),
						},
						[]dyn.Location{leftLocation},
					),
				},
				[]dyn.Location{leftLocation},
			),
			right: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, []dyn.Location{rightLocation}),
						},
						[]dyn.Location{rightLocation},
					),
				},
				[]dyn.Location{rightLocation},
			),
			expected: dyn.NewValue(
				map[string]dyn.Value{
					"jobs": dyn.NewValue(
						map[string]dyn.Value{
							"job_0": dyn.NewValue(42, []dyn.Location{leftLocation}),
						},
						[]dyn.Location{leftLocation},
					),
				},
				[]dyn.Location{leftLocation},
			),
		},
		{
			name:  "sequence - add",
			state: visitorState{added: []string{"root[1]"}},
			left: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),
			right: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{rightLocation}),
					dyn.NewValue(10, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{rightLocation},
			),
			expected: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{leftLocation}),
					dyn.NewValue(10, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{leftLocation},
			),
		},
		{
			name:  "sequence - remove",
			state: visitorState{removed: []string{"root[1]"}},
			left: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{leftLocation}),
					dyn.NewValue(10, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),
			right: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{rightLocation},
			),
			expected: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),
			// location hasn't changed because value hasn't changed
		},
		{
			name:  "sequence (not updated)",
			state: visitorState{},
			left: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),
			right: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{rightLocation}),
				},
				[]dyn.Location{rightLocation},
			),
			expected: dyn.NewValue(
				[]dyn.Value{
					dyn.NewValue(42, []dyn.Location{leftLocation}),
				},
				[]dyn.Location{leftLocation},
			),
		},
		{
			name:     "nil (not updated)",
			state:    visitorState{},
			left:     dyn.NilValue.WithLocations([]dyn.Location{leftLocation}),
			right:    dyn.NilValue.WithLocations([]dyn.Location{rightLocation}),
			expected: dyn.NilValue.WithLocations([]dyn.Location{leftLocation}),
		},
		{
			name:     "nil (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NilValue,
			right:    dyn.NewValue(42, []dyn.Location{rightLocation}),
			expected: dyn.NewValue(42, []dyn.Location{rightLocation}),
		},
		{
			name:     "change kind (updated)",
			state:    visitorState{updated: []string{"root"}},
			left:     dyn.NewValue(42.0, []dyn.Location{leftLocation}),
			right:    dyn.NewValue(42, []dyn.Location{rightLocation}),
			expected: dyn.NewValue(42, []dyn.Location{rightLocation}),
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
				_, visitor := createVisitor(visitorOpts{error: errors.New("unexpected change in test")})
				_, err := override(dyn.EmptyPath, tc.left, tc.right, visitor)

				assert.EqualError(t, err, "unexpected change in test")
			})

			t.Run(tc.name+" - visitor overrides value", func(t *testing.T) {
				expected := dyn.V("return value")
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
	left.SetLoc("a", []dyn.Location{leftKeyLocation}, dyn.NewValue(42, []dyn.Location{leftValueLocation}))

	right := dyn.NewMapping()
	right.SetLoc("a", []dyn.Location{rightKeyLocation}, dyn.NewValue(7, []dyn.Location{rightValueLocation}))

	state, visitor := createVisitor(visitorOpts{})

	out, err := override(
		dyn.EmptyPath,
		dyn.NewValue(left, []dyn.Location{leftLocation}),
		dyn.NewValue(right, []dyn.Location{rightLocation}),
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
		VisitUpdate: func(valuePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
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
