package resourcemutator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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

	// The key of a resource type is the JSON name of its field in Resources.
	expected := NewResourceKeySet()
	for i := range rv.NumField() {
		typ, _, _ := strings.Cut(rv.Type().Field(i).Tag.Get("json"), ",")
		expected.AddResourceKey(ResourceKey{Type: typ, Name: "my_resource"})
	}
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
