package resourcemutator

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {},
					"job_2": {},
				},
			},
		},
	}

	set, err := getAllResources(b)
	require.NoError(t, err)

	assert.ElementsMatch(
		t,
		set.ToArray(),
		[]ResourceKey{
			{Type: "jobs", Name: "job_1"},
			{Type: "jobs", Name: "job_2"},
		},
	)
}

// TestGetAllResourcesMatchesConfigKeys checks that every resource type is collected
// under the same key it has in the configuration (resources.<type>.<name>).
func TestGetAllResourcesMatchesConfigKeys(t *testing.T) {
	b := &bundle.Bundle{}

	// Add one resource of every type.
	rv := reflect.ValueOf(&b.Config.Resources).Elem()
	for i := range rv.NumField() {
		field := rv.Field(i)
		require.Equal(t, reflect.Map, field.Kind(), rv.Type().Field(i).Name)
		m := reflect.MakeMap(field.Type())
		m.SetMapIndex(reflect.ValueOf("my_resource"), reflect.New(field.Type().Elem().Elem()))
		field.Set(m)
	}

	root, err := convert.FromTyped(b.Config, dyn.NilValue)
	require.NoError(t, err)
	expected := NewResourceKeySet()
	err = expected.AddPattern(dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()), root)
	require.NoError(t, err)
	require.Equal(t, rv.NumField(), expected.Size())

	set, err := getAllResources(b)
	require.NoError(t, err)
	assert.Equal(t, expected, set)
}

func TestGetAllResourcesRejectsEmptyName(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{"": {}},
			},
		},
	}

	_, err := getAllResources(b)
	assert.EqualError(t, err, "failed to collect resources: can't parse resource key")
}
