package mutator

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

type getResourceKeyTestCase struct {
	path string
	key  ResourceKey
	err  bool
}

func TestGetResourceKey(t *testing.T) {
	testCases := []getResourceKeyTestCase{
		{
			path: "resources.jobs.job_1",
			key: ResourceKey{
				Type: "jobs",
				Name: "job_1",
			},
		},
		{
			path: "resources.jobs.job_1.name",
			key: ResourceKey{
				Type: "jobs",
				Name: "job_1",
			},
		},
		{
			path: "resources.jobs",
			err:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			key, err := getResourceKey(dyn.MustPathFromString(tc.path))
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.key, key)
			}
		})
	}
}

func TestResourceKeySet_AddPath(t *testing.T) {
	set := NewResourceKeySet()

	err := set.AddPath(dyn.MustPathFromString("resources.jobs.job_1"))

	require.NoError(t, err)
	require.Equal(t, []ResourceKey{
		{
			Type: "jobs",
			Name: "job_1",
		},
	}, set.ToArray())
}

type resourceKeySetAddTestCase struct {
	name     string
	pattern  dyn.Pattern
	root     dyn.Value
	expected []ResourceKey
}

func TestResourceKeySet_AddPattern(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"jobs": dyn.V(map[string]dyn.Value{
				"job_1": dyn.V(map[string]dyn.Value{
					"name": dyn.V("job_1"),
				}),
				"job_2": dyn.V(map[string]dyn.Value{
					"name": dyn.V("job_1"),
				}),
			}),
		}),
	})

	testCases := []resourceKeySetAddTestCase{
		{
			name:    "one job pattern",
			pattern: dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key("job_1")),
			root:    root,
			expected: []ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:    "all resources pattern",
			pattern: dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
			root:    root,
			expected: []ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
				{
					Type: "jobs",
					Name: "job_2",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			set := NewResourceKeySet()

			err := set.AddPattern(tc.pattern, tc.root)

			require.NoError(t, err)
			require.Equal(t, tc.expected, set.ToArray())
		})
	}
}
