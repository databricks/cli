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
			key, err := GetResourceKey(dyn.MustPathFromString(tc.path))
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.key, key)
			}
		})
	}
}

type resourceKeySetAddTestCase struct {
	path     string
	value    dyn.Value
	expected []ResourceKey
}

func TestResourceKeySet_add(t *testing.T) {
	testCases := []resourceKeySetAddTestCase{
		{
			path: "resources.jobs.job_1",
			value: dyn.V(map[string]any{
				"name": dyn.V("job_1"),
			}),
			expected: []ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			path: "resources",
			value: dyn.V(map[string]any{
				"jobs": dyn.V(
					map[string]any{
						"job_1": dyn.V(
							map[string]any{
								"name": dyn.V("job_1"),
							}),
						"job_2": dyn.V(
							map[string]any{
								"name": dyn.V("job_1"),
							}),
					}),
			}),
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
		t.Run(tc.path, func(t *testing.T) {
			set := NewResourceKeySet()
			err := set.Add(dyn.MustPathFromString(tc.path), tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.expected, set.ToArray())
		})
	}
}
